package emlauncher

import (
	"context"
)

// Tag は tag テーブルの行.
type Tag struct {
	ID    int64  `db:"id"`
	AppID int64  `db:"app_id"`
	Name  string `db:"name"`
}

func TagsByApp(ctx context.Context, q Q, appID int64) ([]Tag, error) {
	return dbList[Tag](ctx, q, "SELECT * FROM tag WHERE app_id = ? ORDER BY id", appID)
}

// UnusedTags はどのパッケージにも付いていないタグ.
func UnusedTags(ctx context.Context, q Q, appID int64) ([]Tag, error) {
	return dbList[Tag](ctx, q, "SELECT * FROM tag WHERE app_id = ? AND id NOT IN (SELECT tag_id FROM package_tag) ORDER BY id", appID)
}

func TagsByPackage(ctx context.Context, q Q, pkgID int64) ([]Tag, error) {
	return dbList[Tag](ctx, q,
		"SELECT t.* FROM package_tag j JOIN tag t ON j.tag_id = t.id WHERE j.package_id = ? ORDER BY j.tag_id", pkgID)
}

// TagsByNames は名前に一致する既存のタグを返す (無いものは含まない).
func TagsByNames(ctx context.Context, q Q, appID int64, names []string) ([]Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}
	query, args, err := expandIn("SELECT * FROM tag WHERE app_id = ? AND name IN (?) ORDER BY id", appID, names)
	if err != nil {
		return nil, err
	}
	return dbList[Tag](ctx, q, query, args...)
}

// GetOrInsertTags は名前からタグを返す. 無いタグは作る. 空の名前と重複は無視する.
// アプリの行をロックしたトランザクション内で呼ぶこと.
func GetOrInsertTags(ctx context.Context, q Q, appID int64, names []string) ([]Tag, error) {
	existing, err := dbList[Tag](ctx, q, "SELECT * FROM tag WHERE app_id = ? ORDER BY id FOR UPDATE", appID)
	if err != nil {
		return nil, err
	}
	byName := map[string]Tag{}
	for _, t := range existing {
		if _, ok := byName[t.Name]; !ok {
			byName[t.Name] = t
		}
	}
	var tags []Tag
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		t, ok := byName[name]
		if !ok {
			res, err := q.ExecContext(ctx, "INSERT INTO tag (app_id, name) VALUES (?, ?)", appID, name)
			if err != nil {
				return nil, err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return nil, err
			}
			t = Tag{ID: id, AppID: appID, Name: name}
			byName[name] = t
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// SetPackageTags はパッケージのタグを置き換える.
func SetPackageTags(ctx context.Context, q Q, pkgID int64, tags []Tag) error {
	if _, err := q.ExecContext(ctx, "DELETE FROM package_tag WHERE package_id = ?", pkgID); err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := q.ExecContext(ctx, "INSERT INTO package_tag (package_id, tag_id) VALUES (?, ?)", pkgID, t.ID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTags はアプリのタグを削除する (package_tag からも外す).
func DeleteTags(ctx context.Context, q Q, appID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tags, err := dbList[int64](ctx, q, "SELECT id FROM tag WHERE app_id = ? FOR UPDATE", appID)
	if err != nil {
		return err
	}
	own := map[int64]bool{}
	for _, id := range tags {
		own[id] = true
	}
	var del []int64
	for _, id := range ids {
		if own[id] {
			del = append(del, id)
		}
	}
	if len(del) == 0 {
		return nil
	}
	for _, query := range []string{"DELETE FROM package_tag WHERE tag_id IN (?)", "DELETE FROM tag WHERE id IN (?)"} {
		qs, args, err := expandIn(query, del)
		if err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, qs, args...); err != nil {
			return err
		}
	}
	return nil
}
