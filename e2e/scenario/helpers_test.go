package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"emlauncher-e2e/internal/harness"
)

var (
	reRand16  = regexp.MustCompile(`^[0-9A-Za-z]{16}$`)
	reToken32 = regexp.MustCompile(`^[0-9A-Za-z]{32}$`)
)

// mimeStored はパッケージ・添付ファイル・一時ファイルを S3 に保存したときの ContentType.
// PHP 版は MIME を指定しているつもりだが, MultipartUploader が設定を読まないため
// (アップロード元の一時ファイルに拡張子が無く) 実際にはすべて application/octet-stream になる.
// S3 の構成を変えないため, これを仕様とする.
const mimeStored = "application/octet-stream"

func expectEq[T comparable](t testing.TB, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", what, got, want)
	}
}

func expectMatch(t testing.TB, what string, re *regexp.Regexp, s string) {
	t.Helper()
	if !re.MatchString(s) {
		t.Fatalf("%s = %q, want match %s", what, s, re)
	}
}

func expectContains(t testing.TB, what, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("%s does not contain %q:\n%s", what, sub, truncate(s, 3000))
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// jsonStr は API レスポンスの値が JSON 文字列であることを確認して返す.
// PHP 版は PDO::ATTR_STRINGIFY_FETCHES により DB 由来の数値も文字列で返す.
func jsonStr(t testing.TB, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("json has no %q: %v", key, m)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("json %q = %#v (%T), want string", key, v, v)
	}
	return s
}

// jsonNum は数値フィールドの値を文字列で返す. API ドキュメント (Caveats) のとおり,
// 数値フィールドは JSON の文字列でも数値でもよい (PHP 版はエンドポイントによって混在している).
func jsonNum(t testing.TB, m map[string]any, key string) string {
	t.Helper()
	switch v := m[key].(type) {
	case json.Number:
		return v.String()
	case string:
		return v
	}
	t.Fatalf("json %q = %#v (%T), want number or numeric string", key, m[key], m[key])
	return ""
}

func jsonBool(t testing.TB, m map[string]any, key string) bool {
	t.Helper()
	b, ok := m[key].(bool)
	if !ok {
		t.Fatalf("json %q = %#v (%T), want bool", key, m[key], m[key])
	}
	return b
}

func jsonList(t testing.TB, m map[string]any, key string) []any {
	t.Helper()
	l, ok := m[key].([]any)
	if !ok {
		t.Fatalf("json %q = %#v (%T), want array", key, m[key], m[key])
	}
	return l
}

func jsonKeys(m map[string]any) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkPackageJSON は API が返す package の JSON を DB の行と突き合わせる.
func checkPackageJSON(t testing.TB, e *harness.Env, m map[string]any, pkg harness.Row) {
	t.Helper()
	wantKeys := []string{"application_url", "attached_files", "created", "description", "file_size", "id",
		"identifier", "install_count", "original_file_name", "package_url", "platform", "protect", "tags", "title"}
	if got := jsonKeys(m); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("package json keys = %v, want %v", got, wantKeys)
	}
	base := e.Config.BaseURL
	expectEq(t, "package_url", jsonStr(t, m, "package_url"), base+"/package?id="+pkg.Str("id"))
	expectEq(t, "application_url", jsonStr(t, m, "application_url"), base+"/app?id="+pkg.Str("app_id"))
	expectEq(t, "id", jsonNum(t, m, "id"), pkg.Str("id"))
	expectEq(t, "platform", jsonStr(t, m, "platform"), pkg.Str("platform"))
	expectEq(t, "title", jsonStr(t, m, "title"), pkg.Str("title"))
	if pkg.Null("description") {
		expectEq(t, "description", m["description"], any(nil))
	} else {
		expectEq(t, "description", jsonStr(t, m, "description"), pkg.Str("description"))
	}
	if pkg.Null("identifier") {
		expectEq(t, "identifier", m["identifier"], any(nil))
	} else {
		expectEq(t, "identifier", jsonStr(t, m, "identifier"), pkg.Str("identifier"))
	}
	expectEq(t, "original_file_name", jsonStr(t, m, "original_file_name"), pkg.Str("original_file_name"))
	expectEq(t, "file_size", jsonNum(t, m, "file_size"), pkg.Str("file_size"))
	expectEq(t, "protect", jsonBool(t, m, "protect"), pkg.Str("protect") == "1")
	expectEq(t, "created", jsonStr(t, m, "created"), pkg.Str("created"))
	jsonNum(t, m, "install_count")

	var tags []string
	for _, v := range jsonList(t, m, "tags") {
		tags = append(tags, v.(string))
	}
	var dbTags []string
	for _, r := range e.DB.Rows("SELECT t.name FROM package_tag j JOIN tag t ON t.id = j.tag_id WHERE j.package_id = ?", pkg.Str("id")) {
		dbTags = append(dbTags, r.Str("name"))
	}
	sort.Strings(tags)
	sort.Strings(dbTags)
	if !reflect.DeepEqual(tags, dbTags) {
		t.Fatalf("tags = %v, db = %v", tags, dbTags)
	}

	afs := jsonList(t, m, "attached_files")
	rows := e.DB.Rows("SELECT * FROM attached_file WHERE package_id = ? ORDER BY id", pkg.Str("id"))
	if len(afs) != len(rows) {
		t.Fatalf("attached_files = %d, db = %d", len(afs), len(rows))
	}
	for i, r := range rows {
		checkAttachedJSON(t, afs[i].(map[string]any), r)
	}
}

func checkAttachedJSON(t testing.TB, m map[string]any, af harness.Row) {
	t.Helper()
	if got, want := jsonKeys(m), []string{"created", "file_size", "id", "original_file_name"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("attached json keys = %v, want %v", got, want)
	}
	expectEq(t, "attached.id", jsonNum(t, m, "id"), af.Str("id"))
	expectEq(t, "attached.original_file_name", jsonStr(t, m, "original_file_name"), af.Str("original_file_name"))
	expectEq(t, "attached.file_size", jsonNum(t, m, "file_size"), af.Str("file_size"))
	expectEq(t, "attached.created", jsonStr(t, m, "created"), af.Str("created"))
}

// checkStored はストレージにキーが存在し, 中身と (S3 なら) ContentType・ACL が期待どおりか確認する.
func checkStored(t testing.TB, e *harness.Env, key string, data []byte, contentType string, public bool) {
	t.Helper()
	if !e.Storage.Exists(key) {
		t.Fatalf("storage key %s does not exist (keys: %v)", key, e.Storage.Keys(key[:strings.LastIndex(key, "/")+1]))
	}
	if data != nil && !bytes.Equal(e.Storage.Read(key), data) {
		t.Fatalf("storage %s: content mismatch", key)
	}
	if ct, pub, ok := e.Storage.Meta(key); ok {
		if contentType != "" {
			expectEq(t, key+" ContentType", ct, contentType)
		}
		expectEq(t, key+" public-read", pub, public)
	}
}

func checkNotStored(t testing.TB, e *harness.Env, key string) {
	t.Helper()
	if e.Storage.Exists(key) {
		t.Fatalf("storage key %s still exists", key)
	}
}

// pngSize は PNG のサイズを返す.
func pngSize(t testing.TB, data []byte) (int, int) {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not png: %v", err)
	}
	return cfg.Width, cfg.Height
}

func form(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	return v
}

func itoa(n int64) string { return fmt.Sprint(n) }

// checkPackageJSONWithoutDB は削除済みなど DB に行が無い package の JSON を, 事前に取得した行と比べる.
func checkPackageJSONWithoutDB(t testing.TB, e *harness.Env, m map[string]any, pkg harness.Row) {
	t.Helper()
	expectEq(t, "id", jsonNum(t, m, "id"), pkg.Str("id"))
	expectEq(t, "package_url", jsonStr(t, m, "package_url"), e.Config.BaseURL+"/package?id="+pkg.Str("id"))
	expectEq(t, "platform", jsonStr(t, m, "platform"), pkg.Str("platform"))
	expectEq(t, "title", jsonStr(t, m, "title"), pkg.Str("title"))
	expectEq(t, "original_file_name", jsonStr(t, m, "original_file_name"), pkg.Str("original_file_name"))
	expectEq(t, "created", jsonStr(t, m, "created"), pkg.Str("created"))
}

// expectErrorPage はエラーページ (本文に message を含む) が status で返ることを確認する.
func expectErrorPage(t testing.TB, e *harness.Env, res *harness.Response, status int, message string) {
	t.Helper()
	e.ExpectErrorPage(res, status, message)
}
