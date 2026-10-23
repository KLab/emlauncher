package emlauncher

import (
	"regexp"
	"testing"
)

func TestPassword(t *testing.T) {
	// PHP 版で作られたハッシュ (sha1 を手計算したものと同じ方式)
	known := hashWith("secret", "AbCdEfGh12345678", 12)
	if !CheckPassword(known, "secret") || CheckPassword(known, "Secret") || CheckPassword(known, "") {
		t.Error("CheckPassword with known hash")
	}
	h := HashPassword("パスワード")
	if !regexp.MustCompile(`^(1[0-9]|20):[0-9A-Za-z]{16}:[0-9a-f]{40}$`).MatchString(h) {
		t.Errorf("hash format: %q", h)
	}
	if !CheckPassword(h, "パスワード") {
		t.Error("CheckPassword with new hash")
	}
	for _, bad := range []string{"", "x", "a:b:c", "99999:salt:hash"} {
		if CheckPassword(bad, "secret") {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestCompilePHPRegexp(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		in      string
		want    bool
	}{
		{`/@klab\.com$/`, "a@klab.com", true},
		{`/@klab\.com$/`, "a@KLAB.com", false},
		{`/@klab\.com$/i`, "a@KLAB.com", true},
		{`#@example\.(com|jp)$#`, "b@example.jp", true},
		{`@example\.com$`, "c@example.com", true}, // 区切り文字無し
	} {
		re, err := CompilePHPRegexp(tc.pattern)
		if err != nil {
			t.Fatalf("%s: %v", tc.pattern, err)
		}
		if got := re.MatchString(tc.in); got != tc.want {
			t.Errorf("%s =~ %s: %v", tc.in, tc.pattern, got)
		}
	}
	if _, err := CompilePHPRegexp(`/a/x`); err == nil {
		t.Error("unsupported flag accepted")
	}
}
