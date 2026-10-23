package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func roundTrip(t *testing.T, s *Store, sess *Session) *http.Request {
	t.Helper()
	sess.changed = true
	rec := httptest.NewRecorder()
	s.Save(rec, httptest.NewRequest("GET", "/", nil), sess)
	r := httptest.NewRequest("GET", "/", nil)
	for _, c := range rec.Result().Cookies() {
		r.AddCookie(c)
	}
	return r
}

func TestSaveLoad(t *testing.T) {
	s, err := NewStore("0123456789abcdef-secret", func(*http.Request) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	sess := s.Load(httptest.NewRequest("GET", "/", nil))
	sess.Set("login_user", "user@example.com")
	got := s.Load(roundTrip(t, s, sess))
	if got.Get("login_user") != "user@example.com" {
		t.Errorf("got %q", got.Get("login_user"))
	}

	// 別の鍵では読めない
	other, _ := NewStore("another-secret-0123456789", func(*http.Request) bool { return false })
	if other.Load(roundTrip(t, s, sess)).Get("login_user") != "" {
		t.Error("loaded with a wrong key")
	}

	// 改ざん
	r := roundTrip(t, s, sess)
	c, _ := r.Cookie(cookieName)
	tampered := httptest.NewRequest("GET", "/", nil)
	data, _, _ := strings.Cut(c.Value, ".")
	tampered.AddCookie(&http.Cookie{Name: cookieName, Value: data + "x." + s.sign(data)})
	if s.Load(tampered).Get("login_user") != "" {
		t.Error("loaded a tampered cookie")
	}

	// 期限切れ
	s.now = func() time.Time { return time.Now().Add(-maxAge - time.Hour) }
	r = roundTrip(t, s, sess)
	s.now = time.Now
	if s.Load(r).Get("login_user") != "" {
		t.Error("loaded an expired session")
	}
}

func TestClear(t *testing.T) {
	s, _ := NewStore("0123456789abcdef-secret", func(*http.Request) bool { return true })
	sess := s.Load(httptest.NewRequest("GET", "/", nil))
	sess.Set("a", "1")
	sess.Clear()
	rec := httptest.NewRecorder()
	s.Save(rec, httptest.NewRequest("GET", "/", nil), sess)
	c := rec.Result().Cookies()[0]
	if c.MaxAge >= 0 || !c.Secure || !c.HttpOnly {
		t.Errorf("cookie = %+v", c)
	}
	if NewStoreErr := func() error { _, err := NewStore("short", nil); return err }(); NewStoreErr == nil {
		t.Error("short secret accepted")
	}
}
