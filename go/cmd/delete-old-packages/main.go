// delete-old-packages は保護されていない古いパッケージを削除する (PHP 版の batch/delete_old_packages.php).
// cron に登録して使う.
//
//	delete-old-packages -config /path/to/config.yml [-k 100] [-n]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"
	_ "time/tzdata"

	"emlauncher/internal/clock"
	"emlauncher/internal/config"
	"emlauncher/internal/core"
	"emlauncher/internal/db"
	"emlauncher/internal/model"
	"emlauncher/internal/storage"
)

// 1 回の実行でアプリごとに削除する上限 (PHP 版と同じ).
const limitPerApp = 100

func main() {
	configPath := flag.String("config", "config.yml", "config file")
	keep := flag.Int("k", 100, "keeping package count")
	dryrun := flag.Bool("n", false, "dryrun. don't delete files and records.")
	flag.Parse()

	if err := run(*configPath, *keep, *dryrun); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(configPath string, keep int, dryrun bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer d.Close()
	st, err := storage.New(ctx, cfg)
	if err != nil {
		return err
	}
	clk := clock.New(cfg.Location())
	c := &core.Core{Config: cfg, DB: d, Storage: st, Clock: clk, Log: slog.New(slog.NewTextHandler(os.Stderr, nil))}

	apps, err := model.AllApplications(ctx, d)
	if err != nil {
		return err
	}
	suffix := ""
	if dryrun {
		suffix = " (DRYRUN)"
	}
	now := func() string { return time.Now().In(cfg.Location()).Format(clock.Layout) }
	for _, app := range apps {
		pkgs, err := model.DeletablePackages(ctx, d, app.ID, keep, limitPerApp)
		if err != nil {
			return err
		}
		if len(pkgs) > 0 {
			fmt.Printf("%s Application %d: %d packages%s\n", now(), app.ID, len(pkgs), suffix)
		}
		for _, p := range pkgs {
			fmt.Printf("%s Delete package %d", now(), p.ID)
			if !dryrun {
				if _, _, err := c.DeletePackage(ctx, app.ID, p.ID); err != nil {
					fmt.Println()
					return fmt.Errorf("package %d: %w", p.ID, err)
				}
			}
			fmt.Println(" OK")
		}
	}
	return nil
}
