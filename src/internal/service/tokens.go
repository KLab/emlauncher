package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"emlauncher/internal/cache"
	"emlauncher/internal/db"
	"emlauncher/internal/randstr"
)

// memcached のキー (cache_prefix の後ろ).
const (
	installTokenPrefix = "pkg_install_token_" // PHP 版と共有する
	plistTokenPrefix   = "ios_plist_token_"
	plistTokenTTL      = 60 * time.Second
)

// installToken は memcached に置くインストールトークンの内容. PHP 版と同じ JSON
// ({"mail","package_id","expire"}) で, PHP 版が書いたものも読む.
type installToken struct {
	Mail      string     `json:"mail"`
	PackageID flexibleID `json:"package_id"`
	Expire    string     `json:"expire"`
}

// flexibleID は文字列 ("12") でも数値 (12) でも読める id. 書くときは文字列 (PHP 版の API と同じ).
type flexibleID int64

func (id flexibleID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(id), 10))
}

func (id *flexibleID) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var n int64
	var err error
	switch x := v.(type) {
	case string:
		n, err = strconv.ParseInt(x, 10, 64)
	case float64:
		n = int64(x)
	default:
		err = fmt.Errorf("invalid id %s", b)
	}
	*id = flexibleID(n)
	return err
}

// InstallToken はログインせずにインストールできるトークン. インストールは Mail のユーザとして記録する.
type InstallToken struct {
	Token     string
	Mail      string
	PackageID int64
	Expire    time.Time
}

// CreateInstallToken はインストールトークンを発行する.
func (s *Service) CreateInstallToken(ctx context.Context, p db.Package, mail string, ttl time.Duration) (InstallToken, error) {
	t := InstallToken{Token: randstr.New(32), Mail: mail, PackageID: p.ID, Expire: s.Now().Add(ttl)}
	v, err := json.Marshal(installToken{Mail: mail, PackageID: flexibleID(p.ID), Expire: s.FormatTime(t.Expire)})
	if err != nil {
		return t, err
	}
	return t, s.cache.Set(installTokenPrefix+t.Token, v, ttl)
}

// InstallTokenByToken はインストールトークンを返す. 無いか期限切れなら ErrNotFound.
func (s *Service) InstallTokenByToken(ctx context.Context, token string) (InstallToken, error) {
	if token == "" {
		return InstallToken{}, ErrNotFound
	}
	v, err := s.cache.Get(installTokenPrefix + token)
	if errors.Is(err, cache.ErrNotFound) {
		return InstallToken{}, ErrNotFound
	}
	if err != nil {
		return InstallToken{}, err
	}
	var it installToken
	if err := json.Unmarshal(v, &it); err != nil {
		return InstallToken{}, fmt.Errorf("install token %s: %w", token, ErrNotFound)
	}
	expire, err := s.ParseTime(it.Expire)
	if err != nil || expire.Before(s.now()) {
		return InstallToken{}, ErrNotFound
	}
	return InstallToken{Token: token, Mail: it.Mail, PackageID: int64(it.PackageID), Expire: expire}, nil
}

// CreatePlistToken は iOS のインストール (itms-services) で plist を取得するためのトークンを作る.
// iOS のインストーラはセッションを持たないので, URL にこのトークンを付ける. 60 秒有効.
func (s *Service) CreatePlistToken(ctx context.Context, pkgID int64) (string, error) {
	token := randstr.New(32)
	return token, s.cache.Set(plistTokenPrefix+token, []byte(strconv.FormatInt(pkgID, 10)), plistTokenTTL)
}

// CheckPlistToken は token が pkgID のパッケージのものか確かめる.
func (s *Service) CheckPlistToken(ctx context.Context, token string, pkgID int64) (bool, error) {
	if token == "" {
		return false, nil
	}
	v, err := s.cache.Get(plistTokenPrefix + token)
	if errors.Is(err, cache.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return string(v) == strconv.FormatInt(pkgID, 10), nil
}
