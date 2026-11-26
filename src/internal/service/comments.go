package service

import (
	"context"
	"database/sql"

	"emlauncher/internal/db"
)

// PostComment はアプリにコメントする. pkgID は対象のパッケージ (0 なら無し).
// コメント番号はアプリごとの最大値 + 1. 投稿後, アプリのオーナー全員に通知する.
func (s *Service) PostComment(ctx context.Context, appID int64, mail string, pkgID int64, message, baseURL string) (db.Comment, error) {
	if message == "" {
		return db.Comment{}, invalidf("message is required")
	}
	var app db.Application
	var c db.Comment
	err := s.inTx(ctx, func(q *db.Queries) error {
		var err error
		if app, err = q.GetApplicationForUpdate(ctx, appID); err != nil {
			return notFound(err, "application")
		}
		if pkgID != 0 {
			// 別のアプリのパッケージは対象にしない
			if p, err := q.GetPackage(ctx, pkgID); err != nil || p.AppID != appID {
				pkgID = 0
			}
		}
		last, err := q.MaxCommentNumber(ctx, appID)
		if err != nil {
			return err
		}
		now := s.Now()
		c = db.Comment{
			AppID:     appID,
			PackageID: sql.NullInt64{Int64: pkgID, Valid: pkgID != 0},
			Number:    last + 1,
			Mail:      mail,
			Message:   message,
			Created:   now,
		}
		if c.ID, err = q.InsertComment(ctx, db.InsertCommentParams{
			AppID: c.AppID, PackageID: c.PackageID, Number: c.Number, Mail: c.Mail, Message: c.Message, Created: c.Created,
		}); err != nil {
			return err
		}
		app.LastCommented = sql.NullTime{Time: now, Valid: true}
		return q.UpdateApplicationLastCommented(ctx, db.UpdateApplicationLastCommentedParams{
			LastCommented: app.LastCommented, DateToSort: dateToSort(app), ID: appID,
		})
	})
	if err != nil {
		return db.Comment{}, err
	}
	if err := s.notifyComment(ctx, app, c, baseURL); err != nil {
		s.log.Error("comment notification", "app_id", appID, "err", err)
	}
	return c, nil
}

// ListComments はアプリのコメントを新しい順に返す.
func (s *Service) ListComments(ctx context.Context, appID int64, limit, offset int) ([]db.Comment, error) {
	return s.q.ListComments(ctx, db.ListCommentsParams{AppID: appID, Limit: int32(limit), Offset: int32(offset)})
}

// CountComments はアプリのコメントの数.
func (s *Service) CountComments(ctx context.Context, appID int64) (int64, error) {
	return s.q.CountComments(ctx, appID)
}

// PackagesByID は id のパッケージを id をキーにして返す (無いものは含まない).
func (s *Service) PackagesByID(ctx context.Context, ids []int64) (map[int64]db.Package, error) {
	m := map[int64]db.Package{}
	if len(ids) == 0 {
		return m, nil
	}
	pkgs, err := s.q.GetPackagesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		m[p.ID] = p
	}
	return m, nil
}
