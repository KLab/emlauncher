package emlauncher

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	FileDir = "package/"
	TempDir = "temp-data/"
)

// Package は package テーブルの行.
type Package struct {
	ID               int64          `db:"id"`
	AppID            int64          `db:"app_id"`
	Platform         string         `db:"platform"`
	FileName         string         `db:"file_name"`
	Title            string         `db:"title"`
	Description      sql.NullString `db:"description"`
	Identifier       sql.NullString `db:"identifier"`
	OriginalFileName sql.NullString `db:"original_file_name"`
	FileSize         sql.NullInt64  `db:"file_size"`
	Protect          bool           `db:"protect"`
	Created          time.Time      `db:"created"`
}

// FileKey はストレージのキー: package/{app_id}/{id}_{file_name}
func (p *Package) FileKey() string {
	return fmt.Sprintf("%s%d/%d_%s", FileDir, p.AppID, p.ID, p.FileName)
}

// IsAndroidAppBundle は AAB のパッケージか.
func (p *Package) IsAndroidAppBundle() bool {
	return p.Platform == "Android" && strings.HasSuffix(p.OriginalFileName.String, ".aab")
}

func GetPackage(ctx context.Context, q Q, id int64) (*Package, error) {
	return dbGet[Package](ctx, q, "SELECT * FROM package WHERE id = ?", id)
}

func GetPackageForUpdate(ctx context.Context, q Q, id int64) (*Package, error) {
	return dbGet[Package](ctx, q, "SELECT * FROM package WHERE id = ? FOR UPDATE", id)
}

func GetPackages(ctx context.Context, q Q, ids []int64) ([]Package, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := expandIn("SELECT * FROM package WHERE id IN (?) ORDER BY id DESC", ids)
	if err != nil {
		return nil, err
	}
	return dbList[Package](ctx, q, query, args...)
}

func InsertPackage(ctx context.Context, q Q, p *Package) error {
	res, err := q.ExecContext(ctx,
		"INSERT INTO package (app_id, platform, file_name, title, description, identifier, original_file_name, file_size, protect, created)"+
			" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		p.AppID, p.Platform, p.FileName, p.Title, p.Description, p.Identifier, p.OriginalFileName, p.FileSize, p.Protect, p.Created)
	if err != nil {
		return err
	}
	p.ID, err = res.LastInsertId()
	return err
}

func (p *Package) UpdateInfo(ctx context.Context, q Q) error {
	_, err := q.ExecContext(ctx, "UPDATE package SET title = ?, description = ?, protect = ? WHERE id = ?",
		p.Title, p.Description, p.Protect, p.ID)
	return err
}

// Delete は package と, それに紐付く package_tag, attached_file の行を削除する. ファイルは消さない.
func (p *Package) Delete(ctx context.Context, q Q) error {
	for _, query := range []string{
		"DELETE FROM package_tag WHERE package_id = ?",
		"DELETE FROM attached_file WHERE package_id = ?",
		"DELETE FROM package WHERE id = ?",
	} {
		if _, err := q.ExecContext(ctx, query, p.ID); err != nil {
			return err
		}
	}
	return nil
}

func NewestPackage(ctx context.Context, q Q, appID int64) (*Package, error) {
	return dbGet[Package](ctx, q, "SELECT * FROM package WHERE app_id = ? ORDER BY id DESC LIMIT 1", appID)
}

// PackagesByApp はアプリのパッケージを id の降順で返す.
// platform が空でなければ絞り込み, tagIDs があればそのタグをすべて持つものに絞り込む.
func PackagesByApp(ctx context.Context, q Q, appID int64, platform string, tagIDs []int64, offset, limit int) ([]Package, error) {
	if platform == "" && len(tagIDs) == 0 {
		return dbList[Package](ctx, q, "SELECT * FROM package WHERE app_id = ? ORDER BY id DESC LIMIT ?, ?", appID, offset, limit)
	}
	query := "SELECT p.* FROM package p"
	args := []any{}
	if len(tagIDs) > 0 {
		query += " JOIN package_tag t ON p.id = t.package_id AND t.tag_id IN (" + placeholders(len(tagIDs)) + ")"
		for _, id := range tagIDs {
			args = append(args, id)
		}
	}
	query += " WHERE p.app_id = ?"
	args = append(args, appID)
	if platform != "" {
		query += " AND p.platform = ?"
		args = append(args, platform)
	}
	query += " GROUP BY p.id"
	if len(tagIDs) > 0 {
		query += " HAVING COUNT(p.id) = ?"
		args = append(args, len(tagIDs))
	}
	query += " ORDER BY p.id DESC LIMIT ?, ?"
	args = append(args, offset, limit)
	return dbList[Package](ctx, q, query, args...)
}

// DeletablePackages は保護されていないパッケージのうち, 新しい keep 個を除いたものを古い順に limit 個返す.
func DeletablePackages(ctx context.Context, q Q, appID int64, keep, limit int) ([]Package, error) {
	return dbList[Package](ctx, q,
		"SELECT * FROM package WHERE app_id = ? AND protect = 0 AND id NOT IN"+
			" (SELECT * FROM (SELECT id FROM package WHERE app_id = ? AND protect = 0 ORDER BY id DESC LIMIT ?) t)"+
			" ORDER BY id LIMIT ?", appID, appID, keep, limit)
}

// InstallUserCount はパッケージをインストールしたユーザ数.
func InstallUserCount(ctx context.Context, q Q, pkgID int64) (int, error) {
	var n int
	err := sqlxGet(ctx, q, &n, "SELECT count(DISTINCT mail) FROM install_log WHERE package_id = ?", pkgID)
	return n, err
}

func InstallUsers(ctx context.Context, q Q, pkgID int64) ([]string, error) {
	return dbList[string](ctx, q, "SELECT DISTINCT mail FROM install_log WHERE package_id = ? ORDER BY mail", pkgID)
}

func sqlxGet(ctx context.Context, q Q, dest any, query string, args ...any) error {
	return sqlx.GetContext(ctx, q, dest, query, args...)
}
