// delete-old-packages は古いパッケージを消すバッチ. cron などで実行する.
//
//	delete-old-packages -config /etc/emlauncher/config.yml -k 100
//
// アプリごとに, 保護されていないパッケージを新しいものから -k 個だけ残して消す (ファイルも消す).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // TZ のデータが無い環境でも timezone を読めるように

	"github.com/KLab/emlauncher-go/internal/bootstrap"
	"github.com/KLab/emlauncher-go/internal/config"
)

func main() {
	configPath := flag.String("config", "/etc/emlauncher/config.yml", "設定ファイル")
	keep := flag.Int("k", 100, "アプリごとに残すパッケージの数")
	dryRun := flag.Bool("n", false, "消さずに対象を表示する")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(*configPath, *keep, *dryRun, log); err != nil {
		log.Error("delete-old-packages failed", "err", err)
		os.Exit(1)
	}
}

func run(configPath string, keep int, dryRun bool, log *slog.Logger) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := bootstrap.New(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer app.Close()
	return app.Service.DeleteOldPackages(ctx, keep, dryRun, os.Stdout)
}
