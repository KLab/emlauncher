package service

import (
	"context"
	"time"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/randstr"
)

// GuestPassTTL はゲストパスの有効期間.
const GuestPassTTL = 7 * 24 * time.Hour

// CreateGuestPass はログインしていない人がインストールできるゲストパスを発行する.
func (s *Service) CreateGuestPass(ctx context.Context, p db.Package, mail string) (db.GuestPass, error) {
	now := s.Now()
	gp := db.GuestPass{
		AppID:     p.AppID,
		PackageID: p.ID,
		Mail:      mail,
		Token:     randstr.New(32),
		Created:   now,
		Expired:   now.Add(GuestPassTTL),
	}
	id, err := s.q.InsertGuestPass(ctx, db.InsertGuestPassParams{
		AppID: gp.AppID, PackageID: gp.PackageID, Mail: gp.Mail, Token: gp.Token, Created: gp.Created, Expired: gp.Expired,
	})
	gp.ID = id
	return gp, err
}

// GetGuestPass はゲストパスを返す. 無ければ ErrNotFound.
func (s *Service) GetGuestPass(ctx context.Context, id int64) (db.GuestPass, error) {
	gp, err := s.q.GetGuestPass(ctx, id)
	return gp, notFound(err, "guest pass")
}

// GuestPassByToken はトークンのゲストパスを返す (期限切れも返す). 無ければ ErrNotFound.
func (s *Service) GuestPassByToken(ctx context.Context, token string) (db.GuestPass, error) {
	gp, err := s.q.GetGuestPassByToken(ctx, token)
	return gp, notFound(err, "guest pass")
}

// Expired はゲストパスが失効しているか.
func (s *Service) Expired(gp db.GuestPass) bool {
	return gp.Expired.Before(s.now())
}

// ExpireGuestPass はゲストパスを失効させる.
func (s *Service) ExpireGuestPass(ctx context.Context, id int64) error {
	return s.q.ExpireGuestPass(ctx, db.ExpireGuestPassParams{Expired: s.Now(), ID: id})
}

// RecordGuestInstall はゲストパスでのインストールを記録する.
func (s *Service) RecordGuestInstall(ctx context.Context, gp db.GuestPass, userAgent, ipAddress string) error {
	return s.q.InsertGuestPassLog(ctx, db.InsertGuestPassLogParams{
		GuestPassID: gp.ID, UserAgent: userAgent, IpAddress: ipAddress, Installed: s.Now(),
	})
}

// ActiveGuestPasses はパッケージの有効なゲストパス.
func (s *Service) ActiveGuestPasses(ctx context.Context, pkgID int64) ([]db.ListActiveGuestPassesByPackageRow, error) {
	return s.q.ListActiveGuestPassesByPackage(ctx, db.ListActiveGuestPassesByPackageParams{PackageID: pkgID, Now: s.Now()})
}

// OwnedGuestPasses はユーザがオーナーのアプリの有効なゲストパス (期限の近い順).
func (s *Service) OwnedGuestPasses(ctx context.Context, mail string) ([]db.ListActiveGuestPassesByOwnerRow, error) {
	return s.q.ListActiveGuestPassesByOwner(ctx, db.ListActiveGuestPassesByOwnerParams{OwnerMail: mail, Now: s.Now()})
}
