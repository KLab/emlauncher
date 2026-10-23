package emlauncher

import (
	"context"
	"fmt"
	"time"
)

const (
	AttachedTypeUnknown = ""
	AttachedTypeAPK     = "apk"
	AttachedTypeDSYM    = "dSYM"
)

// AttachedTypeFromExt は拡張子から添付ファイルの種類を決める (apk 以外は不明).
func AttachedTypeFromExt(ext string) string {
	if ext == "apk" {
		return AttachedTypeAPK
	}
	return AttachedTypeUnknown
}

// AttachedFile は attached_file テーブルの行.
type AttachedFile struct {
	ID               int64     `db:"id"`
	AppID            int64     `db:"app_id"`
	PackageID        int64     `db:"package_id"`
	FileName         string    `db:"file_name"`
	OriginalFileName string    `db:"original_file_name"`
	FileSize         int64     `db:"file_size"`
	FileType         string    `db:"file_type"`
	Created          time.Time `db:"created"`
}

// FileKey はストレージのキー: package/{app_id}/{package_id}_{id}_{file_name}
func (a *AttachedFile) FileKey() string {
	return fmt.Sprintf("%s%d/%d_%d_%s", FileDir, a.AppID, a.PackageID, a.ID, a.FileName)
}

func GetAttachedFile(ctx context.Context, q Q, id int64) (*AttachedFile, error) {
	return dbGet[AttachedFile](ctx, q, "SELECT * FROM attached_file WHERE id = ?", id)
}

func AttachedFilesByPackage(ctx context.Context, q Q, pkgID int64) ([]AttachedFile, error) {
	return dbList[AttachedFile](ctx, q, "SELECT * FROM attached_file WHERE package_id = ? ORDER BY id", pkgID)
}

func InsertAttachedFile(ctx context.Context, q Q, a *AttachedFile) error {
	res, err := q.ExecContext(ctx,
		"INSERT INTO attached_file (app_id, package_id, file_name, original_file_name, file_size, file_type, created) VALUES (?, ?, ?, ?, ?, ?, ?)",
		a.AppID, a.PackageID, a.FileName, a.OriginalFileName, a.FileSize, a.FileType, a.Created)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func (a *AttachedFile) Delete(ctx context.Context, q Q) error {
	_, err := q.ExecContext(ctx, "DELETE FROM attached_file WHERE id = ?", a.ID)
	return err
}
