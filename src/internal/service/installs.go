package service

import (
	"context"
	"time"

	"emlauncher/internal/db"
)

// RecordInstall はインストール履歴を記録し, アプリをインストール済みにする (更新通知の対象になる).
func (s *Service) RecordInstall(ctx context.Context, p db.Package, mail, userAgent string) error {
	return s.inTx(ctx, func(q *db.Queries) error {
		now := s.Now()
		if err := q.InsertInstallLog(ctx, db.InsertInstallLogParams{
			AppID: p.AppID, PackageID: p.ID, Mail: mail, UserAgent: userAgent, Installed: now,
		}); err != nil {
			return err
		}
		return q.UpsertAppInstallUser(ctx, db.UpsertAppInstallUserParams{AppID: p.AppID, Mail: mail, Installed: now})
	})
}

// InstalledDates はユーザがアプリのパッケージを最後にインストールした日時 (パッケージの id がキー).
func (s *Service) InstalledDates(ctx context.Context, appID int64, mail string) (map[int64]time.Time, error) {
	logs, err := s.q.ListUserInstallLogs(ctx, db.ListUserInstallLogsParams{AppID: appID, Mail: mail})
	if err != nil {
		return nil, err
	}
	m := map[int64]time.Time{}
	for _, l := range logs {
		if t, ok := m[l.PackageID]; !ok || l.Installed.After(t) {
			m[l.PackageID] = l.Installed
		}
	}
	return m, nil
}

// PackageInstallUsers はパッケージをインストールしたユーザ.
func (s *Service) PackageInstallUsers(ctx context.Context, pkgID int64) ([]string, error) {
	return s.q.ListPackageInstallUsers(ctx, pkgID)
}

// AppInstallUsers はアプリをインストール済みにしているユーザ.
func (s *Service) AppInstallUsers(ctx context.Context, appID int64) ([]db.AppInstallUser, error) {
	return s.q.ListAppInstallUsers(ctx, appID)
}

// InstalledApps はユーザがインストール済みにしているアプリ (app_id の大きい順).
func (s *Service) InstalledApps(ctx context.Context, mail string) ([]db.AppInstallUser, error) {
	return s.q.ListInstalledApps(ctx, mail)
}

// InstalledPackages はユーザがインストールしたことのあるアプリのパッケージ (新しい順).
func (s *Service) InstalledPackages(ctx context.Context, appID int64, mail string) ([]db.Package, error) {
	return s.q.ListInstalledPackages(ctx, db.ListInstalledPackagesParams{AppID: appID, Mail: mail})
}

// SetNotify はアプリの更新通知を受け取るかを変える. インストール済みでなければ ErrNotFound.
func (s *Service) SetNotify(ctx context.Context, appID int64, mail string, notify bool) (db.AppInstallUser, error) {
	iu, err := s.q.GetAppInstallUser(ctx, db.GetAppInstallUserParams{AppID: appID, Mail: mail})
	if err != nil {
		return iu, notFound(err, "installed application")
	}
	iu.Notify = boolInt(notify)
	err = s.q.UpdateNotify(ctx, db.UpdateNotifyParams{Notify: iu.Notify, AppID: appID, Mail: mail})
	return iu, err
}

// RemoveInstalledApp はアプリをインストール済みの一覧から外す (インストール履歴は残す).
func (s *Service) RemoveInstalledApp(ctx context.Context, appID int64, mail string) error {
	return s.q.DeleteAppInstallUser(ctx, db.DeleteAppInstallUserParams{AppID: appID, Mail: mail})
}
