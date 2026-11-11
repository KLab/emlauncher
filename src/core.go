// API と画面で共通の業務処理 (パッケージのアップロード・削除・編集など).

package emlauncher

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"emlauncher/pkgfile"
	"emlauncher/storage"
)

type Core struct {
	Config     *Config
	DB         *sqlx.DB
	Storage    storage.Storage
	Clock      *Clock
	Cache      *Cache
	Mail       Mailer
	Bundletool *pkgfile.Bundletool
	Log        *slog.Logger
}

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

// tx はトランザクション内で fn を実行する. fn がエラーを返したらロールバックする.
func (c *Core) tx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	return runTx(ctx, c.DB, fn)
}

// uploadTx はパッケージのアップロード用の tx. 設定 upload_lock_wait_timeout があれば,
// その接続のロック待ちのタイムアウトを変える. 接続はプールに戻るので, 終わったら元に戻す.
func (c *Core) uploadTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	timeout := c.Config.UploadLockWaitTimeout
	if timeout <= 0 {
		return c.tx(ctx, fn)
	}
	conn, err := c.DB.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET SESSION innodb_lock_wait_timeout = %d", timeout)); err != nil {
		return err
	}
	defer func() {
		if _, err := conn.ExecContext(context.WithoutCancel(ctx), "SET SESSION innodb_lock_wait_timeout = DEFAULT"); err != nil {
			// 戻せなかった接続はプールに戻さない
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	return runTx(ctx, conn, fn)
}

type txBeginner interface {
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

func runTx(ctx context.Context, db txBeginner, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
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
