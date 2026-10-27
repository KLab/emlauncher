package emlauncher

import (
	"context"
	"database/sql"
	"time"
)

// Application は application テーブルの行.
type Application struct {
	ID            int64          `db:"id"`
	Title         string         `db:"title"`
	APIKey        string         `db:"api_key"`
	IconKey       sql.NullString `db:"icon_key"`
	Description   sql.NullString `db:"description"`
	Repository    sql.NullString `db:"repository"`
	LastUpload    sql.NullTime   `db:"last_upload"`
	LastCommented sql.NullTime   `db:"last_commented"`
	Created       time.Time      `db:"created"`
	DateToSort    time.Time      `db:"date_to_sort"`
}

const IconDir = "app-icons/"

func GetApplication(ctx context.Context, q Q, id int64) (*Application, error) {
	return dbGet[Application](ctx, q, "SELECT * FROM application WHERE id = ?", id)
}

func GetApplicationForUpdate(ctx context.Context, q Q, id int64) (*Application, error) {
	return dbGet[Application](ctx, q, "SELECT * FROM application WHERE id = ? FOR UPDATE", id)
}

func GetApplicationByAPIKey(ctx context.Context, q Q, key string) (*Application, error) {
	return dbGet[Application](ctx, q, "SELECT * FROM application WHERE api_key = ?", key)
}

func GetApplications(ctx context.Context, q Q, ids []int64) ([]Application, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := expandIn("SELECT * FROM application WHERE id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	return dbList[Application](ctx, q, query, args...)
}

func CountApplications(ctx context.Context, q Q) (int, error) {
	var n int
	err := sqlxGet(ctx, q, &n, "SELECT count(*) FROM application")
	return n, err
}

// ApplicationsByDateToSort は新着順 (date_to_sort の降順).
func ApplicationsByDateToSort(ctx context.Context, q Q, offset, limit int) ([]Application, error) {
	return dbList[Application](ctx, q, "SELECT * FROM application ORDER BY date_to_sort DESC LIMIT ?, ?", offset, limit)
}

func AllApplications(ctx context.Context, q Q) ([]Application, error) {
	return dbList[Application](ctx, q, "SELECT * FROM application ORDER BY id")
}

// OwnApplications は mail がオーナーのアプリ (id の降順).
func OwnApplications(ctx context.Context, q Q, mail string) ([]Application, error) {
	return dbList[Application](ctx, q,
		"SELECT a.* FROM application a JOIN application_owner o ON o.app_id = a.id WHERE o.owner_mail = ? ORDER BY a.id DESC", mail)
}

func InsertApplication(ctx context.Context, q Q, a *Application) error {
	res, err := q.ExecContext(ctx,
		"INSERT INTO application (title, api_key, icon_key, description, repository, created, date_to_sort) VALUES (?, ?, ?, ?, ?, ?, ?)",
		a.Title, a.APIKey, a.IconKey, a.Description, a.Repository, a.Created, a.DateToSort)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func (a *Application) UpdateIconKey(ctx context.Context, q Q, key string) error {
	a.IconKey = sql.NullString{String: key, Valid: true}
	_, err := q.ExecContext(ctx, "UPDATE application SET icon_key = ? WHERE id = ?", a.IconKey, a.ID)
	return err
}

func (a *Application) UpdateInfo(ctx context.Context, q Q) error {
	_, err := q.ExecContext(ctx,
		"UPDATE application SET title = ?, description = ?, repository = ?, icon_key = ? WHERE id = ?",
		a.Title, a.Description, a.Repository, a.IconKey, a.ID)
	return err
}

func (a *Application) UpdateAPIKey(ctx context.Context, q Q, key string) error {
	a.APIKey = key
	_, err := q.ExecContext(ctx, "UPDATE application SET api_key = ? WHERE id = ?", key, a.ID)
	return err
}

// calcDateToSort は created, last_upload, last_commented のうち最新のもの.
func (a *Application) calcDateToSort() {
	d := a.Created
	if a.LastUpload.Valid && a.LastUpload.Time.After(d) {
		d = a.LastUpload.Time
	}
	if a.LastCommented.Valid && a.LastCommented.Time.After(d) {
		d = a.LastCommented.Time
	}
	a.DateToSort = d
}

// UpdateLastUpload は last_upload と date_to_sort を更新する. t が無効なら NULL にする.
func (a *Application) UpdateLastUpload(ctx context.Context, q Q, t sql.NullTime) error {
	a.LastUpload = t
	a.calcDateToSort()
	_, err := q.ExecContext(ctx, "UPDATE application SET last_upload = ?, date_to_sort = ? WHERE id = ?",
		a.LastUpload, a.DateToSort, a.ID)
	return err
}

func (a *Application) UpdateLastCommented(ctx context.Context, q Q, t time.Time) error {
	a.LastCommented = sql.NullTime{Time: t, Valid: true}
	a.calcDateToSort()
	_, err := q.ExecContext(ctx, "UPDATE application SET last_commented = ?, date_to_sort = ? WHERE id = ?",
		a.LastCommented, a.DateToSort, a.ID)
	return err
}

// Owners はオーナーのメールアドレス.
func Owners(ctx context.Context, q Q, appID int64) ([]string, error) {
	return dbList[string](ctx, q, "SELECT owner_mail FROM application_owner WHERE app_id = ? ORDER BY id", appID)
}

func IsOwner(ctx context.Context, q Q, appID int64, mail string) (bool, error) {
	var n int
	err := sqlxGet(ctx, q, &n, "SELECT count(*) FROM application_owner WHERE app_id = ? AND owner_mail = ?", appID, mail)
	return n > 0, err
}

func AddOwners(ctx context.Context, q Q, appID int64, mails []string) error {
	for _, m := range mails {
		if _, err := q.ExecContext(ctx, "INSERT INTO application_owner (app_id, owner_mail) VALUES (?, ?)", appID, m); err != nil {
			return err
		}
	}
	return nil
}

func DeleteOwners(ctx context.Context, q Q, appID int64, mails []string) error {
	if len(mails) == 0 {
		return nil
	}
	query, args, err := expandIn("DELETE FROM application_owner WHERE app_id = ? AND owner_mail IN (?)", appID, mails)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, query, args...)
	return err
}
