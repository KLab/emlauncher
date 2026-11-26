// Package storage はパッケージやアイコンの保存先 (Amazon S3 / ローカルディスク).
//
// キーの構成 (app-icons/..., package/..., temp-data/...) は呼び出し側が決める.
// S3 の ACL と ContentType は PHP 版で保存されたオブジェクトと揃えている
// (アイコンは public-read / image/png, それ以外は private / application/octet-stream).
package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound はキーが存在しないことを表す.
var ErrNotFound = errors.New("storage: not found")

type Storage interface {
	// PutIcon はアイコン (PNG) を公開で保存する.
	PutIcon(ctx context.Context, key string, png []byte) error
	// PutFile はローカルのファイルを非公開で保存する. 大きなファイル (数 GB) でもよい.
	PutFile(ctx context.Context, key, path string) error
	// Move は src を dst に移す (一時ファイルを正式なキーにする).
	Move(ctx context.Context, src, dst string) error
	// Delete はキーを消す. 存在しなくてもエラーにしない.
	Delete(ctx context.Context, key string) error
	// Size はキーの大きさを返す. 無ければ ErrNotFound.
	Size(ctx context.Context, key string) (int64, error)
	// PublicURL は PutIcon で保存したものの URL (期限なし).
	PublicURL(key string) string
	// DownloadURL は expires の間ダウンロードできる URL. fileName はダウンロード時のファイル名.
	DownloadURL(ctx context.Context, key string, expires time.Duration, fileName string) (string, error)
}

// contentDisposition はダウンロード時のファイル名を RFC 5987 の形式で指定する.
func contentDisposition(fileName string) string {
	return "attachment; filename*=UTF-8''" + rfc5987(fileName)
}

// rfc5987 は英数字と -._~ 以外を %XX にする (url.PathEscape と違い空白も %20, + も %2B).
func rfc5987(s string) string {
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
