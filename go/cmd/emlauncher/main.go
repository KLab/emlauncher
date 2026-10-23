// emlauncher は EMLauncher (Go 版) の HTTP サーバ.
//
//	emlauncher -config /path/to/config.yml
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KLabTools/emlauncher/go/internal/cache"
	"github.com/KLabTools/emlauncher/go/internal/clock"
	"github.com/KLabTools/emlauncher/go/internal/config"
	"github.com/KLabTools/emlauncher/go/internal/db"
	"github.com/KLabTools/emlauncher/go/internal/storage"
	"github.com/KLabTools/emlauncher/go/internal/web"
)

func main() {
	configPath := flag.String("config", "config.yml", "config file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(*configPath, log); err != nil {
		log.Error("exit", "error", err)
		os.Exit(1)
	}
}

func run(configPath string, log *slog.Logger) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer d.Close()
	st, err := storage.New(ctx, cfg)
	if err != nil {
		return err
	}
	srv := &web.Server{
		Config:  cfg,
		DB:      d,
		Cache:   cache.New(cfg.Memcache.Host, cfg.Memcache.Port, cfg.CachePrefix),
		Storage: st,
		Clock:   clock.New(cfg.Location()),
		Log:     log,
	}
	hs := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 30 * time.Second,
		// 大きなパッケージのアップロードがあるので, 本文の読み込みには制限を設けない.
		IdleTimeout: 120 * time.Second,
	}
	log.Info("start", "listen", cfg.Listen, "timezone", cfg.Location().String(),
		"storage", cfg.StorageClass, "cache_prefix", cfg.CachePrefix)

	errc := make(chan error, 1)
	go func() { errc <- hs.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := hs.Shutdown(sctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
