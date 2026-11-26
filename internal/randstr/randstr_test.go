package randstr

import (
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-zA-Z]{32}$`)
	seen := map[string]bool{}
	for range 100 {
		s := New(32)
		if !re.MatchString(s) {
			t.Fatalf("New(32) = %q", s)
		}
		if seen[s] {
			t.Fatalf("duplicated %q", s)
		}
		seen[s] = true
	}
	if New(0) != "" {
		t.Fatal("New(0) is not empty")
	}
}
