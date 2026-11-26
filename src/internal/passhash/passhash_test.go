package passhash

import (
	"regexp"
	"testing"
)

func TestNewAndCheck(t *testing.T) {
	h := New("secret-パス")
	if !regexp.MustCompile(`^(1[0-9]|20):[0-9A-Za-z]{16}:[0-9a-f]{40}$`).MatchString(h) {
		t.Fatalf("New = %q", h)
	}
	if !Check(h, "secret-パス") {
		t.Fatal("Check failed for the right password")
	}
	if Check(h, "wrong") || Check(h, "") {
		t.Fatal("Check succeeded for a wrong password")
	}
}

// PHP 版 (model/UserPass.php) と同じ方式で作ったハッシュ (別の実装で計算した値).
func TestCheckPHPHash(t *testing.T) {
	const h = "12:AbCdEfGh12345678:6931da5244b2c227a0a4a573f34b14397f68fe7e"
	if !Check(h, "pw") {
		t.Fatal("Check failed")
	}
	for _, bad := range []string{"", "x", "a:b:c", "-1:s:h", "999999999:s:h"} {
		if Check(bad, "pw") {
			t.Errorf("Check(%q) succeeded", bad)
		}
	}
}
