package emlauncher

import (
	"net/http"
	"strings"
)

// absURL は path (クエリ付き可) を絶対 URL にする. PHP 版の mfwRequest::makeUrl と同じく,
// ロードバランサ配下では X-Forwarded-Host / X-Forwarded-Proto を使う.
// scheme を指定するとそれを優先する ("https" など).
func absURL(r *http.Request, path, scheme string) string {
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	if scheme == "" {
		scheme = requestScheme(r)
	}
	return scheme + "://" + host + path
}

func requestScheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return p
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// requestURL は現在のリクエストの絶対 URL.
func requestURL(r *http.Request) string {
	return absURL(r, r.URL.RequestURI(), "")
}

// redirect は PHP 版の redirect() と同じく, パスなら絶対 URL にして 302 を返す.
func redirect(w http.ResponseWriter, r *http.Request, target string) {
	if !strings.Contains(target, "://") {
		target = absURL(r, target, "")
	}
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}
