// Package migration は移行リハーサルのテスト.
//
//	E2E_MIGRATION_PHASE=seed   : 移行元 (PHP 版) に中間状態を作り, 状態を E2E_MIGRATION_STATE に書き出す
//	E2E_MIGRATION_PHASE=verify : 移行先に対して, 書き出した状態が引き継がれていることを確認する
//
// 通常は e2e/migration/rehearse.sh から実行する.
package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/KLabTools/emlauncher/e2e/internal/harness"
)

type State struct {
	Users    map[string]UserState `json:"users"` // owner, member
	Apps     map[string]AppState  `json:"apps"`  // main, sub
	Packages map[string]PkgState  `json:"packages"`

	// PHP 版が memcached に書いたインストールトークン (API / 画面で発行)
	InstallTokenAPI string `json:"install_token_api"`
	InstallTokenUI  string `json:"install_token_ui"`

	GuestPassActive  string `json:"guest_pass_active"`
	GuestPassExpired string `json:"guest_pass_expired"`

	CommentCount int    `json:"comment_count"`
	OrphanTemp   string `json:"orphan_temp"` // upload_post されなかった一時ファイルのキー

	// 移行手順で DB とストレージが変わっていないことの確認用
	DB      map[string][]harness.Row `json:"db"`
	Storage map[string]string        `json:"storage"` // key -> sha256

	// 移行前の API の応答 (package_list)
	PackageList map[string][]PackageSummary `json:"package_list"`
}

type UserState struct {
	Mail     string `json:"mail"`
	Password string `json:"password"`
}

type AppState struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	APIKey string `json:"api_key"`
}

type PkgState struct {
	ID       int64  `json:"id"`
	App      string `json:"app"`
	Title    string `json:"title"`
	FileName string `json:"file_name"` // original_file_name
	Key      string `json:"key"`
	SHA256   string `json:"sha256"`
	// Android 端末でインストールしたときに落ちてくるファイル (AAB なら生成した APK)
	AndroidKey string `json:"android_key"`
}

// PackageSummary は package_list の応答のうち, 移行で変わってはいけない部分.
type PackageSummary struct {
	ID            string   `json:"id"`
	Platform      string   `json:"platform"`
	Title         string   `json:"title"`
	Description   any      `json:"description"`
	Identifier    any      `json:"identifier"`
	OriginalName  string   `json:"original_file_name"`
	FileSize      string   `json:"file_size"`
	Protect       bool     `json:"protect"`
	Created       string   `json:"created"`
	Tags          []string `json:"tags"`
	InstallCount  string   `json:"install_count"`
	AttachedFiles []string `json:"attached_files"` // id:original_file_name:file_size
}

var tables = map[string]string{
	"user_pass":         "mail",
	"application":       "id",
	"application_owner": "id",
	"tag":               "id",
	"package":           "id",
	"package_tag":       "package_id, tag_id",
	"attached_file":     "id",
	"comment":           "id",
	"install_log":       "id",
	"app_install_user":  "mail, app_id",
	"guest_pass":        "id",
	"guestpass_log":     "id",
}

func phase(t *testing.T, want string) *harness.Env {
	t.Helper()
	if os.Getenv("E2E_MIGRATION_PHASE") != want {
		t.Skipf("set E2E_MIGRATION_PHASE=%s to run", want)
	}
	return harness.New(t)
}

func statePath() string {
	if p := os.Getenv("E2E_MIGRATION_STATE"); p != "" {
		return p
	}
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", ".work", "migration-state.json")
}

func saveState(t *testing.T, s *State) {
	t.Helper()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath(), b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("state saved to %s", statePath())
}

func loadState(t *testing.T) *State {
	t.Helper()
	b, err := os.ReadFile(statePath())
	if err != nil {
		t.Fatalf("load state (run seed first): %v", err)
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return &s
}

func snapshotDB(e *harness.Env) map[string][]harness.Row {
	out := map[string][]harness.Row{}
	for table, order := range tables {
		out[table] = e.DB.Rows("SELECT * FROM `" + table + "` ORDER BY " + order)
	}
	return out
}

func snapshotStorage(e *harness.Env) map[string]string {
	out := map[string]string{}
	for _, k := range e.Storage.Keys("") {
		out[k] = sha(e.Storage.Read(k))
	}
	return out
}

func sha(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// packageList は package_list API の応答を PackageSummary に揃える.
func packageList(t *testing.T, e *harness.Env, apiKey string) []PackageSummary {
	t.Helper()
	res := e.APICall("package_list", apiKey, map[string][]string{"limit": {"100"}})
	harness.ExpectStatus(t, res, 200)
	var out []PackageSummary
	for _, v := range res.JSON(t).([]any) {
		m := v.(map[string]any)
		s := PackageSummary{
			ID:           str(m["id"]),
			Platform:     str(m["platform"]),
			Title:        str(m["title"]),
			Description:  m["description"],
			Identifier:   m["identifier"],
			OriginalName: str(m["original_file_name"]),
			FileSize:     str(m["file_size"]),
			Protect:      m["protect"].(bool),
			Created:      str(m["created"]),
			InstallCount: str(m["install_count"]),
			Tags:         []string{},
		}
		for _, tag := range m["tags"].([]any) {
			s.Tags = append(s.Tags, str(tag))
		}
		for _, a := range m["attached_files"].([]any) {
			am := a.(map[string]any)
			s.AttachedFiles = append(s.AttachedFiles, str(am["id"])+":"+str(am["original_file_name"])+":"+str(am["file_size"]))
		}
		out = append(out, s)
	}
	return out
}

// str は数値 (json.Number) と文字列を同じ表現にする.
func str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	}
	b, _ := json.Marshal(v)
	return string(b)
}
