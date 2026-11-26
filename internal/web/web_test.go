package web

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/KLab/emlauncher-go/internal/config"
)

func TestFormatSize(t *testing.T) {
	for in, want := range map[int64]string{
		0: "0 B", 665: "665 B", 1024: "1 KB", 1536: "1.5 KB", 1339565: "1.3 MB", 7 << 30: "7 GB",
	} {
		if got := formatSize(in); got != want {
			t.Errorf("formatSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestModuleAction(t *testing.T) {
	for in, want := range map[string][2]string{
		"/": {"top", "index"}, "/app": {"app", "index"}, "/app/": {"app", "index"},
		"/package/delete_confirm": {"package", "delete_confirm"}, "/login/google_error": {"login", "google_error"},
	} {
		m, a := moduleAction(in)
		if [2]string{m, a} != want {
			t.Errorf("moduleAction(%q) = %s, %s", in, m, a)
		}
	}
}

func TestPaging(t *testing.T) {
	render := func(v pagingView) string {
		var b strings.Builder
		for _, it := range v.Items {
			switch {
			case it.Omission:
				b.WriteString(".. ")
			case it.Current:
				b.WriteString("[" + strconv.Itoa(it.Num) + "] ")
			default:
				b.WriteString(strconv.Itoa(it.Num) + " ")
			}
		}
		return strings.TrimSpace(b.String())
	}
	url := func(int) string { return "" }
	for _, tc := range []struct {
		cur, max int
		want     string
	}{
		{1, 0, ""},
		{1, 1, "[1]"},
		{3, 7, "1 2 [3] 4 5 6 7"},
		{1, 20, "[1] 2 3 4 5 6 .. 20"},
		{10, 20, "1 .. 8 9 [10] 11 12 .. 20"},
		{19, 20, "1 .. 15 16 17 18 [19] 20"},
		{9999, 3, "1 2 3"},
	} {
		if got := render(paging(tc.cur, tc.max, url)); got != tc.want {
			t.Errorf("paging(%d, %d) = %q, want %q", tc.cur, tc.max, got, tc.want)
		}
	}
}

func TestUserAgent(t *testing.T) {
	for ua, want := range map[string][2]bool{
		"Mozilla/5.0 (Linux; Android 13; Pixel 7) Chrome/116":           {true, false},
		"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) Safari": {false, true},
		"Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) Safari":          {false, true},
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/116":          {false, false},
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("User-Agent", ua)
		if got := [2]bool{isAndroid(r), isIOS(r)}; got != want {
			t.Errorf("%s: android, ios = %v, want %v", ua, got, want)
		}
	}
}

func TestBaseURL(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://internal:8080/app?id=1", nil)
	if got := baseURL(r); got != "http://internal:8080" {
		t.Errorf("baseURL = %q", got)
	}
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "emlauncher.example.com")
	if got := requestURL(r); got != "https://emlauncher.example.com/app?id=1" {
		t.Errorf("requestURL = %q", got)
	}
}

func TestSession(t *testing.T) {
	c := newSessionCodec("0123456789abcdef")
	now := time.Unix(1_700_000_000, 0)
	c.now = func() time.Time { return now }
	v := c.encode(&session{Mail: "a@example.com", Issued: now.Unix()})
	s, ok := c.decode(v)
	if !ok || s.Mail != "a@example.com" {
		t.Fatalf("decode = %+v, %v", s, ok)
	}
	if _, ok := newSessionCodec("another-secret-xx").decode(v); ok {
		t.Fatal("decoded with another secret")
	}
	if _, ok := c.decode(strings.Replace(v, ".", "x.", 1)); ok {
		t.Fatal("decoded a tampered value")
	}
	now = now.Add(sessionLifetime + time.Hour)
	if _, ok := c.decode(v); ok {
		t.Fatal("decoded an expired session")
	}
}

// multipart のファイルは一時ファイルに書かれ, cleanup で消える.
func TestParseFormMultipart(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("title", "t")
	mw.WriteField("tags[]", "a")
	mw.WriteField("tags[]", "b")
	fw, _ := mw.CreateFormFile("file", `C:\dir\my app.apk`)
	data := bytes.Repeat([]byte("x"), 1<<20)
	fw.Write(data)
	mw.CreateFormFile("empty", "") // ファイルが選ばれていない
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/api/upload?api_key=k&title=query", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	f, err := parseForm(httptest.NewRecorder(), r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Get("api_key") != "k" || f.Get("title") != "t" || len(f.Values["tags[]"]) != 2 {
		t.Errorf("values = %v", f.Values)
	}
	file := f.file("file")
	if file == nil || file.Name != "my app.apk" || file.Size != int64(len(data)) {
		t.Fatalf("file = %+v", file)
	}
	if f.file("empty") != nil || f.has("empty") {
		t.Error("empty file input is not ignored")
	}
	b, err := os.ReadFile(file.Path)
	if err != nil || !bytes.Equal(b, data) {
		t.Fatalf("temp file: %v", err)
	}
	f.cleanup()
	if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
		t.Fatalf("temp file remains: %v", err)
	}
}

// テンプレートがすべて読める.
func TestLoadTemplates(t *testing.T) {
	sub, err := fs.Sub(Templates, "templates")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := loadTemplates(sub)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"top/index", "app/index", "package/index", "doc/api_upload", "login/oauth_error", errorTemplate} {
		if ts[name] == nil {
			t.Errorf("template %s not loaded", name)
		}
	}
}

func TestPHPInt(t *testing.T) {
	for in, want := range map[string]int64{"10": 10, "10abc": 10, "abc": 0, "": 0, " 7": 7, "-5": -5, "+3x": 3, "-": 0} {
		if got := phpInt(in); got != want {
			t.Errorf("phpInt(%q) = %d, want %d", in, got, want)
		}
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	tmpl, _ := fs.Sub(Templates, "templates")
	static, _ := fs.Sub(Static, "static")
	s, err := New(Options{
		Config:    &config.Config{Session: config.Session{Secret: "0123456789abcdef"}, Location: time.UTC},
		Templates: tmpl, Static: static,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s.Handler()
}

// URL の形は PHP 版 (PATH_INFO での振り分け) と同じく揃える. メソッドは問わない.
func TestRouting(t *testing.T) {
	h := newTestServer(t)
	do := func(method, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w
	}
	if w := do(http.MethodHead, "/health"); w.Code != http.StatusOK {
		t.Errorf("HEAD /health: %d", w.Code)
	}
	for _, p := range []string{"/no_such_module", "/top/no_such", "/doc"} {
		w := do(http.MethodGet, p)
		if w.Code != http.StatusNotFound || w.Body.String() != "404 Not Found" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
			t.Errorf("GET %s: %d %q %q", p, w.Code, w.Body.String(), w.Header().Get("Content-Type"))
		}
	}
	// ログインが必要な画面はログイン画面へ (どの形の URL でも同じ画面)
	for _, p := range []string{"/", "/top", "/top/index", "/app/", "/app/index/foo", "/app/comment/", "/package/delete"} {
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodHead} {
			if w := do(m, p); w.Code != http.StatusFound || !strings.HasSuffix(w.Header().Get("Location"), "/login") {
				t.Errorf("%s %s: %d %s", m, p, w.Code, w.Header().Get("Location"))
			}
		}
	}
	if w := do(http.MethodPost, "/ajax/upload_package_temporary/x"); w.Code != http.StatusForbidden {
		t.Errorf("ajax without login: %d", w.Code)
	}
}
