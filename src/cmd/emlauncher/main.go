// emlauncher は EMLauncher (Go 版) の HTTP サーバ.
//
//	emlauncher -config /path/to/config.yml [-pid-file /run/emlauncher/emlauncher.pid]
//
// SIGHUP で無停止の再起動を行う: 新しいプロセスを起動して待ち受けソケットを引き継ぎ,
// 古いプロセスは処理中のリクエスト (長いアップロードを含む) を終えてから終了する.
// 新しいプロセスは設定ファイルを読み直すので, 設定の変更もこれで反映できる.
// 新しいプロセスが起動できなければ (設定の誤りなど), 古いプロセスがそのまま動き続ける.
//
// SIGTERM / SIGINT では新しい受け付けをやめ, 処理中のリクエストを終えてから終了する.
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
	_ "time/tzdata" // tzdata の無い環境でも timezone を指定できるように

	"github.com/cloudflare/tableflip"

	"emlauncher"
	"emlauncher/pkgfile"
	"emlauncher/storage"
)

func main() {
	configPath := flag.String("config", "config.yml", "config file")
	pidFile := flag.String("pid-file", "", "PID file (systemd の PIDFile. 再起動でプロセスが入れ替わるので必要)")
	drainTimeout := flag.Duration("drain-timeout", time.Hour, "終了・再起動時に, 処理中のリクエストを待つ時間の上限")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("pid", os.Getpid())
	if err := run(*configPath, *pidFile, *drainTimeout, log); err != nil {
		log.Error("exit", "error", err)
		os.Exit(1)
	}
}

func run(configPath, pidFile string, drainTimeout time.Duration, log *slog.Logger) error {
	upg, err := tableflip.New(tableflip.Options{PIDFile: pidFile})
	if err != nil {
		return err
	}
	defer upg.Stop()

	cfg, err := emlauncher.LoadConfig(configPath)
	if err != nil {
		return err
	}
	srv, closeFn, err := newServer(cfg, log)
	if err != nil {
		return err
	}
	defer closeFn()

	ln, err := upg.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	hs := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 30 * time.Second,
		// 大きなパッケージのアップロードがあるので, 本文の読み込みには制限を設けない.
		IdleTimeout: 120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- hs.Serve(ln) }()

	// 待ち受けを始めたことを親プロセス (再起動の場合) に知らせる. 親はここで受け付けをやめる.
	if err := upg.Ready(); err != nil {
		return err
	}
	log.Info("start", "listen", cfg.Listen, "timezone", cfg.Location().String(),
		"storage", cfg.StorageClass, "cache_prefix", cfg.CachePrefix, "upgraded", upg.HasParent())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sig)
loop:
	for {
		select {
		case err := <-errc:
			return err
		case <-upg.Exit():
			log.Info("upgraded: draining")
			break loop
		case s := <-sig:
			if s != syscall.SIGHUP {
				log.Info("shutting down", "signal", s.String())
				break loop
			}
			log.Info("upgrading")
			// 新しいプロセスが Ready になるまで待つ. 失敗したらこのプロセスが動き続ける.
			if err := upg.Upgrade(); err != nil {
				log.Error("upgrade failed", "error", err)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := hs.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}

func newServer(cfg *emlauncher.Config, log *slog.Logger) (*emlauncher.Server, func(), error) {
	d, err := emlauncher.OpenDB(cfg)
	if err != nil {
		return nil, nil, err
	}
	st, err := storage.New(context.Background(), cfg.StorageClass, cfg.AWS, cfg.LocalFile)
	if err != nil {
		d.Close()
		return nil, nil, err
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
		d.Close()
		return nil, nil, err
	}
	return srv, func() { d.Close() }, nil
}
