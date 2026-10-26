package scenario

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	e2e "emlauncher-e2e"
)

var reGuestpassURL = regexp.MustCompile(`https?://[^"<\s]+/package/guestpass_install\?token=([0-9A-Za-z]{32})`)

func TestGuestPass(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, e2e.Unique("app"))
	apk := e2e.APK("com.example.guest")
	pid := e.MustAPIUpload(app.APIKey, "apk", "guest.apk", apk, nil)
	id := itoa(pid)

	expectErrorPage(t, e, other.Get("/package/create_guestpass?id="+id), http.StatusNotFound, "Not Found")

	res := owner.Get("/package/create_guestpass?id=" + id)
	owner.ExpectRedirect(res, "/package/guestpass?id="+id+"&guestpass_id=")
	gpID := e2e.QueryInt(t, res.Location(), "guestpass_id")
	gp := e.DB.Row("SELECT * FROM guest_pass WHERE id = ?", gpID)
	expectEq(t, "app_id", gp.Int("app_id"), app.ID)
	expectEq(t, "package_id", gp.Int("package_id"), pid)
	expectEq(t, "mail", gp.Str("mail"), owner.Mail)
	expectMatch(t, "token", reToken32, gp.Str("token"))
	e.ExpectRecent("created", gp.Str("created"))
	if d := e.ParseLocalTime(gp.Str("expired")).Sub(e.ParseLocalTime(gp.Str("created"))); d != 7*24*time.Hour {
		t.Fatalf("expired - created = %v, want 1 week", d)
	}

	page := owner.Get(res.Location())
	e2e.ExpectStatus(t, page, http.StatusOK)
	m := reGuestpassURL.FindStringSubmatch(page.String())
	if m == nil || m[1] != gp.Str("token") {
		t.Fatalf("guestpass url not found:\n%s", page.Body)
	}
	expectErrorPage(t, e, other.Get("/package/guestpass?id="+id+"&guestpass_id="+gp.Str("id")), http.StatusForbidden, "Forbidden")
	expectContains(t, "package page", owner.Get("/package?id="+id).String(), "guestpass_id="+gp.Str("id"))
	e2e.ExpectStatus(t, owner.Get("/guestpass/list"), http.StatusOK)

	// ログインしていないゲストがインストールできる
	guest := e.NewClient()
	guest.UserAgent = e2e.UAAndroid
	res = guest.Get(m[0])
	e2e.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), e2e.PackageKey(e.Package(pid)), apk, "guest.apk")
	log := e.DB.Row("SELECT * FROM guestpass_log WHERE guest_pass_id = ?", gpID)
	expectEq(t, "user_agent", log.Str("user_agent"), e2e.UAAndroid)
	if log.Str("ip_address") == "" {
		t.Fatal("ip_address is empty")
	}
	e.ExpectRecent("installed", log.Str("installed"))
	// ゲストのインストールは install_log には残らない
	expectEq(t, "install_log", e.DB.Count("SELECT count(*) FROM install_log WHERE package_id = ?", pid), 0)

	// iOS のゲスト
	ipa := e2e.IPA("Guest", "com.example.guest.ios")
	ipaID := itoa(e.MustAPIUpload(app.APIKey, "ipa", "guest.ipa", ipa, nil))
	res = owner.Get("/package/create_guestpass?id=" + ipaID)
	iosToken := e.DB.Row("SELECT token FROM guest_pass WHERE id = ?", e2e.QueryInt(t, res.Location(), "guestpass_id")).Str("token")
	ios := e.NewClient()
	ios.UserAgent = e2e.UAiOS
	res = ios.Get("/package/guestpass_install?token=" + iosToken)
	e2e.ExpectStatus(t, res, http.StatusFound)
	if !strings.HasPrefix(res.Location(), "itms-services://") {
		t.Fatalf("location = %q", res.Location())
	}

	// 失効
	res = owner.Get("/package/expire_guestpass?id=" + id + "&guestpass_id=" + gp.Str("id"))
	owner.ExpectRedirect(res, "/package")
	if exp := e.ParseLocalTime(e.DB.Row("SELECT expired FROM guest_pass WHERE id = ?", gpID).Str("expired")); time.Until(exp) > time.Minute {
		t.Fatalf("guest pass not expired: %v", exp)
	}
	time.Sleep(1100 * time.Millisecond)
	expectErrorPage(t, e, guest.Get(m[0]), http.StatusForbidden, "Invalid token")
	expectErrorPage(t, e, guest.Get("/package/guestpass_install?token=nosuchtoken"), http.StatusNotFound, "Not Found")
}
