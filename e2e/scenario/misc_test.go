package scenario

import (
	"bytes"
	"image/png"
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
	// ログイン不要
	res := e.NewClient().Get("/qr/code?s=250&q=%2Fpackage%2Finstall%3Ftoken%3Dabc")
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectEq(t, "Content-Type", res.Header.Get("Content-Type"), "image/png")
	img, err := png.Decode(bytes.NewReader(res.Body))
	if err != nil {
		t.Fatalf("not png: %v", err)
	}
	if w := img.Bounds().Dx(); w < 250 || w > 300 {
		t.Fatalf("qr width = %d", w)
	}
	// サイズの上限は 300 (+余白)
	res = e.NewClient().Get("/qr/code?s=1000&q=%2F")
	img, _ = png.Decode(bytes.NewReader(res.Body))
	if w := img.Bounds().Dx(); w > 320 {
		t.Fatalf("qr width = %d, want <= 320", w)
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
