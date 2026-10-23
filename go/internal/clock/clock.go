// Package clock は DB の DATETIME と同じ表現 (設定した TZ のローカル時刻, 秒単位) で時刻を扱う.
//
// PHP 版は date('Y-m-d H:i:s') で書き込んでいる. Go 版も同じ TZ・同じ精度にしないと,
// 時刻がずれたり, 等値比較 (application.last_upload == package.created) が成り立たなくなる.
package clock

import "time"

// Layout は DATETIME の文字列表現.
const Layout = "2006-01-02 15:04:05"

type Clock struct {
	loc *time.Location
	now func() time.Time
}

func New(loc *time.Location) *Clock {
	return &Clock{loc: loc, now: time.Now}
}

// NewFixed はテスト用に固定時刻を返す Clock を作る.
func NewFixed(loc *time.Location, t time.Time) *Clock {
	return &Clock{loc: loc, now: func() time.Time { return t }}
}

func (c *Clock) Location() *time.Location { return c.loc }

// Now は秒未満を切り捨てた現在時刻を返す.
// MySQL 5.7 は DATETIME の小数秒を丸める (切り上げうる) ため, 書き込む前に必ず切り捨てる.
func (c *Clock) Now() time.Time {
	return c.now().In(c.loc).Truncate(time.Second)
}

// Format は DATETIME の文字列 (API の created など) にする.
func (c *Clock) Format(t time.Time) string {
	return t.In(c.loc).Format(Layout)
}

func (c *Clock) Parse(s string) (time.Time, error) {
	return time.ParseInLocation(Layout, s, c.loc)
}
