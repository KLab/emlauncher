// Package storage はパッケージやアイコンの保存先 (S3 / LocalFile) を扱う.
//
// キーの構成, S3 の ACL と ContentType は PHP 版と同じ:
//   - アイコン: public-read, image/png
//   - パッケージ・添付ファイル・一時ファイル: private, application/octet-stream
//     (PHP 版はコード上 MIME を指定しているが SDK に無視され, 実際はすべて octet-stream になっている)
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// storage_class の値 (PHP 版と同じ).
const (
	ClassS3        = "S3"
	ClassLocalFile = "LocalFile"
)

// S3Config は S3 の設定 (設定ファイルの aws).
type S3Config struct {
	// Key / Secret が空なら IAM ロールなど既定の認証情報を使う.
	Key        string `yaml:"key"`
	Secret     string `yaml:"secret"`
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
	// BaseURL は S3 互換ストレージ (LocalStack など) のエンドポイント. AWS の S3 では指定しない.
	BaseURL string `yaml:"base_url"`
	// ExternalURL はブラウザから見たエンドポイント (BaseURL と同じなら省略).
	ExternalURL string `yaml:"external_url"`
}

// LocalConfig は LocalFile の設定 (設定ファイルの local_file).
type LocalConfig struct {
	Path      string `yaml:"path"`
	URLPrefix string `yaml:"url_prefix"`
	// Serve が true なら Go 版が URLPrefix 以下のファイルを配信する.
	// false なら従来どおり Apache の Alias などに任せる.
	Serve bool `yaml:"serve"`
}

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

// New は storage_class に応じたストレージを作る.
func New(ctx context.Context, class string, s3 S3Config, local LocalConfig) (Storage, error) {
	switch class {
	case ClassS3:
		return NewS3(ctx, s3)
	case ClassLocalFile:
		return NewLocal(local), nil
	}
	return nil, fmt.Errorf("unknown storage_class %q", class)
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

// randomName は一時ファイル名などに使う推測しにくい名前.
func randomName() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}
