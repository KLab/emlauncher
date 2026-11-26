package web

import (
	"net/http/httptest"
	"testing"

	"github.com/KLab/emlauncher-go/internal/config"
)

func TestRequestAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		paramEnabled bool
		url          string
		auth         string
		want         string
		ok           bool
	}{
		{"param", true, "/api/upload?api_key=p", "", "p", true},
		{"header", true, "/api/upload", "Bearer h", "h", true},
		{"header takes precedence", true, "/api/upload?api_key=p", "Bearer h", "h", true},
		{"scheme is case insensitive", true, "/api/upload", "bearer  h ", "h", true},
		{"not bearer", true, "/api/upload?api_key=p", "Basic xxx", "p", true},
		{"none", true, "/api/upload", "", "", false},
		{"param disabled", false, "/api/upload?api_key=p", "", "", false},
		{"param disabled, header", false, "/api/upload?api_key=p", "Bearer h", "h", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{cfg: &config.Config{APIKeyParamEnabled: tt.paramEnabled}}
			r := httptest.NewRequest("GET", tt.url, nil)
			if tt.auth != "" {
				r.Header.Set("Authorization", tt.auth)
			}
			got, ok := s.requestAPIKey(r)
			if got != tt.want || ok != tt.ok {
				t.Errorf("requestAPIKey = %q, %v, want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
