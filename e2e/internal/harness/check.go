package harness

import (
	"net/http"
	"net/url"
	"strings"
)

// ExpectErrorPage はエラーページ (本文に message を含む) が status で返ることを確認する.
func (e *Env) ExpectErrorPage(res *Response, status int, message string) {
	e.t.Helper()
	if !strings.Contains(res.String(), message) {
		e.t.Fatalf("%s: error page does not contain %q (status=%d)\n%s", res.URL, message, res.Status, truncate(res.Body, 2000))
	}
	if res.Status == http.StatusOK && e.PHPBug("app / package モジュールのエラーページがステータス 200 で返る (build() が headers を捨てる)") {
		return
	}
	ExpectStatus(e.t, res, status)
}

// KeyFromURL はダウンロード URL (LocalFile の /files/... / S3 の署名付き URL) からストレージのキーを取り出す.
func (e *Env) KeyFromURL(rawURL string) string {
	e.t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		e.t.Fatalf("parse %q: %v", rawURL, err)
	}
	var prefix string
	switch e.Storage.Kind() {
	case "local":
		prefix = "/files/"
	case "s3":
		prefix = "/" + e.Config.S3Bucket + "/"
	}
	if !strings.HasPrefix(u.Path, prefix) {
		e.t.Fatalf("download url %q does not start with %q", rawURL, prefix)
	}
	return strings.TrimPrefix(u.Path, prefix)
}
