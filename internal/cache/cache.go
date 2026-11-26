// Package cache は memcached に, 設定の接頭辞 (cache_prefix) を付けたキーで値を読み書きする.
//
// 値はバイト列のまま扱う. PHP 版 (php-memcached) と値を共有するため, 文字列以外の形式
// (PHP のシリアライズなど) は使わない.
package cache

import (
	"errors"
	"math"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// ErrNotFound はキーが無い (期限切れを含む) ことを表す.
var ErrNotFound = errors.New("cache: not found")

// maxRelativeExpiration を超える有効期限は, memcached の仕様で UNIX 時刻として渡す.
const maxRelativeExpiration = 30 * 24 * time.Hour

type Cache struct {
	client *memcache.Client
	prefix string
	now    func() time.Time
}

func New(addr, prefix string) *Cache {
	c := memcache.New(addr)
	c.Timeout = 2 * time.Second
	return &Cache{client: c, prefix: prefix, now: time.Now}
}

// Get は key の値を返す. 無ければ ErrNotFound.
func (c *Cache) Get(key string) ([]byte, error) {
	it, err := c.client.Get(c.prefix + key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return it.Value, nil
}

// Set は ttl の間有効な値を書く.
func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	return c.client.Set(&memcache.Item{Key: c.prefix + key, Value: value, Expiration: c.expiration(ttl)})
}

// Delete は key を消す. 無くてもエラーにしない.
func (c *Cache) Delete(key string) error {
	err := c.client.Delete(c.prefix + key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil
	}
	return err
}

// expiration は memcached に渡す有効期限. 30 日以下は秒数, それを超えるものは UNIX 時刻.
// memcached の有効期限は 32 ビットなので, 2038 年以降になるものはその上限にする.
func (c *Cache) expiration(ttl time.Duration) int32 {
	secs := int64((ttl + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	if ttl > maxRelativeExpiration {
		return int32(min(c.now().Unix()+secs, math.MaxInt32))
	}
	return int32(secs)
}
