package migration

import (
	"net/http"
	"testing"

	e2e "emlauncher-e2e"
)

// TestRollback は移行元 (PHP 版) に切り戻した後, 移行先で作ったデータが使えることを確認する.
// verify の後に, 移行先を止めて移行元を起動してから実行する.
func TestRollback(t *testing.T) {
	e := phase(t, "rollback")
	s := loadState(t)
	if s.After == nil {
		t.Fatal("verify has not recorded the data created after migration")
	}
	mainApp := s.Apps["main"]

	// 移行先で変更したパスワード (移行先が作ったハッシュ) でログインできる
	member := e.NewClient()
	member.Login(s.Users["member"].Mail, s.After.MemberPassword)

	t.Run("pages", func(t *testing.T) {
		app := member.Get("/app?id=" + itoa(mainApp.ID))
		e2e.ExpectStatus(t, app, http.StatusOK)
		contains(t, "app page", app.String(), s.After.Package.Title)
		e2e.ExpectStatus(t, member.Get("/package?id="+itoa(s.After.Package.ID)), http.StatusOK)
		e2e.ExpectStatus(t, member.Get("/app/comment?id="+itoa(mainApp.ID)), http.StatusOK)
	})

	t.Run("download package uploaded by the new version", func(t *testing.T) {
		member.UserAgent = e2e.UAAndroid
		res := member.Get("/package/install?id=" + itoa(s.After.Package.ID))
		e2e.ExpectStatus(t, res, http.StatusFound)
		if got := e.KeyFromURL(res.Location()); got != s.After.Package.Key {
			t.Fatalf("download key = %q, want %q", got, s.After.Package.Key)
		}
		dl := e.Download(res.Location())
		e2e.ExpectStatus(t, dl, http.StatusOK)
		if sha(dl.Body) != s.After.Package.SHA256 {
			t.Fatal("content mismatch")
		}
	})

	t.Run("install token issued by the new version", func(t *testing.T) {
		c := e.NewClient()
		c.UserAgent = e2e.UAAndroid
		res := c.Get("/package/install?token=" + s.After.InstallToken)
		e2e.ExpectStatus(t, res, http.StatusFound)
		if got := e.KeyFromURL(res.Location()); got != s.After.Package.Key {
			t.Fatalf("download key = %q, want %q", got, s.After.Package.Key)
		}
	})

	t.Run("api", func(t *testing.T) {
		list := packageList(t, e, mainApp.APIKey)
		found := false
		for _, p := range list {
			if p.ID == itoa(s.After.Package.ID) {
				found = true
			}
		}
		if !found {
			t.Errorf("package %d is not in package_list", s.After.Package.ID)
		}
	})
}
