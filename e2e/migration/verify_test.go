package migration

import (
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"emlauncher-e2e/internal/fixture"
	"emlauncher-e2e/internal/harness"
)

// TestVerify は移行後の環境で, 移行元で作った中間状態がそのまま使えることを確認する.
// 前半は読み取りのみ, 後半は状態を変更する操作 (順序に意味がある).
func TestVerify(t *testing.T) {
	e := phase(t, "verify")
	s := loadState(t)

	// 移行手順の前後で DB とストレージが変わっていない
	t.Run("db unchanged", func(t *testing.T) {
		got := snapshotDB(e)
		for table := range tables {
			if !reflect.DeepEqual(got[table], s.DB[table]) {
				t.Errorf("table %s changed by migration:\n got: %v\nwant: %v", table, rowsString(got[table]), rowsString(s.DB[table]))
			}
		}
	})
	t.Run("storage unchanged", func(t *testing.T) {
		got := snapshotStorage(e)
		if !reflect.DeepEqual(got, s.Storage) {
			t.Errorf("storage changed by migration:\n got: %v\nwant: %v", keys(got), keys(s.Storage))
		}
	})
	if t.Failed() {
		t.FailNow()
	}

	// 既存のパスワードで再ログインできる
	login := func(name string) *harness.User {
		u := s.Users[name]
		c := e.NewClient()
		c.Login(u.Mail, u.Password)
		return &harness.User{Mail: u.Mail, Password: u.Password, Client: c}
	}
	owner, member := login("owner"), login("member")
	mainApp, subApp := s.Apps["main"], s.Apps["sub"]

	t.Run("pages", func(t *testing.T) {
		top := member.Get("/").String()
		for _, a := range s.Apps {
			contains(t, "top", top, a.Title)
		}
		app := member.Get("/app?id=" + itoa(mainApp.ID))
		harness.ExpectStatus(t, app, http.StatusOK)
		contains(t, "app page", app.String(), e.Application(mainApp.ID).Str("icon_key"))
		for _, p := range s.Packages {
			page := member.Get("/package?id=" + itoa(p.ID))
			harness.ExpectStatus(t, page, http.StatusOK)
			contains(t, "package page", page.String(), htmlEscape(p.Title))
		}
		contains(t, "installed apps", member.Get("/myapps/installed").String(), mainApp.Title)
		contains(t, "own apps", owner.Get("/myapps/own").String(), mainApp.Title)
		contains(t, "comments", member.Get("/app/comment?id="+itoa(mainApp.ID)).String(), "member comment on apk")
		harness.ExpectStatus(t, owner.Get("/app/preferences?id="+itoa(mainApp.ID)), http.StatusOK)
		harness.ExpectStatus(t, member.Get("/app/preferences?id="+itoa(mainApp.ID)), http.StatusOK) // 共同オーナー
	})

	t.Run("api", func(t *testing.T) {
		for name, a := range s.Apps {
			if got := packageList(t, e, a.APIKey); !reflect.DeepEqual(got, s.PackageList[name]) {
				t.Errorf("package_list of %s:\n got: %+v\nwant: %+v", name, got, s.PackageList[name])
			}
		}
	})

	t.Run("download files uploaded before migration", func(t *testing.T) {
		for name, p := range s.Packages {
			c := e.NewUser("dl")
			c.UserAgent = harness.UAPC
			res := c.Get("/package/install?id=" + itoa(p.ID))
			harness.ExpectStatus(t, res, http.StatusFound)
			expectFile(t, e, s, res.Location(), p.Key)
			if strings.HasSuffix(p.FileName, ".apk") || strings.HasSuffix(p.FileName, ".aab") {
				c.UserAgent = harness.UAAndroid
				res = c.Get("/package/install?id=" + itoa(p.ID))
				harness.ExpectStatus(t, res, http.StatusFound)
				expectFile(t, e, s, res.Location(), p.AndroidKey)
			}
			_ = name
		}
	})

	t.Run("ios install", func(t *testing.T) {
		member.UserAgent = harness.UAiOS
		defer func() { member.UserAgent = harness.UAPC }()
		res := member.Get("/package/install?id=" + itoa(s.Packages["ipa"].ID))
		harness.ExpectStatus(t, res, http.StatusFound)
		plistURL, _ := url.QueryUnescape(strings.TrimPrefix(res.Location(), "itms-services://?action=download-manifest&url="))
		plist := e.NewClient().Get(plistURL)
		harness.ExpectStatus(t, plist, http.StatusOK)
		contains(t, "plist", plist.String(), "com.example.main")
		m := regexp.MustCompile(`<string>([^<]*` + regexp.QuoteMeta(s.Packages["ipa"].Key[strings.LastIndex(s.Packages["ipa"].Key, "_")+1:]) + `[^<]*)</string>`).FindStringSubmatch(plist.String())
		if m == nil {
			t.Fatalf("ipa url not found in plist:\n%s", plist.Body)
		}
		expectFile(t, e, s, strings.ReplaceAll(m[1], "&amp;", "&"), s.Packages["ipa"].Key)
	})

	t.Run("install tokens issued before migration", func(t *testing.T) {
		for _, tc := range []struct {
			token, pkg, mail string
		}{
			{s.InstallTokenAPI, "aab", owner.Mail},
			{s.InstallTokenUI, "apk", member.Mail},
		} {
			c := e.NewClient()
			c.UserAgent = harness.UAAndroid
			before := e.DB.Count("SELECT count(*) FROM install_log WHERE package_id = ? AND mail = ?", s.Packages[tc.pkg].ID, tc.mail)
			res := c.Get("/package/install?token=" + tc.token)
			harness.ExpectStatus(t, res, http.StatusFound)
			expectFile(t, e, s, res.Location(), s.Packages[tc.pkg].AndroidKey)
			after := e.DB.Count("SELECT count(*) FROM install_log WHERE package_id = ? AND mail = ?", s.Packages[tc.pkg].ID, tc.mail)
			if after != before+1 {
				t.Errorf("install_log for %s: %d -> %d", tc.mail, before, after)
			}
		}
	})

	t.Run("guest passes", func(t *testing.T) {
		g := e.NewClient()
		g.UserAgent = harness.UAAndroid
		res := g.Get("/package/guestpass_install?token=" + s.GuestPassActive)
		harness.ExpectStatus(t, res, http.StatusFound)
		expectFile(t, e, s, res.Location(), s.Packages["apk"].Key)
		e.ExpectErrorPage(g.Get("/package/guestpass_install?token="+s.GuestPassExpired), http.StatusForbidden, "Invalid token")
	})

	t.Run("orphan temp file is left as is", func(t *testing.T) {
		if !e.Storage.Exists(s.OrphanTemp) {
			t.Fatalf("%s is gone", s.OrphanTemp)
		}
	})

	// ここから状態を変更する操作
	t.Run("upload notification follows settings", func(t *testing.T) {
		ownerMails, memberMails := len(e.Mailbox.To(owner.Mail)), len(e.Mailbox.To(member.Mail))
		newID := e.MustAPIUpload(mainApp.APIKey, "after migration", "new.apk", fixture.APK("com.example.main"),
			url.Values{"notify": {"true"}, "tags": {"android"}})
		k := harness.PackageKey(e.Package(newID))
		s.After = &AfterState{Package: PkgState{ID: newID, App: "main", Title: "after migration", FileName: "new.apk",
			Key: k, SHA256: sha(e.Storage.Read(k)), AndroidKey: k}}
		mails := e.Mailbox.WaitTo(owner.Mail, ownerMails+1)
		contains(t, "subject", mails[len(mails)-1].Subject, mainApp.Title)
		if n := len(e.Mailbox.To(member.Mail)); n != memberMails {
			t.Errorf("muted member received notification (%d -> %d)", memberMails, n)
		}
		tags := e.DB.Count("SELECT count(*) FROM tag WHERE app_id = ? AND name = 'android'", mainApp.ID)
		if tags != 1 {
			t.Errorf("tag 'android' rows = %d, want 1 (existing tag must be reused)", tags)
		}
	})

	t.Run("comment number continues", func(t *testing.T) {
		member.ExpectRedirect(member.PostForm("/app/comment_post",
			url.Values{"id": {itoa(mainApp.ID)}, "message": {"after migration"}, "package_id": {""}}), "/app/comment")
		n := e.DB.Row("SELECT number FROM comment WHERE app_id = ? ORDER BY id DESC LIMIT 1", mainApp.ID).Int("number")
		if n != int64(s.CommentCount+1) {
			t.Errorf("comment number = %d, want %d", n, s.CommentCount+1)
		}
	})

	t.Run("edit package uploaded before migration", func(t *testing.T) {
		res := e.APICall("package_edit", mainApp.APIKey, url.Values{"id": {itoa(s.Packages["ipa"].ID)}, "title": {"renamed"}})
		harness.ExpectStatus(t, res, http.StatusOK)
		if got := e.Package(s.Packages["ipa"].ID).Str("title"); got != "renamed" {
			t.Errorf("title = %q", got)
		}
	})

	t.Run("delete package uploaded before migration", func(t *testing.T) {
		p := s.Packages["aab"]
		attached := e.DB.Rows("SELECT * FROM attached_file WHERE package_id = ?", p.ID)
		confirm := owner.Get("/package/delete_confirm?id=" + itoa(p.ID))
		owner.ExpectRedirect(owner.PostForm("/package/delete",
			url.Values{"id": {itoa(p.ID)}, "token": {harness.HiddenToken(t, confirm)}}), "/app?id=")
		if e.Storage.Exists(p.Key) {
			t.Errorf("%s still exists", p.Key)
		}
		for _, a := range attached {
			if e.Storage.Exists(harness.AttachedKey(a)) {
				t.Errorf("%s still exists", harness.AttachedKey(a))
			}
		}
	})

	t.Run("password reset", func(t *testing.T) {
		c := e.NewClient()
		before := len(e.Mailbox.To(member.Mail))
		harness.ExpectStatus(t, c.PostForm("/login/password_confirm", url.Values{"email": {member.Mail}}), http.StatusOK)
		mails := e.Mailbox.WaitTo(member.Mail, before+1)
		key := reResetKey.FindStringSubmatch(mails[len(mails)-1].Body)[1]
		harness.ExpectStatus(t, c.PostForm("/login/password_commit", url.Values{"key": {key}, "password": {"changed"}}), http.StatusOK)
		e.NewClient().Login(member.Mail, "changed")
		s.After.MemberPassword = "changed"
	})

	t.Run("issue install token for rollback check", func(t *testing.T) {
		res := e.APICall("create_token", mainApp.APIKey, url.Values{
			"id": {itoa(s.After.Package.ID)}, "mail": {owner.Mail}, "expire_hour": {"24"}})
		harness.ExpectStatus(t, res, http.StatusOK)
		s.After.InstallToken = reInstallTokenURL.FindStringSubmatch(res.JSONObject(t)["install_url"].(string))[1]
		saveState(t, s)
	})

	t.Run("delete old packages batch", func(t *testing.T) {
		const keep = 2
		before := map[int64][]harness.Row{}
		for _, a := range []AppState{mainApp, subApp} {
			before[a.ID] = e.DB.Rows("SELECT * FROM package WHERE app_id = ? ORDER BY id DESC", a.ID)
		}
		t.Log(e.RunBatch(keep))
		for appID, pkgs := range before {
			kept := 0
			for _, p := range pkgs {
				exists := e.DB.Count("SELECT count(*) FROM package WHERE id = ?", p.Str("id")) == 1
				want := p.Str("protect") == "1" || kept < keep
				if p.Str("protect") != "1" && kept < keep {
					kept++
				}
				if exists != want {
					t.Errorf("app %d package %s (protect=%s) exists=%v, want %v", appID, p.Str("id"), p.Str("protect"), exists, want)
				}
				if exists != e.Storage.Exists(harness.PackageKey(p)) {
					t.Errorf("package %s: db exists=%v but storage exists=%v", p.Str("id"), exists, !exists)
				}
			}
		}
	})
}

// expectFile はダウンロード URL が key を指し, 中身が移行前のものと同じであることを確認する.
func expectFile(t *testing.T, e *harness.Env, s *State, rawURL, key string) {
	t.Helper()
	if got := e.KeyFromURL(rawURL); got != key {
		t.Fatalf("download key = %q, want %q", got, key)
	}
	res := e.Download(rawURL)
	harness.ExpectStatus(t, res, http.StatusOK)
	if got := sha(res.Body); got != s.Storage[key] {
		t.Fatalf("content of %s changed", key)
	}
}

func contains(t *testing.T, what, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("%s does not contain %q", what, sub)
	}
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func rowsString(rows []harness.Row) string {
	var b strings.Builder
	for _, r := range rows {
		var cols []string
		for k := range r {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		b.WriteString("{")
		for _, c := range cols {
			b.WriteString(c + "=" + r.Str(c) + " ")
		}
		b.WriteString("} ")
	}
	return b.String()
}
