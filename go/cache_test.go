package emlauncher

import (
	"os"
	"testing"
	"time"
)

func TestExpiration(t *testing.T) {
	c := &Cache{now: func() time.Time { return time.Unix(1_800_000_000, 0) }}
	for _, tc := range []struct {
		ttl  time.Duration
		want int32
	}{
		{0, 0},
		{60 * time.Second, 60},
		{1500 * time.Millisecond, 2},
		{30 * 24 * time.Hour, 30 * 24 * 3600},
		{31 * 24 * time.Hour, 1_800_000_000 + 31*24*3600},
	} {
		if got := c.expiration(tc.ttl); got != tc.want {
			t.Errorf("expiration(%v) = %d, want %d", tc.ttl, got, tc.want)
		}
	}
}

// E2E 環境の memcached (e2e/env/up.sh) を使う.
func TestGetSetDelete(t *testing.T) {
	host := os.Getenv("EML_TEST_MEMCACHED_HOST")
	if host == "" {
		t.Skip("EML_TEST_MEMCACHED_HOST is not set")
	}
	c := NewCache(host, 21211, "gotest_")
	key := "k_" + randomString(8)
	if _, ok, err := c.Get(key); ok || err != nil {
		t.Fatalf("Get before Set: ok=%v err=%v", ok, err)
	}
	if err := c.Set(key, []byte(`{"a":1}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	v, ok, err := c.Get(key)
	if !ok || err != nil || string(v) != `{"a":1}` {
		t.Fatalf("Get = %q %v %v", v, ok, err)
	}
	// 接頭辞付きのキーで保存されている
	if _, ok, _ := NewCache(host, 21211, "").Get("gotest_" + key); !ok {
		t.Error("key is not prefixed")
	}
	if err := c.Delete(key); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(key); err != nil {
		t.Fatalf("Delete missing key: %v", err)
	}
}
