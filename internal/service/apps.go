package service

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/icon"
	"github.com/KLab/emlauncher-go/internal/randstr"
)

// GetApp はアプリを返す. 無ければ ErrNotFound.
func (s *Service) GetApp(ctx context.Context, id int64) (db.Application, error) {
	app, err := s.q.GetApplication(ctx, id)
	return app, notFound(err, "application")
}

// GetAppByAPIKey は API キーのアプリを返す. 無ければ ErrNotFound.
func (s *Service) GetAppByAPIKey(ctx context.Context, apiKey string) (db.Application, error) {
	if apiKey == "" {
		return db.Application{}, fmt.Errorf("application: %w", ErrNotFound)
	}
	app, err := s.q.GetApplicationByAPIKey(ctx, apiKey)
	return app, notFound(err, "application")
}

// IconURL はアプリのアイコンの URL.
func (s *Service) IconURL(app db.Application) string {
	if !app.IconKey.Valid {
		return ""
	}
	return s.storage.PublicURL(app.IconKey.String)
}

// IsOwner は mail がアプリのオーナーか.
func (s *Service) IsOwner(ctx context.Context, appID int64, mail string) (bool, error) {
	if mail == "" {
		return false, nil
	}
	return s.q.IsOwner(ctx, db.IsOwnerParams{AppID: appID, OwnerMail: mail})
}

// AppInfo はアプリの画面で編集する項目. Icon は画像のデータ (PNG / GIF / JPEG). 更新では nil なら変えない.
type AppInfo struct {
	Title       string
	Icon        []byte
	Description *string // nil なら NULL (PHP 版と同じく, 送られた値は空でもそのまま保存する)
	Repository  *string
}

// CreateApp はアプリを作り, owner をオーナーにする.
func (s *Service) CreateApp(ctx context.Context, owner string, info AppInfo) (int64, error) {
	if info.Title == "" || info.Icon == nil {
		return 0, invalidf("title and icon are required")
	}
	png, err := icon.ToPNG(info.Icon)
	if err != nil {
		return 0, &InvalidError{Msg: err.Error()}
	}
	var appID int64
	var iconKey string
	err = s.inTx(ctx, func(q *db.Queries) error {
		apiKey, err := s.newAPIKey(ctx, q)
		if err != nil {
			return err
		}
		now := s.Now()
		appID, err = q.InsertApplication(ctx, db.InsertApplicationParams{
			Title:       info.Title,
			ApiKey:      apiKey,
			Description: optionalString(info.Description),
			Repository:  optionalString(info.Repository),
			Created:     now,
			DateToSort:  now,
		})
		if err != nil {
			return err
		}
		if err := q.InsertOwner(ctx, db.InsertOwnerParams{AppID: appID, OwnerMail: owner}); err != nil {
			return err
		}
		iconKey = newIconKey(appID)
		if err := s.storage.PutIcon(ctx, iconKey, png); err != nil {
			return err
		}
		return q.UpdateApplicationIcon(ctx, db.UpdateApplicationIconParams{IconKey: nullString(iconKey), ID: appID})
	})
	if err != nil {
		if iconKey != "" {
			s.deleteFiles(ctx, iconKey)
		}
		return 0, err
	}
	return appID, nil
}

// newAPIKey は使われていない API キーを作る.
func (s *Service) newAPIKey(ctx context.Context, q *db.Queries) (string, error) {
	for {
		key := randstr.New(16)
		exists, err := q.APIKeyExists(ctx, key)
		if err != nil {
			return "", err
		}
		if !exists {
			return key, nil
		}
	}
}

// UpdateApp はアプリの情報を更新する. アイコンを変えたら古いアイコンを消す.
func (s *Service) UpdateApp(ctx context.Context, appID int64, info AppInfo) error {
	if info.Title == "" {
		return invalidf("title is required")
	}
	var png []byte
	if info.Icon != nil {
		var err error
		if png, err = icon.ToPNG(info.Icon); err != nil {
			return &InvalidError{Msg: err.Error()}
		}
	}
	var oldIcon, newIcon string
	err := s.inTx(ctx, func(q *db.Queries) error {
		app, err := q.GetApplicationForUpdate(ctx, appID)
		if err != nil {
			return notFound(err, "application")
		}
		iconKey := app.IconKey
		if png != nil {
			newIcon = newIconKey(appID)
			if err := s.storage.PutIcon(ctx, newIcon, png); err != nil {
				return err
			}
			oldIcon, iconKey = app.IconKey.String, nullString(newIcon)
		}
		return q.UpdateApplicationInfo(ctx, db.UpdateApplicationInfoParams{
			Title:       info.Title,
			Description: optionalString(info.Description),
			Repository:  optionalString(info.Repository),
			IconKey:     iconKey,
			ID:          appID,
		})
	})
	if err != nil {
		if newIcon != "" {
			s.deleteFiles(ctx, newIcon)
		}
		return err
	}
	if oldIcon != "" {
		s.deleteFiles(ctx, oldIcon)
	}
	return nil
}

// RefreshAPIKey は API キーを作り直す. 画面に表示していたキー (oldKey) が現在のキーと
// 違えば (別の画面で更新済みなど) 何もしない.
func (s *Service) RefreshAPIKey(ctx context.Context, appID int64, oldKey string) error {
	return s.inTx(ctx, func(q *db.Queries) error {
		app, err := q.GetApplicationForUpdate(ctx, appID)
		if err != nil {
			return notFound(err, "application")
		}
		if app.ApiKey != oldKey {
			return nil
		}
		key, err := s.newAPIKey(ctx, q)
		if err != nil {
			return err
		}
		return q.UpdateApplicationAPIKey(ctx, db.UpdateApplicationAPIKeyParams{ApiKey: key, ID: appID})
	})
}

// Owners はアプリのオーナーのメールアドレス (登録順).
func (s *Service) Owners(ctx context.Context, appID int64) ([]string, error) {
	return s.q.ListOwners(ctx, appID)
}

// SetOwners はオーナーを owners にする. 更新した本人 (by) は必ず残す.
func (s *Service) SetOwners(ctx context.Context, appID int64, by string, owners []string) error {
	want := []string{by}
	for _, o := range owners {
		o = strings.TrimSpace(o)
		if o != "" && !slices.Contains(want, o) {
			want = append(want, o)
		}
	}
	return s.inTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetApplicationForUpdate(ctx, appID); err != nil {
			return notFound(err, "application")
		}
		current, err := q.ListOwners(ctx, appID)
		if err != nil {
			return err
		}
		for _, o := range current {
			if !slices.Contains(want, o) {
				if err := q.DeleteOwner(ctx, db.DeleteOwnerParams{AppID: appID, OwnerMail: o}); err != nil {
					return err
				}
			}
		}
		for _, o := range want {
			if !slices.Contains(current, o) {
				if err := q.InsertOwner(ctx, db.InsertOwnerParams{AppID: appID, OwnerMail: o}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DeleteTags はアプリのタグを消す (パッケージに付いていれば外す).
func (s *Service) DeleteTags(ctx context.Context, appID int64, tagIDs []int64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	return s.inTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetApplicationForUpdate(ctx, appID); err != nil {
			return notFound(err, "application")
		}
		tags, err := q.ListTagsByApp(ctx, appID)
		if err != nil {
			return err
		}
		// 他のアプリのタグは消さない
		var ids []int64
		for _, t := range tags {
			if slices.Contains(tagIDs, t.ID) {
				ids = append(ids, t.ID)
			}
		}
		if len(ids) == 0 {
			return nil
		}
		if err := q.DeletePackageTagsByTagIDs(ctx, ids); err != nil {
			return err
		}
		return q.DeleteTagsByIDs(ctx, db.DeleteTagsByIDsParams{AppID: appID, TagIds: ids})
	})
}

// tagsByName はタグ名に対応するアプリのタグを返す. 無いタグは create なら作り, そうでなければ ok=false.
// 空の名前と重複は無視する. 呼び出し側でアプリの行をロックしておくこと.
func tagsByName(ctx context.Context, q *db.Queries, appID int64, names []string, create bool) (tags []db.Tag, ok bool, err error) {
	existing, err := q.ListTagsByApp(ctx, appID)
	if err != nil {
		return nil, false, err
	}
	var seen []string
	for _, name := range names {
		if name == "" || slices.Contains(seen, name) {
			continue
		}
		seen = append(seen, name)
		i := slices.IndexFunc(existing, func(t db.Tag) bool { return t.Name == name })
		if i >= 0 {
			tags = append(tags, existing[i])
			continue
		}
		if !create {
			return nil, false, nil
		}
		id, err := q.InsertTag(ctx, db.InsertTagParams{AppID: appID, Name: name})
		if err != nil {
			return nil, false, err
		}
		t := db.Tag{ID: id, AppID: appID, Name: name}
		existing = append(existing, t)
		tags = append(tags, t)
	}
	return tags, true, nil
}

// TagIDsByName はタグ名に対応するアプリのタグの id. 存在しない名前があれば ok=false.
func (s *Service) TagIDsByName(ctx context.Context, appID int64, names []string) (ids []int64, ok bool, err error) {
	tags, ok, err := tagsByName(ctx, s.q, appID, names, false)
	for _, t := range tags {
		ids = append(ids, t.ID)
	}
	return ids, ok, err
}

// touchLastUpload は application.last_upload を t にし, date_to_sort を計算し直す.
func touchLastUpload(ctx context.Context, q *db.Queries, app db.Application, t sql.NullTime) error {
	app.LastUpload = t
	return q.UpdateApplicationLastUpload(ctx, db.UpdateApplicationLastUploadParams{
		LastUpload: t,
		DateToSort: dateToSort(app),
		ID:         app.ID,
	})
}

// dateToSort は max(created, last_upload, last_commented).
func dateToSort(app db.Application) time.Time {
	t := app.Created
	if app.LastUpload.Valid && app.LastUpload.Time.After(t) {
		t = app.LastUpload.Time
	}
	if app.LastCommented.Valid && app.LastCommented.Time.After(t) {
		t = app.LastCommented.Time
	}
	return t
}

// deleteFiles はストレージのファイルを消す. 失敗してもログに残すだけにする
// (DB からは消えているので, 処理全体は成功にする).
func (s *Service) deleteFiles(ctx context.Context, keys ...string) {
	ctx = context.WithoutCancel(ctx)
	for _, k := range keys {
		if err := s.storage.Delete(ctx, k); err != nil {
			s.log.Error("delete file", "key", k, "err", err)
		}
	}
}
