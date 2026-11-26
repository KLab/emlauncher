package web

import (
	"bytes"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/KLab/emlauncher-go/internal/config"
)

func TestLogURI(t *testing.T) {
	for in, want := range map[string]string{
		"/app?id=1":                                  "/app?id=1",
		"/api/upload?api_key=secret&id=1":            "/api/upload?api_key=***&id=1",
		"/package/install?token=secret":              "/package/install?token=***",
		"/login/password_reset?key=secret":           "/login/password_reset?key=***",
		"/package/install_plist?id=3&t=secret":       "/package/install_plist?id=3&t=***",
		"/login/password?email=a&password=secret":    "/login/password?email=a&password=***",
		"/login/google_callback?code=secret&state=s": "/login/google_callback?code=***&state=s",
		"/top": "/top",
	} {
		u, err := url.Parse(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := logURI(u); got != want {
			t.Errorf("logURI(%q) = %q, want %q", in, got, want)
		}
	}
}

// panic したリクエストは 500 を返し, エラー (スタック付き) とアクセスログの両方に残る.
func TestAccessLogRecoversPanic(t *testing.T) {
	var buf bytes.Buffer
	tmpl, _ := fs.Sub(Templates, "templates")
	static, _ := fs.Sub(Static, "static")
	s, err := New(Options{
		Config:    &config.Config{Session: config.Session{Secret: "0123456789abcdef"}, Location: time.UTC},
		Templates: tmpl, Static: static,
		Logger: slog.New(slog.NewTextHandler(&buf, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	h := s.accessLog(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/app?id=1&token=secret", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	log := buf.String()
	for _, want := range []string{`msg=panic`, `panic=boom`, `stack=`, `msg=access`, `status=500`} {
		if !strings.Contains(log, want) {
			t.Errorf("log does not contain %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "secret") {
		t.Errorf("log contains the token:\n%s", log)
	}
}
