package web

import (
	"net/http/httptest"
	"testing"
)

func TestAbsURL(t *testing.T) {
	r := httptest.NewRequest("GET", "http://emlauncher.example.com/app?id=1", nil)
	if got := absURL(r, "/package?id=2", ""); got != "http://emlauncher.example.com/package?id=2" {
		t.Errorf("got %q", got)
	}
	if got := absURL(r, "/login", "https"); got != "https://emlauncher.example.com/login" {
		t.Errorf("got %q", got)
	}

	// ロードバランサ配下
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "public.example.com")
	if got := absURL(r, "/", ""); got != "https://public.example.com/" {
		t.Errorf("got %q", got)
	}
	if got := requestURL(r); got != "https://public.example.com/app?id=1" {
		t.Errorf("got %q", got)
	}
}

func TestRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	if got := remoteAddr(r); got != "10.0.0.1" {
		t.Errorf("got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.2")
	if got := remoteAddr(r); got != "203.0.113.5" {
		t.Errorf("got %q", got)
	}
}
