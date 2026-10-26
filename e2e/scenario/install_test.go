package scenario

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	e2e "emlauncher-e2e"
)

// expectDownload は URL がキーを指し, 中身が data と一致することを確認する.
func expectDownload(t testing.TB, e *e2e.Env, rawURL, key string, data []byte, fileName string) {
	t.Helper()
	expectEq(t, "download key", e.KeyFromURL(rawURL), key)
	res := e.Download(rawURL)
	e2e.ExpectStatus(t, res, http.StatusOK)
	if !bytes.Equal(res.Body, data) {
		t.Fatalf("downloaded content mismatch (%d bytes, want %d)", len(res.Body), len(data))
	}
	if e.Storage.Kind() == "s3" {
		// 署名付き URL は 60 分有効で, 元のファイル名でダウンロードさせる
		expectContains(t, "url", rawURL, "X-Amz-Expires=3600")
		want := "filename*=UTF-8''" + rfc5987(fileName)
		if fileName != url.QueryEscape(fileName) && e.PHPBug("ダウンロード時のファイル名を urlencode() でエンコードしており, 空白が + になる") {
			want = "filename*=UTF-8''" + url.QueryEscape(fileName)
		}
		expectContains(t, "Content-Disposition", res.Header.Get("Content-Disposition"), want)
	}
}

func TestInstallAndroid(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	apk := e2e.APK("com.example.install")
	id := e.MustAPIUpload(app.APIKey, "apk", "my app.apk", apk, nil)
	pkg := e.Package(id)

	user := e.NewUser("user")
	user.UserAgent = e2e.UAAndroid
	res := user.Get("/package/install?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.PackageKey(pkg), apk, "my app.apk")

	log := e.DB.Row("SELECT * FROM install_log WHERE package_id = ?", id)
	expectEq(t, "app_id", log.Int("app_id"), app.ID)
	expectEq(t, "mail", log.Str("mail"), user.Mail)
	expectEq(t, "user_agent", log.Str("user_agent"), e2e.UAAndroid)
	e.ExpectRecent("installed", log.Str("installed"))

	iu := e.DB.Row("SELECT * FROM app_install_user WHERE app_id = ? AND mail = ?", app.ID, user.Mail)
	expectEq(t, "notify", iu.Str("notify"), "1")
	expectEq(t, "last_installed", iu.Str("last_installed"), log.Str("installed"))

	// 2 回目
	e2e.ExpectStatus(t, user.Get("/package/install?id="+itoa(id)), http.StatusFound)
	expectEq(t, "install_log rows", e.DB.Count("SELECT count(*) FROM install_log WHERE package_id = ?", id), 2)
	expectEq(t, "app_install_user rows", e.DB.Count("SELECT count(*) FROM app_install_user WHERE app_id = ?", app.ID), 1)

	// install_count, インストール済み表示
	res = e.APICall("package_list", app.APIKey, nil)
	expectEq(t, "install_count", jsonNum(t, res.JSON(t).([]any)[0].(map[string]any), "install_count"), "1")
	expectContains(t, "installed apps", user.Get("/myapps/installed").String(), app.Title)
}

func TestInstallIOS(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	ipa := e2e.IPA("Sample", "com.example.install.ios")
	id := e.MustAPIUpload(app.APIKey, "ios build", "sample.ipa", ipa, nil)
	pkg := e.Package(id)

	user := e.NewUser("user")
	user.UserAgent = e2e.UAiOS
	res := user.Get("/package/install?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusFound)
	loc := res.Location()
	const itms = "itms-services://?action=download-manifest&url="
	if !strings.HasPrefix(loc, itms) {
		t.Fatalf("location = %q, want itms-services", loc)
	}
	plistURL, err := url.QueryUnescape(strings.TrimPrefix(loc, itms))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plistURL, e.Config.BaseURL+"/package/install_plist?") {
		t.Fatalf("plist url = %q", plistURL)
	}
	expectEq(t, "plist id", e2e.QueryInt(t, plistURL, "id"), id)
	u, _ := url.Parse(plistURL)
	tok := u.Query().Get("t")
	expectMatch(t, "plist token", reToken32, tok)
	expectEq(t, "memcached plist token", string(e.MemcacheGet("ios_plist_token_"+tok)), itoa(id))

	// iOS のインストーラはセッションを持たない
	res = e.NewClient().Get(plistURL)
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectContains(t, "Content-Type", res.Header.Get("Content-Type"), "text/xml")
	body := res.String()
	expectContains(t, "plist", body, "<string>com.example.install.ios</string>")
	expectContains(t, "plist", body, "ios build")
	m := regexp.MustCompile(`<string>([^<]*` + regexp.QuoteMeta(pkg.Str("file_name")) + `[^<]*)</string>`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("ipa url not found in plist:\n%s", body)
	}
	expectDownload(t, e, htmlUnescape(m[1]), e2e.PackageKey(pkg), ipa, "sample.ipa")

	expectErrorPage(t, e, e.NewClient().Get("/package/install_plist?id="+itoa(id)+"&t=invalid"), http.StatusForbidden, "Permission Denied")

	// PC からは ipa を直接ダウンロード
	pc := e.NewUser("pc")
	res = pc.Get("/package/install?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.PackageKey(pkg), ipa, "sample.ipa")
}

func TestInstallAABOnAndroid(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	aab := e2e.AAB("com.example.install.aab")
	id := e.MustAPIUpload(app.APIKey, "aab", "bundle.aab", aab, nil)
	pkg := e.Package(id)
	apk := e.DB.Row("SELECT * FROM attached_file WHERE package_id = ? AND file_type = 'apk'", id)

	user := e.NewUser("user")
	user.UserAgent = e2e.UAAndroid
	res := user.Get("/package/install?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.AttachedKey(apk), e.Storage.Read(e2e.AttachedKey(apk)), "bundle.apk")

	user.UserAgent = e2e.UAPC
	res = user.Get("/package/install?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.PackageKey(pkg), aab, "bundle.aab")
}

var reInstallTokenURL = regexp.MustCompile(`https?://[^"<\s]+/package/install\?token=([0-9A-Za-z]{32})`)

func TestInstallTokenFromUI(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	apk := e2e.APK("com.example.token")
	id := e.MustAPIUpload(app.APIKey, "apk", "a.apk", apk, nil)

	// オーナー以外でもトークンを作れる (作った人の名前でインストールされる)
	user := e.NewUser("user")
	res := user.Get("/package/create_token?id=" + itoa(id))
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := reInstallTokenURL.FindStringSubmatch(res.String())
	if m == nil {
		t.Fatalf("token url not found:\n%s", res.Body)
	}

	c := e.NewClient()
	c.UserAgent = e2e.UAAndroid
	res = c.Get(m[0])
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.PackageKey(e.Package(id)), apk, "a.apk")
	expectEq(t, "install_log.mail", e.DB.Row("SELECT mail FROM install_log WHERE package_id = ?", id).Str("mail"), user.Mail)
}

func TestMyAppsNotificationSetting(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	id := e.MustAPIUpload(app.APIKey, "apk", "a.apk", e2e.APK("com.example.a"), nil)
	user := e.NewUser("user")
	e2e.ExpectStatus(t, user.Get("/package/install?id="+itoa(id)), http.StatusFound)

	notify := func() string {
		return e.DB.Row("SELECT notify FROM app_install_user WHERE app_id = ? AND mail = ?", app.ID, user.Mail).Str("notify")
	}

	res := user.Get("/ajax/notification_setting?id=" + itoa(app.ID) + "&value=0")
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	expectEq(t, "notify", jsonBool(t, m, "notify"), false)
	expectEq(t, "mail", jsonStr(t, m, "mail"), user.Mail)
	expectEq(t, "notify", notify(), "0")

	res = user.Get("/ajax/notification_setting?id=" + itoa(app.ID) + "&value=1")
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectEq(t, "notify", notify(), "1")

	res = owner.Get("/ajax/notification_setting?id=" + itoa(app.ID) + "&value=1")
	e2e.ExpectStatus(t, res, http.StatusNotFound)

	// インストール済み一覧から外す (install_log は残る)
	user.ExpectRedirect(user.Get("/myapps/delete?id="+itoa(app.ID)), "/myapps/installed")
	expectEq(t, "app_install_user", e.DB.Count("SELECT count(*) FROM app_install_user WHERE app_id = ? AND mail = ?", app.ID, user.Mail), 0)
	expectEq(t, "install_log", e.DB.Count("SELECT count(*) FROM install_log WHERE app_id = ? AND mail = ?", app.ID, user.Mail), 1)
	if strings.Contains(user.Get("/myapps/installed").String(), app.Title) {
		t.Fatal("installed apps still contains the app")
	}
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(s)
}

// rfc5987 は RFC 5987 の ext-value の形式でエンコードする (英数字と -._~ 以外を %XX に).
func rfc5987(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-._~", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
