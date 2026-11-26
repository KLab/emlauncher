// Package service は EMLauncher の業務処理 (アプリ, パッケージ, アップロード, インストール,
// コメント, ゲストパス, ユーザ) をまとめたもの. HTTP には依存しない.
//
// DB の行は sqlc が生成した型 (internal/db) のまま扱う. 複数の行を変更する処理はトランザクションで行い,
// アプリのパッケージ・タグ・コメント・オーナーを変更するときはアプリの行をロック (SELECT ... FOR UPDATE)
// して直列化する (PHP 版と同じ).
package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"emlauncher/internal/cache"
	"emlauncher/internal/config"
	"emlauncher/internal/db"
	"emlauncher/internal/mailer"
	"emlauncher/internal/pkgfile"
	"emlauncher/internal/storage"
)

// 処理の結果を表すエラー. 呼び出し側 (HTTP) はこれを見てステータスを決める.
var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// InvalidError は入力の誤り (アップロードされたファイルが壊れているなど).
type InvalidError struct{ Msg string }

func (e *InvalidError) Error() string { return e.Msg }

func invalidf(format string, args ...any) error {
	return &InvalidError{Msg: fmt.Sprintf(format, args...)}
}

// Mailer はメールの送信. 実装は mailer.Sendmail.
type Mailer interface {
	Send(ctx context.Context, m mailer.Message) error
}

type Service struct {
	db         *sql.DB
	q          *db.Queries
	storage    storage.Storage
	cache      *cache.Cache
	mailer     Mailer
	bundletool *pkgfile.Bundletool
	cfg        *config.Config
	log        *slog.Logger
	now        func() time.Time
}

type Deps struct {
	DB         *sql.DB
	Storage    storage.Storage
	Cache      *cache.Cache
	Mailer     Mailer
	Bundletool *pkgfile.Bundletool
	Config     *config.Config
	Logger     *slog.Logger
}

func New(d Deps) *Service {
	return &Service{
		db:         d.DB,
		q:          db.New(d.DB),
		storage:    d.Storage,
		cache:      d.Cache,
		mailer:     d.Mailer,
		bundletool: d.Bundletool,
		cfg:        d.Config,
		log:        d.Logger,
		now:        time.Now,
	}
}

// Now は DB に書く現在時刻 (設定の TZ, 秒単位).
// DATETIME は小数部を丸めて保存するので, あらかじめ切り捨てておく.
func (s *Service) Now() time.Time {
	return s.now().In(s.cfg.Location).Truncate(time.Second)
}

// TimeFormat は DB の DATETIME と同じ表記 (API の JSON や画面に出す).
const TimeFormat = "2006-01-02 15:04:05"

// FormatTime は t を設定の TZ で TimeFormat にする.
func (s *Service) FormatTime(t time.Time) string {
	return t.In(s.cfg.Location).Format(TimeFormat)
}

// ParseTime は設定の TZ の TimeFormat の文字列を読む.
func (s *Service) ParseTime(v string) (time.Time, error) {
	return time.ParseInLocation(TimeFormat, v, s.cfg.Location)
}

// Queries は読み取り用のクエリ. 変更は Service のメソッドで行う.
func (s *Service) Queries() *db.Queries { return s.q }

// inTx は fn をトランザクションの中で実行する. fn がエラーを返したらロールバックする.
func (s *Service) inTx(ctx context.Context, fn func(q *db.Queries) error) error {
	return s.inTxWithLockTimeout(ctx, 0, fn)
}

// inUploadTx はアップロード用のトランザクション. 設定 upload_lock_wait_timeout があれば,
// その間だけロック待ちのタイムアウトを変える (大きなファイルを並列にアップロードする場合).
func (s *Service) inUploadTx(ctx context.Context, fn func(q *db.Queries) error) error {
	return s.inTxWithLockTimeout(ctx, s.cfg.UploadLockWaitTimeout, fn)
}

func (s *Service) inTxWithLockTimeout(ctx context.Context, lockWaitTimeout int, fn func(q *db.Queries) error) (err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if lockWaitTimeout > 0 {
		if _, err := conn.ExecContext(ctx, "SET SESSION innodb_lock_wait_timeout = ?", lockWaitTimeout); err != nil {
			return err
		}
		defer func() {
			// 接続はプールに戻るので元に戻す. 戻せなければ接続を捨てる
			if _, rerr := conn.ExecContext(context.WithoutCancel(ctx), "SET SESSION innodb_lock_wait_timeout = DEFAULT"); rerr != nil {
				conn.Raw(func(any) error { return driver.ErrBadConn })
			}
		}()
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// notFound は sql.ErrNoRows を ErrNotFound にする.
func notFound(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", what, ErrNotFound)
	}
	return err
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// optionalString は nil なら NULL.
func optionalString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func boolInt(b bool) int8 {
	if b {
		return 1
	}
	return 0
}
