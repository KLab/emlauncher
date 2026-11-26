package web

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// requestScheme はブラウザから見たスキーム. リバースプロキシ / ALB の X-Forwarded-Proto を優先する.
func requestScheme(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		p, _, _ = strings.Cut(p, ",")
		return strings.ToLower(strings.TrimSpace(p))
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// requestHost はブラウザから見たホスト. X-Forwarded-Host を優先する.
func requestHost(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		h, _, _ = strings.Cut(h, ",")
		return strings.TrimSpace(h)
	}
	return r.Host
}

// baseURL は "scheme://host". 画面のリンクや API の応答の URL の起点.
func baseURL(r *http.Request) string {
	return requestScheme(r) + "://" + requestHost(r)
}

// absURL は path ("/app?id=1" など) をブラウザから見た絶対 URL にする. scheme が空ならリクエストのもの.
func absURL(r *http.Request, scheme, path string) string {
	if scheme == "" {
		scheme = requestScheme(r)
	}
	return scheme + "://" + requestHost(r) + path
}

// requestURL はリクエストの URL (絶対 URL).
func requestURL(r *http.Request) string {
	return baseURL(r) + r.URL.RequestURI()
}

// redirect は 302 でリダイレクトする. path が "/" で始まればリクエストのホストの絶対 URL にする.
func redirect(w http.ResponseWriter, r *http.Request, path string) error {
	if strings.HasPrefix(path, "/") {
		path = baseURL(r) + path
	}
	http.Redirect(w, r, path, http.StatusFound)
	return nil
}

// remoteIP はクライアントの IP アドレス. X-Forwarded-For があればその値 (PHP 版と同じ).
// secretParams はログに値を出さないクエリパラメータ (API キー, トークン, パスワードなど).
var secretParams = map[string]bool{
	"api_key": true, "api-key": true, "token": true, "key": true, "t": true, "password": true, "code": true,
}

// logURI はログに出す URI (パスとクエリ). secretParams の値は *** にする.
func logURI(u *url.URL) string {
	if u.RawQuery == "" {
		return u.RequestURI()
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return u.EscapedPath() + "?(invalid query)"
	}
	redacted := false
	for k, vs := range q {
		if secretParams[k] {
			for i := range vs {
				vs[i] = "***"
			}
			redacted = true
		}
	}
	if !redacted {
		return u.RequestURI()
	}
	// Encode は * をエスケープするので, 読みやすいように戻す
	return u.EscapedPath() + "?" + strings.ReplaceAll(q.Encode(), "=%2A%2A%2A", "=***")
}

func remoteIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return f
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isAndroid / isIOS は User-Agent から端末を判定する (PHP 版の mfwUserAgent と同じ規則).
func isAndroid(r *http.Request) bool {
	return strings.Contains(r.UserAgent(), "Android")
}

func isIOS(r *http.Request) bool {
	ua := r.UserAgent()
	if strings.Contains(ua, "Android") {
		return false
	}
	for _, s := range []string{"(iPhone;", "(iPod;", "(iPod touch;", "(iPad;"} {
		if strings.Contains(ua, s) {
			return true
		}
	}
	return false
}

type ctxKey int

const (
	ctxForm ctxKey = iota
	ctxUser
	ctxLog
)

// formOf はリクエストのパラメータ (withForm で読んだもの).
func formOf(r *http.Request) *form {
	if f, ok := r.Context().Value(ctxForm).(*form); ok {
		return f
	}
	return &form{Values: r.URL.Query()}
}

// withForm はリクエストのパラメータを読んで context に入れる. 処理が終わったら一時ファイルを消す.
// 読めなければ onError (エラーページ / JSON) で応答する.
func (s *Server) withForm(onError func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f, err := parseForm(w, r)
			if err != nil {
				onError(w, r, err)
				return
			}
			defer f.cleanup()
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxForm, f)))
		})
	}
}

// loginUser はログイン中のユーザのメールアドレス (requireLogin などで context に入れたもの).
func loginUser(r *http.Request) string {
	u, _ := r.Context().Value(ctxUser).(string)
	return u
}

func withUser(r *http.Request, mail string) *http.Request {
	logUser(r, mail)
	return r.WithContext(context.WithValue(r.Context(), ctxUser, mail))
}

// intParam は整数のパラメータ. 無いか整数でなければ def.
func (f *form) intParam(name string, def int64) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(f.Get(name)), 10, 64)
	if err != nil {
		return def
	}
	return n
}

// phpInt は PHP の (int) と同じ規則の整数のパラメータ. 無ければ def, 先頭の数字だけを使い
// ("10abc" は 10), 数字で始まらなければ 0.
func (f *form) phpInt(name string, def int64) int64 {
	if !f.has(name) {
		return def
	}
	return phpInt(f.Get(name))
}

func phpInt(v string) int64 {
	v = strings.TrimLeft(v, " \t\n\r\v\f")
	end := 0
	if end < len(v) && (v[end] == '-' || v[end] == '+') {
		end++
	}
	for end < len(v) && '0' <= v[end] && v[end] <= '9' {
		end++
	}
	n, _ := strconv.ParseInt(v[:end], 10, 64)
	return n
}

// optional は name があればその値へのポインタ, 無ければ nil.
func (f *form) optional(name string) *string {
	if !f.has(name) {
		return nil
	}
	v := f.Get(name)
	return &v
}

// truthy は PHP の (bool) と同じ判定 (空文字列と "0" が偽).
func truthy(v string) bool {
	return v != "" && v != "0"
}
