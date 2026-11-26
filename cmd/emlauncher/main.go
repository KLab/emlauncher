// emlauncher は EMLauncher の HTTP サーバ.
//
//	emlauncher -config /etc/emlauncher/config.yml -pid-file /run/emlauncher/emlauncher.pid
//
// SIGHUP で無停止の再起動を行う. 新しいプロセス (設定を読み直す) に待ち受けソケットを引き継ぎ,
// 古いプロセスは処理中のリクエスト (長いアップロードを含む) を終えてから終了する. 新しいプロセスが
// 起動できなければ古いプロセスが動き続ける. SIGTERM / SIGINT では処理中のリクエストを終えてから終了する.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // TZ のデータが無い環境でも timezone を読めるように

	"github.com/cloudflare/tableflip"

	"github.com/KLab/emlauncher-go/internal/bootstrap"
	"github.com/KLab/emlauncher-go/internal/config"
	"github.com/KLab/emlauncher-go/internal/web"
)

func main() {
	configPath := flag.String("config", "/etc/emlauncher/config.yml", "設定ファイル")
	pidFile := flag.String("pid-file", "", "PID ファイル (無停止の再起動で新しいプロセスの PID に書き換わる)")
	drainTimeout := flag.Duration("drain-timeout", time.Hour, "終了・再起動のときに処理中のリクエストを待つ時間")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(*configPath, *pidFile, *drainTimeout, log); err != nil {
		log.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run(configPath, pidFile string, drainTimeout time.Duration, log *slog.Logger) error {
	upg, err := tableflip.New(tableflip.Options{PIDFile: pidFile})
	if err != nil {
		return err
	}
	defer upg.Stop()
	go handleSignals(upg, log)

	// ここで失敗すると (設定の誤りなど), 再起動を指示した古いプロセスが動き続ける
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	srv, cleanup, err := newServer(cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	ln, err := upg.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Listen, err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	if err := upg.Ready(); err != nil {
		return err
	}
	log.Info("ready", "listen", cfg.Listen, "pid", os.Getpid(), "timezone", cfg.Location.String(),
		"storage", cfg.StorageClass, "cache_prefix", cfg.CachePrefix)

	select {
	case <-upg.Exit():
	case err := <-serveErr:
		return err
	}

	// 待ち受けを閉じ (新しいプロセスが同じソケットで受け続ける), 処理中のリクエストを待つ
	log.Info("shutting down", "pid", os.Getpid(), "drain_timeout", drainTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("exited", "pid", os.Getpid())
	return nil
}

// handleSignals は SIGHUP で再起動し, SIGTERM / SIGINT で終了の処理を始める.
func handleSignals(upg *tableflip.Upgrader, log *slog.Logger) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	for s := range sig {
		if s != syscall.SIGHUP {
			log.Info("signal received, draining", "signal", s.String())
			upg.Stop()
			return
		}
		log.Info("upgrade requested", "pid", os.Getpid())
		if err := upg.Upgrade(); err != nil {
			log.Error("upgrade failed", "err", err)
			continue
		}
		log.Info("upgrade succeeded, draining", "pid", os.Getpid())
	}
}

func newServer(cfg *config.Config, log *slog.Logger) (*http.Server, func(), error) {
	app, err := bootstrap.New(context.Background(), cfg, log)
	if err != nil {
		return nil, nil, err
	}
	templates, err := fs.Sub(web.Templates, "templates")
	if err != nil {
		return nil, nil, err
	}
	static, err := fs.Sub(web.Static, "static")
	if err != nil {
		return nil, nil, err
	}
	h, err := web.New(web.Options{
		Service:    app.Service,
		Config:     cfg,
		Templates:  templates,
		Static:     static,
		LocalFiles: app.LocalFiles,
		Bundletool: app.Bundletool,
		Logger:     log,
	})
	if err != nil {
		app.Close()
		return nil, nil, err
	}
	// DB に繋がらなければ起動しない. 設定の誤りで無停止の再起動をしても, 古いプロセスが動き続ける
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.DB.PingContext(ctx); err != nil {
		app.Close()
		return nil, nil, fmt.Errorf("database is not reachable: %w", err)
	}
	srv := &http.Server{
		Handler:           h.Handler(),
		ReadHeaderTimeout: time.Minute,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	return srv, func() { app.Close() }, nil
}
