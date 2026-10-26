package scenario

import (
	"net/http"
	"testing"

	e2e "emlauncher-e2e"
)

func TestHealth(t *testing.T) {
	e := e2e.New(t)
	res := e.NewClient().Get("/health")
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectEq(t, "body", res.String(), "OK")
}

func TestQRCode(t *testing.T) {
	e := e2e.New(t)
	for _, tc := range []struct {
		query string // /qr/code のクエリ
		path  string // QR コードの中身になるパス
		size  int    // QR コード部分の大きさ (上限 300, 既定 150). 周りに 10px の余白が付く
	}{
		{"s=250&q=%2Fpackage%2Finstall%3Ftoken%3Dabc", "/package/install?token=abc", 250},
		{"s=150&q=%2Fapp%3Fid%3D1", "/app?id=1", 150},
		{"q=%2Fpackage%3Fid%3D2", "/package?id=2", 150},
		{"s=1000&q=%2F", "/", 300},
	} {
		// ログイン不要
		res := e.NewClient().Get("/qr/code?" + tc.query)
		e2e.ExpectStatus(t, res, http.StatusOK)
		expectEq(t, "Content-Type", res.Header.Get("Content-Type"), "image/png")
		text, w, h := e2e.DecodeQR(t, res.Body)
		expectEq(t, "qr content of "+tc.query, text, e.Config.BaseURL+tc.path)
		expectEq(t, "qr image size of "+tc.query, [2]int{w, h}, [2]int{tc.size + 20, tc.size + 20})
	}
}

func TestDocPages(t *testing.T) {
	e := e2e.New(t)
	u := e.NewUser("user")
	for _, p := range []string{"api", "api_upload", "api_package_list", "api_delete", "api_create_token",
		"api_package_edit", "api_package_attach", "api_package_detach"} {
		e2e.ExpectStatus(t, u.Get("/doc/"+p), http.StatusOK)
	}
	e2e.ExpectStatus(t, u.Get("/doc/no_such_page"), http.StatusNotFound)
	res := u.Get("/doc/keystore")
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectContains(t, "keystore page", res.String(), "emlauncher")
}

func TestTopPagePaging(t *testing.T) {
	e := e2e.New(t)
	u := e.NewUser("user")
	app := e.CreateApp(u, e2e.Unique("newest"))
	// 最新のアプリは 1 ページ目に出る
	expectContains(t, "top page", u.Get("/?page=1").String(), app.Title)
	e2e.ExpectStatus(t, u.Get("/?page=9999"), http.StatusOK)
}
