package harness

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	_ "github.com/go-sql-driver/mysql"
)

// Env は 1 つのテストから見たテスト対象環境.
type Env struct {
	t       testing.TB
	Config  Config
	DB      *DB
	Storage Storage
	Mailbox *Mailbox
	MC      *memcache.Client
}

var (
	sharedOnce sync.Once
	sharedDB   *sql.DB
	sharedErr  error
)

func New(t testing.TB) *Env {
	t.Helper()
	cfg := LoadConfig()
	sharedOnce.Do(func() {
		sharedDB, sharedErr = sql.Open("mysql", cfg.DBDSN)
		if sharedErr == nil {
			sharedErr = sharedDB.Ping()
		}
	})
	if sharedErr != nil {
		t.Fatalf("db: %v", sharedErr)
	}
	e := &Env{
		t:       t,
		Config:  cfg,
		DB:      &DB{t: t, db: sharedDB},
		Mailbox: &Mailbox{t: t, dir: cfg.MailboxDir},
		MC:      memcache.New(cfg.Memcached),
	}
	e.Storage = newStorage(t, cfg)
	return e
}

var uniqueSeq atomic.Int64

// Unique はテスト間で衝突しない名前を返す.
func Unique(prefix string) string {
	return fmt.Sprintf("%s-%s%d", prefix, strconv.FormatInt(time.Now().UnixMilli(), 36), uniqueSeq.Add(1))
}

// QueryInt は URL (相対/絶対) のクエリパラメータを整数で返す.
func QueryInt(t testing.TB, rawURL, key string) int64 {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url %q: %v", rawURL, err)
	}
	v, err := strconv.ParseInt(u.Query().Get(key), 10, 64)
	if err != nil {
		t.Fatalf("%q has no integer %s", rawURL, key)
	}
	return v
}

// MemcacheGet は接頭辞を付けたキーで memcached から値を取得する. 無ければ nil.
func (e *Env) MemcacheGet(key string) []byte {
	e.t.Helper()
	it, err := e.MC.Get(e.Config.CachePrefix + key)
	if err == memcache.ErrCacheMiss {
		return nil
	}
	if err != nil {
		e.t.Fatalf("memcache get %s: %v", key, err)
	}
	return it.Value
}

// ParseLocalTime は DB の DATETIME 文字列をテスト対象の TZ で解釈する.
func (e *Env) ParseLocalTime(s string) time.Time {
	e.t.Helper()
	tm, err := time.ParseInLocation("2006-01-02 15:04:05", s, e.Config.TZ)
	if err != nil {
		e.t.Fatalf("parse datetime %q: %v", s, err)
	}
	return tm
}

// ExpectRecent は DATETIME 文字列が現在時刻の前後 2 分以内であることを確認する (TZ の確認).
func (e *Env) ExpectRecent(what, s string) {
	e.t.Helper()
	d := time.Since(e.ParseLocalTime(s))
	if d < -2*time.Minute || d > 2*time.Minute {
		e.t.Fatalf("%s=%q is not recent in %s (diff %v)", what, s, e.Config.TZ, d)
	}
}

// PHPBug は PHP 版の既知バグを踏むテストで使う. テスト対象が PHP 版なら true を返し,
// 呼び出し側はバグのある挙動 (現状) を確認する. Go 版では正しい挙動を確認する.
func (e *Env) PHPBug(description string) bool {
	e.t.Helper()
	if e.Config.Target == "php" {
		e.t.Logf("known PHP bug: %s", description)
		return true
	}
	return false
}
