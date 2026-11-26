package cache

import (
	"math"
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

func TestExpirationOverflow(t *testing.T) {
	c := &Cache{now: func() time.Time { return time.Unix(2_100_000_000, 0) }}
	if got := c.expiration(10 * 366 * 24 * time.Hour); got != math.MaxInt32 {
		t.Errorf("expiration = %d, want MaxInt32", got)
	}
	if got := c.expiration(time.Duration(math.MaxInt64)); got <= 0 {
		t.Errorf("expiration of a huge ttl = %d", got)
	}
}
