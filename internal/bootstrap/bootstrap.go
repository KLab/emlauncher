// Package bootstrap は設定から DB・ストレージ・memcached などを用意し, service.Service を作る.
// HTTP サーバとバッチで共通.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/KLab/emlauncher-go/internal/cache"
	"github.com/KLab/emlauncher-go/internal/config"
	"github.com/KLab/emlauncher-go/internal/mailer"
	"github.com/KLab/emlauncher-go/internal/pkgfile"
	"github.com/KLab/emlauncher-go/internal/service"
	"github.com/KLab/emlauncher-go/internal/storage"
)

// App は用意したもの. 使い終わったら Close する.
type App struct {
	Service    *service.Service
	DB         *sql.DB
	Storage    storage.Storage
	LocalFiles *storage.Local // storage_class が LocalFile のとき
	Bundletool *pkgfile.Bundletool
}

func (a *App) Close() error { return a.DB.Close() }

// New は設定から App を作る. DB には接続しない (起動時に DB へ書き込まないことは移行手順の前提).
func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	db, err := OpenDB(cfg)
	if err != nil {
		return nil, err
	}
	a := &App{DB: db}

	switch cfg.StorageClass {
	case config.StorageS3:
		s, err := storage.NewS3(ctx, storage.S3Config{
			Key: cfg.AWS.Key, Secret: cfg.AWS.Secret, Region: cfg.AWS.Region, Bucket: cfg.AWS.BucketName,
			BaseURL: cfg.AWS.BaseURL, ExternalURL: cfg.AWS.ExternalURL, Logger: log,
		})
		if err != nil {
			db.Close()
			return nil, err
		}
		a.Storage = s
	case config.StorageLocalFile:
		l := storage.NewLocal(cfg.LocalFile.Path, cfg.LocalFile.URLPrefix)
		a.Storage = l
		if cfg.LocalFile.Serve {
			a.LocalFiles = l
		}
	}

	send, err := mailer.NewSendmail(cfg.Mail.Sendmail)
	if err != nil {
		db.Close()
		return nil, err
	}
	a.Bundletool = &pkgfile.Bundletool{
		Java:     strings.Fields(cfg.APKFile.Java),
		Jar:      cfg.APKFile.Bundletool,
		Keystore: cfg.APKFile.Keystore,
		KSPass:   cfg.APKFile.KSPass,
		KeyAlias: cfg.APKFile.KeyAlias,
		KeyPass:  cfg.APKFile.KeyPass,
		AAPT2:    cfg.APKFile.AAPT2,

		MaxConcurrency: cfg.APKFile.MaxConcurrency,
		Timeout:        time.Duration(cfg.APKFile.Timeout) * time.Second,
	}
	a.Service = service.New(service.Deps{
		DB:         db,
		Storage:    a.Storage,
		Cache:      cache.New(cfg.Memcache.Addr(), cfg.CachePrefix),
		Mailer:     send,
		Bundletool: a.Bundletool,
		Config:     cfg,
		Logger:     log,
	})
	return a, nil
}

// OpenDB は MySQL への接続 (プール) を作る. DATETIME は設定の TZ のローカル時刻として読み書きする.
func OpenDB(cfg *config.Config) (*sql.DB, error) {
	mc := mysql.NewConfig()
	mc.Net = "tcp"
	mc.Addr = fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)
	mc.User = cfg.Database.User
	mc.Passwd = cfg.Database.Password
	mc.DBName = cfg.Database.Name
	mc.ParseTime = true
	mc.Loc = cfg.Location
	mc.Params = map[string]string{"charset": "utf8mb4"}
	mc.Timeout = 10 * time.Second
	connector, err := mysql.NewConnector(mc)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	// 再起動中は 2 プロセス分になるので max_connections (既定 151) の半分未満にする
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)                 // go-sql-driver/mysql の推奨 (MaxOpenConns と同じ)
	db.SetConnMaxLifetime(3 * time.Minute) // go-sql-driver/mysql の推奨 (5 分未満)
	return db, nil
}
