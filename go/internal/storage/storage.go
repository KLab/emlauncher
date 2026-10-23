// Package storage はパッケージやアイコンの保存先 (S3 / LocalFile) を扱う.
//
// キーの構成, S3 の ACL と ContentType は PHP 版と同じ:
//   - アイコン: public-read, image/png
//   - パッケージ・添付ファイル・一時ファイル: private, application/octet-stream
//     (PHP 版はコード上 MIME を指定しているが SDK に無視され, 実際はすべて octet-stream になっている)
package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"emlauncher/internal/config"
)

// FileContentType はパッケージ類を保存するときの ContentType.
const FileContentType = "application/octet-stream"

type Storage interface {
	// SaveIcon はアイコン (PNG) を公開状態で保存する.
	SaveIcon(ctx context.Context, key string, png []byte) error
	// SaveFile はローカルファイルを非公開で保存する.
	SaveFile(ctx context.Context, key, srcPath string) error
	// Rename はキーを変更する (一時ファイルを正式なキーへ移す).
	Rename(ctx context.Context, srcKey, dstKey string) error
	Delete(ctx context.Context, key string) error
	// PublicURL は公開オブジェクト (アイコン) の URL.
	PublicURL(key string) string
	// DownloadURL は期限付きのダウンロード URL. fileName でダウンロードさせる.
	// LocalFile では期限とファイル名は使われない (PHP 版と同じ).
	DownloadURL(ctx context.Context, key string, expires time.Duration, fileName string) (string, error)
}

func New(ctx context.Context, c *config.Config) (Storage, error) {
	switch c.StorageClass {
	case config.StorageS3:
		return NewS3(ctx, c.AWS)
	case config.StorageLocalFile:
		return NewLocal(c.LocalFile), nil
	}
	return nil, fmt.Errorf("unknown storage_class %q", c.StorageClass)
}

// ContentDisposition はダウンロード時のファイル名を RFC 5987 の形式で指定する.
// PHP 版は urlencode() を使っており, 空白が "+" になっていた.
func ContentDisposition(fileName string) string {
	return "attachment; filename*=UTF-8''" + encodeRFC5987(fileName)
}

func encodeRFC5987(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-._~", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// escapeKey は URL のパスに使えるようキーの各セグメントをエスケープする.
func escapeKey(key string) string {
	segs := strings.Split(key, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}
