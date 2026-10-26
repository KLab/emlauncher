// emlauncher は EMLauncher (Go 版) の HTTP サーバ.
//
//	emlauncher -config /path/to/config.yml
package main

import (
	"context"
	"emlauncher"
	"emlauncher/pkgfile"
	"emlauncher/storage"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // tzdata の無い環境でも timezone を指定できるように
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
	cfg, err := emlauncher.LoadConfig(configPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d, err := emlauncher.OpenDB(cfg)
	if err != nil {
		return err
	}
	defer d.Close()
	st, err := storage.New(ctx, cfg.StorageClass, cfg.AWS, cfg.LocalFile)
	if err != nil {
		return err
	}
	srv := &emlauncher.Server{
		Config: cfg,
		Core: &emlauncher.Core{
			Config:     cfg,
			DB:         d,
			Storage:    st,
			Clock:      emlauncher.NewClock(cfg.Location()),
			Cache:      emlauncher.NewCache(cfg.Memcache.Host, cfg.Memcache.Port, cfg.CachePrefix),
			Mail:       &emlauncher.Sendmail{Command: cfg.Mail.Sendmail},
			Bundletool: pkgfile.NewBundletool(cfg.APKFile),
			Log:        log,
		},
		Log: log,
	}
	if err := srv.Init(); err != nil {
		return err
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
