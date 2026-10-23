package emlauncher

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/jmoiron/sqlx"
)

// AppInfo はアプリの基本情報.
type AppInfo struct {
	Title       string
	Description string
	Repository  string
}

func validString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

// newAPIKey は使われていない API キーを作る.
func (c *Core) newAPIKey(ctx context.Context, q Q) (string, error) {
	for {
		key := randomString(16)
		a, err := GetApplicationByAPIKey(ctx, q, key)
		if err != nil {
			return "", err
		}
		if a == nil {
			return key, nil
		}
	}
}

// saveIcon は画像を 144x144 の PNG にして app-icons/{app_id}/{rand}.png に保存する.
func (c *Core) saveIcon(ctx context.Context, appID int64, image []byte) (string, error) {
	png, err := convertIcon(image)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	key := fmt.Sprintf("%s%d/%s.png", IconDir, appID, randomString(16))
	if err := c.Storage.SaveIcon(ctx, key, png); err != nil {
		return "", err
	}
	return key, nil
}

// CreateApp はアプリを作り, 作成者をオーナーにする.
func (c *Core) CreateApp(ctx context.Context, owner string, info AppInfo, image []byte) (*Application, error) {
	var app *Application
	var iconKey string
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		key, err := c.newAPIKey(ctx, tx)
		if err != nil {
			return err
		}
		now := c.Clock.Now()
		app = &Application{
			Title:       info.Title,
			APIKey:      key,
			Description: validString(info.Description),
			Repository:  validString(info.Repository),
			Created:     now,
			DateToSort:  now,
		}
		if err := InsertApplication(ctx, tx, app); err != nil {
			return err
		}
		if iconKey, err = c.saveIcon(ctx, app.ID, image); err != nil {
			return err
		}
		if err := app.UpdateIconKey(ctx, tx, iconKey); err != nil {
			return err
		}
		return AddOwners(ctx, tx, app.ID, []string{owner})
	})
	if err != nil {
		if iconKey != "" {
			c.deleteKeys(context.WithoutCancel(ctx), iconKey)
		}
		return nil, err
	}
	return app, nil
}

// UpdateAppInfo はアプリの情報を更新する. image があればアイコンも差し替え, 古いアイコンを消す.
func (c *Core) UpdateAppInfo(ctx context.Context, appID int64, info AppInfo, image []byte) error {
	var oldIcon, newIcon string
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		app, err := GetApplicationForUpdate(ctx, tx, appID)
		if err != nil {
			return err
		}
		if app == nil {
			return notFound("application", appID)
		}
		app.Title = info.Title
		app.Description = validString(info.Description)
		app.Repository = validString(info.Repository)
		if image != nil {
			if newIcon, err = c.saveIcon(ctx, appID, image); err != nil {
				return err
			}
			oldIcon = app.IconKey.String
			app.IconKey = sql.NullString{String: newIcon, Valid: true}
		}
		return app.UpdateInfo(ctx, tx)
	})
	if err != nil {
		if newIcon != "" {
			c.deleteKeys(context.WithoutCancel(ctx), newIcon)
		}
		return err
	}
	if oldIcon != "" {
		c.deleteKeys(ctx, oldIcon)
	}
	return nil
}

// RefreshAPIKey は API キーを作り直す. oldKey が現在のキーと違えば何もしない (二重送信対策).
func (c *Core) RefreshAPIKey(ctx context.Context, appID int64, oldKey string) error {
	return c.tx(ctx, func(tx *sqlx.Tx) error {
		app, err := GetApplicationForUpdate(ctx, tx, appID)
		if err != nil || app == nil || app.APIKey != oldKey {
			return err
		}
		key, err := c.newAPIKey(ctx, tx)
		if err != nil {
			return err
		}
		return app.UpdateAPIKey(ctx, tx, key)
	})
}

// SetOwners はオーナーを owners にする.
func (c *Core) SetOwners(ctx context.Context, appID int64, owners []string) error {
	return c.tx(ctx, func(tx *sqlx.Tx) error {
		if _, err := GetApplicationForUpdate(ctx, tx, appID); err != nil {
			return err
		}
		cur, err := Owners(ctx, tx, appID)
		if err != nil {
			return err
		}
		var del, add []string
		for _, o := range cur {
			if !slices.Contains(owners, o) {
				del = append(del, o)
			}
		}
		for _, o := range owners {
			if !slices.Contains(cur, o) && !slices.Contains(add, o) {
				add = append(add, o)
			}
		}
		if err := DeleteOwners(ctx, tx, appID, del); err != nil {
			return err
		}
		return AddOwners(ctx, tx, appID, add)
	})
}

func (c *Core) DeleteTags(ctx context.Context, appID int64, tagIDs []int64) error {
	return c.tx(ctx, func(tx *sqlx.Tx) error {
		if _, err := GetApplicationForUpdate(ctx, tx, appID); err != nil {
			return err
		}
		return DeleteTags(ctx, tx, appID, tagIDs)
	})
}

// PostComment はコメントを追加し, アプリの last_commented を更新する.
func (c *Core) PostComment(ctx context.Context, appID int64, user string, pkgID int64, message string) (*Application, *Comment, error) {
	var app *Application
	cm := &Comment{AppID: appID, Mail: user, Message: message}
	if pkgID > 0 {
		cm.PackageID = sql.NullInt64{Int64: pkgID, Valid: true}
	}
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		var err error
		app, err = GetApplicationForUpdate(ctx, tx, appID)
		if err != nil {
			return err
		}
		if app == nil {
			return notFound("application", appID)
		}
		cm.Created = c.Clock.Now()
		if err := InsertComment(ctx, tx, cm); err != nil {
			return err
		}
		return app.UpdateLastCommented(ctx, tx, cm.Created)
	})
	return app, cm, err
}
