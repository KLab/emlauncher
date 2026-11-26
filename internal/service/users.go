package service

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"emlauncher/internal/cache"
	"emlauncher/internal/db"
	"emlauncher/internal/mailer"
	"emlauncher/internal/passhash"
)

// passwordResetTTL はパスワードリセットの URL の有効期間.
const passwordResetTTL = 30 * time.Minute

// Authenticate はメールアドレスとパスワードを確かめる.
func (s *Service) Authenticate(ctx context.Context, mail, password string) (bool, error) {
	u, err := s.q.GetUserPass(ctx, mail)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return passhash.Check(u.Passhash.String, password), nil
}

// CanLogin は外部の認証 (Google / AzureAD) で確かめたアカウントがログインできるか.
// login.allowed_mailaddr_pattern に一致するか, user_pass に登録されていればよい.
func (s *Service) CanLogin(ctx context.Context, mail string) (bool, error) {
	if mail == "" {
		return false, nil
	}
	if re := s.cfg.Login.AllowedMail; re != nil && re.MatchString(mail) {
		return true, nil
	}
	_, err := s.q.GetUserPass(ctx, mail)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

type passwordReset struct {
	Mail string `json:"mail"`
}

// SendPasswordReset はパスワードを設定する URL をメールで送る. 登録されていないアドレスなら ErrNotFound.
// resetURL はキーから URL を作る.
func (s *Service) SendPasswordReset(ctx context.Context, mail string, resetURL func(key string) string) error {
	if _, err := s.q.GetUserPass(ctx, mail); err != nil {
		return notFound(err, "user")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	sum := sha1.Sum(b)
	key := hex.EncodeToString(sum[:])
	v, _ := json.Marshal(passwordReset{Mail: mail})
	if err := s.cache.Set(key, v, passwordResetTTL); err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		From:    s.cfg.MailSender,
		To:      []string{mail},
		Subject: "Reset password",
		Body:    fmt.Sprintf("EMLauncher password reset URL:\n%s\n", resetURL(key)),
	})
}

// PasswordResetMail はパスワードリセットのキーのメールアドレス. 無効なキーなら ErrNotFound.
func (s *Service) PasswordResetMail(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", ErrNotFound
	}
	v, err := s.cache.Get(key)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	var pr passwordReset
	if err := json.Unmarshal(v, &pr); err != nil || pr.Mail == "" {
		return "", ErrNotFound
	}
	return pr.Mail, nil
}

// ResetPassword はパスワードリセットのキーのユーザのパスワードを変える. キーは使い捨て.
func (s *Service) ResetPassword(ctx context.Context, key, password string) error {
	if password == "" {
		return invalidf("password is required")
	}
	mail, err := s.PasswordResetMail(ctx, key)
	if err != nil {
		return err
	}
	if _, err := s.q.GetUserPass(ctx, mail); err != nil {
		return notFound(err, "user")
	}
	if err := s.q.UpdatePasshash(ctx, db.UpdatePasshashParams{Passhash: nullString(passhash.New(password)), Mail: mail}); err != nil {
		return err
	}
	return s.cache.Delete(key)
}
