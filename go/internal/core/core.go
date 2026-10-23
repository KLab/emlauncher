// Package core は API と画面で共通の業務処理 (パッケージのアップロード・削除・編集など).
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/KLabTools/emlauncher/go/internal/cache"
	"github.com/KLabTools/emlauncher/go/internal/clock"
	"github.com/KLabTools/emlauncher/go/internal/config"
	"github.com/KLabTools/emlauncher/go/internal/mail"
	"github.com/KLabTools/emlauncher/go/internal/pkgfile"
	"github.com/KLabTools/emlauncher/go/internal/storage"
)

type Core struct {
	Config     *config.Config
	DB         *sqlx.DB
	Storage    storage.Storage
	Clock      *clock.Clock
	Cache      *cache.Cache
	Mail       mail.Sender
	Bundletool *pkgfile.Bundletool
	Log        *slog.Logger
}

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

// tx はトランザクション内で fn を実行する. fn がエラーを返したらロールバックする.
func (c *Core) tx(ctx context.Context, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := c.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// deleteKeys はストレージのファイルを消す. 失敗はログに残すだけ (DB からは消えているので許容する).
func (c *Core) deleteKeys(ctx context.Context, keys ...string) {
	for _, k := range keys {
		if err := c.Storage.Delete(ctx, k); err != nil {
			c.Log.Error("storage delete failed", "key", k, "error", err)
		}
	}
}

func notFound(what string, id any) error {
	return fmt.Errorf("%w: %s %v", ErrNotFound, what, id)
}
