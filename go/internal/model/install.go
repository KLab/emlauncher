package model

import (
	"context"
	"time"
)

// AppInstallUser は app_install_user テーブルの行 (あるユーザがあるアプリをインストールした記録と通知設定).
type AppInstallUser struct {
	AppID         int64     `db:"app_id"`
	Mail          string    `db:"mail"`
	Notify        bool      `db:"notify"` // true なら更新を通知する (スキーマのコメントとは逆だが PHP 版のコードどおり)
	LastInstalled time.Time `db:"last_installed"`
}

// LogInstall は install_log に記録し, app_install_user を更新する.
func LogInstall(ctx context.Context, q Q, appID, pkgID int64, mail, userAgent string, now time.Time) error {
	if _, err := q.ExecContext(ctx,
		"INSERT INTO install_log (app_id, package_id, mail, user_agent, installed) VALUES (?, ?, ?, ?, ?)",
		appID, pkgID, mail, userAgent, now); err != nil {
		return err
	}
	_, err := q.ExecContext(ctx,
		"INSERT INTO app_install_user (app_id, mail, last_installed) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE last_installed = ?",
		appID, mail, now, now)
	return err
}

func InstallUsersByApp(ctx context.Context, q Q, appID int64) ([]AppInstallUser, error) {
	return list[AppInstallUser](ctx, q, "SELECT * FROM app_install_user WHERE app_id = ? ORDER BY mail", appID)
}

// InstalledApps はユーザがインストールしたアプリ (app_id の降順).
func InstalledApps(ctx context.Context, q Q, mail string) ([]AppInstallUser, error) {
	return list[AppInstallUser](ctx, q, "SELECT * FROM app_install_user WHERE mail = ? ORDER BY app_id DESC", mail)
}

func GetInstalledApp(ctx context.Context, q Q, mail string, appID int64) (*AppInstallUser, error) {
	return get[AppInstallUser](ctx, q, "SELECT * FROM app_install_user WHERE mail = ? AND app_id = ?", mail, appID)
}

func (u *AppInstallUser) UpdateNotify(ctx context.Context, q Q, notify bool) error {
	u.Notify = notify
	_, err := q.ExecContext(ctx, "UPDATE app_install_user SET notify = ? WHERE app_id = ? AND mail = ?", notify, u.AppID, u.Mail)
	return err
}

func (u *AppInstallUser) Delete(ctx context.Context, q Q) error {
	_, err := q.ExecContext(ctx, "DELETE FROM app_install_user WHERE app_id = ? AND mail = ?", u.AppID, u.Mail)
	return err
}

// PackageInstalledDates はユーザがアプリの各パッケージを最後にインストールした日時.
func PackageInstalledDates(ctx context.Context, q Q, mail string, appID int64) (map[int64]time.Time, error) {
	rows, err := list[struct {
		PackageID int64     `db:"package_id"`
		Installed time.Time `db:"installed"`
	}](ctx, q, "SELECT package_id, max(installed) AS installed FROM install_log WHERE app_id = ? AND mail = ? GROUP BY package_id", appID, mail)
	if err != nil {
		return nil, err
	}
	m := map[int64]time.Time{}
	for _, r := range rows {
		m[r.PackageID] = r.Installed
	}
	return m, nil
}

// InstalledPackageIDs はユーザがインストールしたことのあるアプリのパッケージ.
func InstalledPackageIDs(ctx context.Context, q Q, mail string, appID int64) ([]int64, error) {
	return list[int64](ctx, q, "SELECT DISTINCT package_id FROM install_log WHERE mail = ? AND app_id = ?", mail, appID)
}
