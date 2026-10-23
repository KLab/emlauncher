package emlauncher

import (
	"context"
	"database/sql"
	"time"
)

// Comment は comment テーブルの行.
type Comment struct {
	ID        int64         `db:"id"`
	AppID     int64         `db:"app_id"`
	PackageID sql.NullInt64 `db:"package_id"`
	Number    int64         `db:"number"`
	Mail      string        `db:"mail"`
	Message   string        `db:"message"`
	Created   time.Time     `db:"created"`
}

func CountComments(ctx context.Context, q Q, appID int64) (int, error) {
	var n int
	err := sqlxGet(ctx, q, &n, "SELECT count(*) FROM comment WHERE app_id = ?", appID)
	return n, err
}

// CommentCounts はアプリごとのコメント数.
func CommentCounts(ctx context.Context, q Q, appIDs []int64) (map[int64]int, error) {
	m := map[int64]int{}
	if len(appIDs) == 0 {
		return m, nil
	}
	query, args, err := expandIn("SELECT app_id, count(*) AS n FROM comment WHERE app_id IN (?) GROUP BY app_id", appIDs)
	if err != nil {
		return nil, err
	}
	rows, err := dbList[struct {
		AppID int64 `db:"app_id"`
		N     int   `db:"n"`
	}](ctx, q, query, args...)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		m[r.AppID] = r.N
	}
	return m, nil
}

// CommentsByApp は新しい順.
func CommentsByApp(ctx context.Context, q Q, appID int64, offset, limit int) ([]Comment, error) {
	return dbList[Comment](ctx, q, "SELECT * FROM comment WHERE app_id = ? ORDER BY id DESC LIMIT ?, ?", appID, offset, limit)
}

// InsertComment は番号 (アプリ毎の通し番号) を振ってコメントを追加する.
// アプリの行をロックしたトランザクション内で呼ぶこと.
func InsertComment(ctx context.Context, q Q, c *Comment) error {
	var last sql.NullInt64
	if err := sqlxGet(ctx, q, &last, "SELECT max(number) FROM comment WHERE app_id = ?", c.AppID); err != nil {
		return err
	}
	c.Number = last.Int64 + 1
	res, err := q.ExecContext(ctx,
		"INSERT INTO comment (app_id, package_id, number, mail, message, created) VALUES (?, ?, ?, ?, ?, ?)",
		c.AppID, c.PackageID, c.Number, c.Mail, c.Message, c.Created)
	if err != nil {
		return err
	}
	c.ID, err = res.LastInsertId()
	return err
}

// GuestPass は guest_pass テーブルの行.
type GuestPass struct {
	ID        int64     `db:"id"`
	AppID     int64     `db:"app_id"`
	PackageID int64     `db:"package_id"`
	Mail      string    `db:"mail"`
	Token     string    `db:"token"`
	Created   time.Time `db:"created"`
	Expired   time.Time `db:"expired"`
}

func GetGuestPass(ctx context.Context, q Q, id int64) (*GuestPass, error) {
	return dbGet[GuestPass](ctx, q, "SELECT * FROM guest_pass WHERE id = ?", id)
}

func GetGuestPassByToken(ctx context.Context, q Q, token string) (*GuestPass, error) {
	return dbGet[GuestPass](ctx, q, "SELECT * FROM guest_pass WHERE token = ? LIMIT 1", token)
}

func InsertGuestPass(ctx context.Context, q Q, g *GuestPass) error {
	res, err := q.ExecContext(ctx,
		"INSERT INTO guest_pass (app_id, package_id, mail, token, created, expired) VALUES (?, ?, ?, ?, ?, ?)",
		g.AppID, g.PackageID, g.Mail, g.Token, g.Created, g.Expired)
	if err != nil {
		return err
	}
	g.ID, err = res.LastInsertId()
	return err
}

func (g *GuestPass) Expire(ctx context.Context, q Q, now time.Time) error {
	g.Expired = now
	_, err := q.ExecContext(ctx, "UPDATE guest_pass SET expired = ? WHERE id = ?", now, g.ID)
	return err
}

// ActiveGuestPassesByPackage は有効なゲストパス.
func ActiveGuestPassesByPackage(ctx context.Context, q Q, pkgID int64, now time.Time) ([]GuestPass, error) {
	return dbList[GuestPass](ctx, q, "SELECT * FROM guest_pass WHERE package_id = ? AND expired > ? ORDER BY id", pkgID, now)
}

// ActiveGuestPassesByOwner は mail がオーナーのアプリの有効なゲストパス (失効が近い順).
func ActiveGuestPassesByOwner(ctx context.Context, q Q, mail string, now time.Time) ([]GuestPass, error) {
	return dbList[GuestPass](ctx, q,
		"SELECT g.* FROM guest_pass g JOIN application_owner o ON g.app_id = o.app_id"+
			" WHERE o.owner_mail = ? AND g.expired > ? ORDER BY g.expired ASC", mail, now)
}

func LogGuestPassInstall(ctx context.Context, q Q, guestPassID int64, userAgent, ip string, now time.Time) error {
	_, err := q.ExecContext(ctx,
		"INSERT INTO guestpass_log (guest_pass_id, user_agent, ip_address, installed) VALUES (?, ?, ?, ?)",
		guestPassID, userAgent, ip, now)
	return err
}

func GuestPassInstallCount(ctx context.Context, q Q, guestPassID int64) (int, error) {
	var n int
	err := sqlxGet(ctx, q, &n, "SELECT count(*) FROM guestpass_log WHERE guest_pass_id = ?", guestPassID)
	return n, err
}

// UserPass は user_pass テーブルの行.
type UserPass struct {
	Mail     string         `db:"mail"`
	Passhash sql.NullString `db:"passhash"`
}

func GetUserPass(ctx context.Context, q Q, mail string) (*UserPass, error) {
	return dbGet[UserPass](ctx, q, "SELECT * FROM user_pass WHERE mail = ?", mail)
}

func (u *UserPass) UpdatePasshash(ctx context.Context, q Q, hash string) error {
	u.Passhash = sql.NullString{String: hash, Valid: true}
	_, err := q.ExecContext(ctx, "UPDATE user_pass SET passhash = ? WHERE mail = ?", hash, u.Mail)
	return err
}
