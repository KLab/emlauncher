package clock

import (
	"testing"
	"time"
)

func TestNowTruncatesAndUsesLocation(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	c := NewFixed(tokyo, time.Date(2026, 1, 2, 15, 4, 5, 999_000_000, time.UTC))
	now := c.Now()
	if got := c.Format(now); got != "2026-01-03 00:04:05" {
		t.Errorf("Format(Now()) = %q", got)
	}
	if now.Nanosecond() != 0 {
		t.Errorf("not truncated: %v", now)
	}
	p, err := c.Parse("2026-01-03 00:04:05")
	if err != nil || !p.Equal(now) {
		t.Errorf("Parse = %v, %v", p, err)
	}
}
