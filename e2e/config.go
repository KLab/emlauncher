// Package e2e はシナリオテストから EMLauncher (PHP 版 / Go 版) を操作し,
// DB・ストレージ・メールの状態を確認するための道具をまとめたもの.
//
// テスト対象は環境変数で指定する. 既定値は e2e/env/up.sh で起動する PHP 版の環境.
package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Target      string // E2E_TARGET: php | go. PHP 版の既知バグの扱いを切り替える
	BaseURL     string // E2E_BASE_URL
	DBDSN       string // E2E_DB_DSN
	Storage     string // E2E_STORAGE: local | s3
	LocalDir    string // E2E_LOCAL_DIR: LocalFile の保存先 (ホスト側のパス)
	S3Endpoint  string // E2E_S3_ENDPOINT
	S3Bucket    string // E2E_S3_BUCKET
	MailboxDir  string // E2E_MAILBOX: 送信メールが 1 通 1 ファイルで置かれるディレクトリ
	Memcached   string // E2E_MEMCACHED
	CachePrefix string // E2E_CACHE_PREFIX: memcached のキー接頭辞
	BatchCmd    string // E2E_BATCH_CMD: 古いパッケージ削除バッチ. {keep} が保持数に置き換わる
	GoExec      string // E2E_GO_EXEC: Go 版のコンテナでコマンドを実行する (無停止の再起動のテスト用)
	GoConfig    string // E2E_GO_CONFIG: Go 版が読んでいる設定ファイル (ホスト側のパス)
	TZ          *time.Location
}

func LoadConfig() Config {
	root := e2eRoot()
	storage := env("E2E_STORAGE", "local")
	tzName := env("E2E_TZ", "Asia/Tokyo")
	tz, err := time.LoadLocation(tzName)
	if err != nil {
		panic("E2E_TZ: " + err.Error())
	}
	compose := filepath.Join(root, "env", "compose.yml")
	target := env("E2E_TARGET", "php")
	baseURL := "http://localhost:18081"
	batch := "docker compose -f " + compose + " exec -T php php /srv/emlauncher/batch/delete_old_packages.php -e e2e_" + storage + " -k {keep}"
	if target == "go" {
		baseURL = "http://localhost:18082"
		batch = "docker compose -f " + compose + " exec -T go delete-old-packages -config /etc/emlauncher/config.e2e_" + storage + ".yml -k {keep}"
	}
	return Config{
		Target:      target,
		BaseURL:     strings.TrimRight(env("E2E_BASE_URL", baseURL), "/"),
		DBDSN:       env("E2E_DB_DSN", "emlauncher:emlauncher@tcp(localhost:13306)/emlauncher?charset=utf8mb4"),
		Storage:     storage,
		LocalDir:    env("E2E_LOCAL_DIR", filepath.Join(root, ".work", "files")),
		S3Endpoint:  env("E2E_S3_ENDPOINT", "http://localhost:14566"),
		S3Bucket:    env("E2E_S3_BUCKET", "emlauncher-e2e"),
		MailboxDir:  env("E2E_MAILBOX", filepath.Join(root, ".work", "mailbox")),
		Memcached:   env("E2E_MEMCACHED", "localhost:21211"),
		CachePrefix: env("E2E_CACHE_PREFIX", "emlauncher_e2e_"+storage+"_emlauncher_"),
		BatchCmd:    env("E2E_BATCH_CMD", batch),
		GoExec:      env("E2E_GO_EXEC", "docker compose -f "+compose+" exec -T go"),
		GoConfig:    env("E2E_GO_CONFIG", filepath.Join(root, ".work", "go-runtime-config", "config.e2e_"+storage+".yml")),
		TZ:          tz,
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// e2eRoot は e2e モジュールのディレクトリを返す.
func e2eRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}
