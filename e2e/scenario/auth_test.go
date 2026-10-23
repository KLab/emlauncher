package scenario

import (
	"net/http"
	"regexp"
	"testing"

	"emlauncher-e2e/internal/harness"
)

func TestLoginRequired(t *testing.T) {
	e := harness.New(t)
	c := e.NewClient()
	for _, path := range []string{"/", "/app?id=1", "/package?id=1", "/myapps/installed", "/guestpass/list", "/doc/api"} {
		c.ExpectRedirect(c.Get(path), "/login")
	}
	harness.ExpectStatus(t, c.Get("/login"), http.StatusOK)
	res := c.Get("/ajax/notification_setting?id=1&value=1")
	harness.ExpectStatus(t, res, http.StatusForbidden)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "login required")
}

func TestPasswordLoginAndLogout(t *testing.T) {
	e := harness.New(t)
	mail, password := e.CreateUserRecord("user")
	c := e.NewClient()

	res := c.PostForm("/login/password", form("email", mail, "password", "wrong-password"))
	if res.Status == http.StatusFound {
		t.Fatalf("login with wrong password redirected to %s", res.Location())
	}
	res = c.PostForm("/login/password", form("email", "nobody@example.com", "password", password))
	if res.Status == http.StatusFound {
		t.Fatalf("login with unknown user redirected to %s", res.Location())
	}
	c.ExpectRedirect(c.Get("/"), "/login")

	// ログイン前に開こうとした URL に戻る
	c.ExpectRedirect(c.Get("/myapps/installed"), "/login")
	expectEq(t, "redirect after login", c.Login(mail, password), c.URL("/myapps/installed"))
	harness.ExpectStatus(t, c.Get("/"), http.StatusOK)
	// ログイン済みでログイン画面を開くとトップへ
	c.ExpectRedirect(c.Get("/login"), "/")

	c.ExpectRedirect(c.Get("/logout"), "/login")
	c.ExpectRedirect(c.Get("/"), "/login")

	// 保存された URL が無ければトップへ
	expectEq(t, "redirect after login", c.Login(mail, password), c.URL("/"))
}

func TestPasswordHashCompatibility(t *testing.T) {
	e := harness.New(t)
	// PHP 版が作ったものと同じ形式のハッシュ (stretch 10..20) でログインできる
	for _, stretch := range []int{10, 20} {
		mail := harness.Unique("hash") + "@example.com"
		e.DB.Exec("INSERT INTO user_pass (mail, passhash) VALUES (?, ?)",
			mail, harness.PasswordHash("secret-パス", "AbCdEfGh12345678", stretch))
		c := e.NewClient()
		c.Login(mail, "secret-パス")
	}
	// passhash が NULL のユーザ (パスワード未設定) はログインできない
	mail := harness.Unique("nopass") + "@example.com"
	e.DB.Exec("INSERT INTO user_pass (mail) VALUES (?)", mail)
	res := e.NewClient().PostForm("/login/password", form("email", mail, "password", ""))
	if res.Status == http.StatusFound {
		t.Fatal("user without password logged in")
	}
}

var reResetURL = regexp.MustCompile(`(https?://\S+/login/password_reset\?key=([0-9a-f]{40}))`)

func TestPasswordReset(t *testing.T) {
	e := harness.New(t)
	// 新規ユーザは mail のみ登録し, パスワードリセットでパスワードを設定する
	mail := harness.Unique("reset") + "@example.com"
	e.DB.Exec("INSERT INTO user_pass (mail) VALUES (?)", mail)
	c := e.NewClient()

	harness.ExpectStatus(t, c.Get("/login/password_reminder"), http.StatusOK)
	harness.ExpectStatus(t, c.PostForm("/login/password_confirm", form("email", mail)), http.StatusOK)

	mails := e.Mailbox.WaitTo(mail, 1)
	m := reResetURL.FindStringSubmatch(mails[0].Body)
	if m == nil {
		t.Fatalf("reset url not found in mail:\n%s", mails[0].Body)
	}
	resetURL, key := m[1], m[2]
	expectContains(t, "reset url", resetURL, e.Config.BaseURL)

	harness.ExpectStatus(t, c.Get(resetURL), http.StatusOK)
	harness.ExpectStatus(t, c.PostForm("/login/password_commit", form("key", key, "password", "new-password")), http.StatusOK)

	passhash := e.DB.Row("SELECT passhash FROM user_pass WHERE mail = ?", mail).Str("passhash")
	expectMatch(t, "passhash", regexp.MustCompile(`^(1[0-9]|20):[0-9A-Za-z]{16}:[0-9a-f]{40}$`), passhash)
	if !harness.CheckPasswordHash(passhash, "new-password") {
		t.Fatalf("passhash %q does not match the new password", passhash)
	}
	e.NewClient().Login(mail, "new-password")

	// 使用済みのキーは使えない
	c.PostForm("/login/password_commit", form("key", key, "password", "another-password"))
	if after := e.DB.Row("SELECT passhash FROM user_pass WHERE mail = ?", mail).Str("passhash"); after != passhash {
		t.Fatal("password changed with used key")
	}

	// 登録されていないアドレスにはメールを送らない
	unknown := harness.Unique("unknown") + "@example.com"
	harness.ExpectStatus(t, c.PostForm("/login/password_confirm", form("email", unknown)), http.StatusOK)
	expectEq(t, "mails to unknown", len(e.Mailbox.To(unknown)), 0)
}
