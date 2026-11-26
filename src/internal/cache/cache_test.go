package cache

import (
	"testing"
	"time"
)

func TestExpiration(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	c := &Cache{now: func() time.Time { return now }}
	for _, tc := range []struct {
		ttl  time.Duration
		want int32
	}{
		{60 * time.Second, 60},
		{1500 * time.Millisecond, 2},
		{0, 1},
		{30 * 24 * time.Hour, 30 * 24 * 3600},
		{31 * 24 * time.Hour, int32(now.Unix()) + 31*24*3600},
	} {
		if got := c.expiration(tc.ttl); got != tc.want {
			t.Errorf("expiration(%v) = %d, want %d", tc.ttl, got, tc.want)
		}
	}
}
