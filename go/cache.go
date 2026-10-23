// memcached を, PHP 版 (mfwMemcache) と同じキー接頭辞で使う.
// 値は文字列 (バイト列) のみ扱う. PHP 版と共有するキーでは, PHP 版も文字列で読み書きしていること.

package emlauncher

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// memcached は 30 日を超える有効期限を UNIX 時刻として解釈する.
const maxRelativeExpiration = 30 * 24 * time.Hour

type Cache struct {
	mc     *memcache.Client
	prefix string
	now    func() time.Time
}

func NewCache(host string, port int, prefix string) *Cache {
	return &Cache{
		mc:     memcache.New(net.JoinHostPort(host, strconv.Itoa(port))),
		prefix: prefix,
		now:    time.Now,
	}
}

// Get は値を返す. 無ければ ok=false.
func (c *Cache) Get(key string) (value []byte, ok bool, err error) {
	it, err := c.mc.Get(c.prefix + key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return it.Value, true, nil
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	return c.mc.Set(&memcache.Item{Key: c.prefix + key, Value: value, Expiration: c.expiration(ttl)})
}

func (c *Cache) Delete(key string) error {
	err := c.mc.Delete(c.prefix + key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil
	}
	return err
}

func (c *Cache) expiration(ttl time.Duration) int32 {
	if ttl <= 0 {
		return 0
	}
	if ttl > maxRelativeExpiration {
		// PHP 版は秒数をそのまま渡すため, 30 日を超えるトークンは即座に失効していた.
		return int32(c.now().Add(ttl).Unix())
	}
	return int32((ttl + time.Second - 1) / time.Second)
}
