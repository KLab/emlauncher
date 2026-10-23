package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

const (
	UAAndroid = "Mozilla/5.0 (Linux; Android 13; Pixel 7 Build/TQ3A.230805.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36"
	UAiOS     = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1"
	UAPC      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)

// Client はリダイレクトを追わない HTTP クライアント. Cookie (セッション) は保持する.
type Client struct {
	t         testing.TB
	base      string
	http      *http.Client
	UserAgent string
}

func (e *Env) NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		t:    e.t,
		base: e.Config.BaseURL,
		http: &http.Client{
			Jar: jar,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		UserAgent: UAPC,
	}
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
	URL    string
}

func (r *Response) Location() string { return r.Header.Get("Location") }

func (r *Response) String() string { return string(r.Body) }

// JSON はレスポンスボディを json.Number を保ったままデコードする.
func (r *Response) JSON(t testing.TB) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(r.Body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("invalid json (status=%d): %v\n%s", r.Status, err, r.Body)
	}
	return v
}

func (r *Response) JSONObject(t testing.TB) map[string]any {
	t.Helper()
	m, ok := r.JSON(t).(map[string]any)
	if !ok {
		t.Fatalf("json is not an object: %s", r.Body)
	}
	return m
}

// URL は path (クエリ付き可) から絶対 URL を作る. 絶対 URL はそのまま返す.
func (c *Client) URL(path string) string {
	if strings.Contains(path, "://") {
		return path
	}
	return c.base + path
}

func (c *Client) Do(req *http.Request) *Response {
	c.t.Helper()
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("%s %s: read body: %v", req.Method, req.URL, err)
	}
	return &Response{Status: res.StatusCode, Header: res.Header, Body: body, URL: req.URL.String()}
}

func (c *Client) Get(path string) *Response {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.URL(path), nil)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.Do(req)
}

func (c *Client) PostForm(path string, form url.Values) *Response {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodPost, c.URL(path), strings.NewReader(form.Encode()))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Do(req)
}

type File struct {
	Field       string
	Name        string
	Data        []byte
	ContentType string // 空なら application/octet-stream
}

func (c *Client) PostMultipart(path string, form url.Values, files ...File) *Response {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, vs := range form {
		for _, v := range vs {
			mw.WriteField(k, v)
		}
	}
	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, f.Field, f.Name))
		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		w, err := mw.CreatePart(h)
		if err != nil {
			c.t.Fatal(err)
		}
		w.Write(f.Data)
	}
	mw.Close()
	req, err := http.NewRequest(http.MethodPost, c.URL(path), &buf)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.Do(req)
}

// Login はパスワードでログインする. ログイン後のリダイレクト先を返す.
func (c *Client) Login(mail, password string) string {
	c.t.Helper()
	res := c.PostForm("/login/password", url.Values{"email": {mail}, "password": {password}})
	if res.Status != http.StatusFound {
		c.t.Fatalf("login %s failed: status=%d\n%s", mail, res.Status, res.Body)
	}
	return res.Location()
}

var hiddenTokenRe = regexp.MustCompile(`name="token"\s+value="([0-9A-Za-z]+)"`)

// HiddenToken はページ内の <input name="token" value="..."> の値を返す (削除確認の CSRF トークン).
func HiddenToken(t testing.TB, res *Response) string {
	t.Helper()
	m := hiddenTokenRe.FindSubmatch(res.Body)
	if m == nil {
		t.Fatalf("token not found in %s\n%s", res.URL, res.Body)
	}
	return string(m[1])
}

// ExpectStatus はステータスコードを確認する.
func ExpectStatus(t testing.TB, res *Response, want int) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("%s: status=%d, want %d\n%s", res.URL, res.Status, want, truncate(res.Body, 2000))
	}
}

// ExpectRedirect は 302 で want (パス or 絶対 URL の前方一致) にリダイレクトすることを確認する.
func (c *Client) ExpectRedirect(res *Response, want string) {
	c.t.Helper()
	ExpectStatus(c.t, res, http.StatusFound)
	loc := res.Location()
	if !strings.HasPrefix(loc, c.URL(want)) && !strings.HasPrefix(loc, want) {
		c.t.Fatalf("%s: redirect to %q, want %q", res.URL, loc, want)
	}
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "..."
	}
	return string(b)
}
