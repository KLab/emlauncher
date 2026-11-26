package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/pkgfile"
)

// GetPackage はパッケージを返す. 無ければ ErrNotFound.
func (s *Service) GetPackage(ctx context.Context, id int64) (db.Package, error) {
	p, err := s.q.GetPackage(ctx, id)
	return p, notFound(err, "package")
}

// PackageQuery はパッケージの一覧の条件.
type PackageQuery struct {
	AppID    int64
	Platform string  // 空ならすべて
	TagIDs   []int64 // すべてのタグを持つもの (重複の無いこと)
	Limit    int
	Offset   int
}

// ListPackages はパッケージを新しい順に返す.
func (s *Service) ListPackages(ctx context.Context, pq PackageQuery) ([]db.Package, error) {
	platform := nullString(pq.Platform)
	if len(pq.TagIDs) == 0 {
		return s.q.ListPackages(ctx, db.ListPackagesParams{
			AppID: pq.AppID, Platform: platform, Limit: int32(pq.Limit), Offset: int32(pq.Offset),
		})
	}
	return s.q.ListPackagesWithTags(ctx, db.ListPackagesWithTagsParams{
		AppID: pq.AppID, Platform: platform, TagIds: pq.TagIDs, TagCount: int64(len(pq.TagIDs)),
		Limit: int32(pq.Limit), Offset: int32(pq.Offset),
	})
}

// PackageDetail はパッケージとタグ・添付ファイル・インストールしたユーザ数.
type PackageDetail struct {
	db.Package
	Tags         []db.Tag
	Attached     []db.AttachedFile
	InstallCount int64
}

// Details はパッケージのタグ・添付ファイル・インストール数をまとめて読む.
func (s *Service) Details(ctx context.Context, pkgs ...db.Package) ([]PackageDetail, error) {
	if len(pkgs) == 0 {
		return nil, nil
	}
	ids := make([]int64, len(pkgs))
	for i, p := range pkgs {
		ids[i] = p.ID
	}
	tags, err := s.q.ListTagsByPackages(ctx, ids)
	if err != nil {
		return nil, err
	}
	attached, err := s.q.ListAttachedFilesByPackages(ctx, ids)
	if err != nil {
		return nil, err
	}
	counts, err := s.q.CountPackageInstallUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	index := make(map[int64]int, len(pkgs))
	details := make([]PackageDetail, len(pkgs))
	for i, p := range pkgs {
		index[p.ID] = i
		details[i] = PackageDetail{Package: p, Tags: []db.Tag{}, Attached: []db.AttachedFile{}}
	}
	for _, t := range tags {
		d := &details[index[t.PackageID]]
		d.Tags = append(d.Tags, db.Tag{ID: t.ID, AppID: d.AppID, Name: t.Name})
	}
	for _, a := range attached {
		d := &details[index[a.PackageID]]
		d.Attached = append(d.Attached, a)
	}
	for _, c := range counts {
		details[index[c.PackageID]].InstallCount = c.Users
	}
	return details, nil
}

// Detail は 1 つのパッケージの PackageDetail.
func (s *Service) Detail(ctx context.Context, p db.Package) (PackageDetail, error) {
	ds, err := s.Details(ctx, p)
	if err != nil {
		return PackageDetail{}, err
	}
	return ds[0], nil
}

// PackageEdit はパッケージの変更. nil の項目は変えない.
type PackageEdit struct {
	Title       *string
	Description *string
	Protect     *bool
	Tags        *[]string // タグ名. 無いタグは作る
}

// EditPackage はパッケージの情報を変更する.
func (s *Service) EditPackage(ctx context.Context, pkgID int64, e PackageEdit) (db.Package, error) {
	if e.Title != nil && *e.Title == "" {
		return db.Package{}, invalidf("title must not be empty")
	}
	var pkg db.Package
	err := s.inTx(ctx, func(q *db.Queries) error {
		p, err := q.GetPackage(ctx, pkgID)
		if err != nil {
			return notFound(err, "package")
		}
		if _, err := q.GetApplicationForUpdate(ctx, p.AppID); err != nil {
			return err
		}
		if p, err = q.GetPackageForUpdate(ctx, pkgID); err != nil {
			return notFound(err, "package")
		}
		if e.Title != nil {
			p.Title = *e.Title
		}
		if e.Description != nil {
			p.Description = optionalString(e.Description)
		}
		if e.Protect != nil {
			p.Protect = boolInt(*e.Protect)
		}
		if err := q.UpdatePackageInfo(ctx, db.UpdatePackageInfoParams{
			Title: p.Title, Description: p.Description, Protect: p.Protect, ID: p.ID,
		}); err != nil {
			return err
		}
		if e.Tags != nil {
			tags, _, err := tagsByName(ctx, q, p.AppID, *e.Tags, true)
			if err != nil {
				return err
			}
			if err := setPackageTags(ctx, q, p.ID, tags); err != nil {
				return err
			}
		}
		pkg = p
		return nil
	})
	return pkg, err
}

func setPackageTags(ctx context.Context, q *db.Queries, pkgID int64, tags []db.Tag) error {
	if err := q.DeletePackageTags(ctx, pkgID); err != nil {
		return err
	}
	for _, t := range tags {
		if err := q.InsertPackageTag(ctx, db.InsertPackageTagParams{PackageID: pkgID, TagID: t.ID}); err != nil {
			return err
		}
	}
	return nil
}

// DeletePackage はパッケージ (タグの対応, 添付ファイルを含む) を消す.
// 消したのがアプリの最新のアップロードなら, last_upload を残っている最新のパッケージのものに戻す.
// 消す前の内容を返す.
func (s *Service) DeletePackage(ctx context.Context, pkgID int64) (PackageDetail, error) {
	return s.deletePackage(ctx, pkgID, false)
}

// ErrProtected は保護されたパッケージを (古いパッケージの削除で) 消そうとしたことを表す.
var ErrProtected = errors.New("package is protected")

// deletePackage はパッケージを消す. unprotectedOnly なら, ロックを取った後に保護されていれば
// ErrProtected を返して消さない.
func (s *Service) deletePackage(ctx context.Context, pkgID int64, unprotectedOnly bool) (PackageDetail, error) {
	var deleted PackageDetail
	err := s.inTx(ctx, func(q *db.Queries) error {
		p, err := q.GetPackage(ctx, pkgID)
		if err != nil {
			return notFound(err, "package")
		}
		app, err := q.GetApplicationForUpdate(ctx, p.AppID)
		if err != nil {
			return err
		}
		if p, err = q.GetPackageForUpdate(ctx, pkgID); err != nil {
			return notFound(err, "package")
		}
		if unprotectedOnly && p.Protect != 0 {
			return ErrProtected
		}
		if deleted, err = s.detailIn(ctx, q, p); err != nil {
			return err
		}
		if err := q.DeletePackageTags(ctx, p.ID); err != nil {
			return err
		}
		if err := q.DeleteAttachedFilesByPackage(ctx, p.ID); err != nil {
			return err
		}
		if err := q.DeletePackage(ctx, p.ID); err != nil {
			return err
		}
		if !app.LastUpload.Valid || !app.LastUpload.Time.Equal(p.Created) {
			return nil
		}
		var last sql.NullTime
		newest, err := q.GetNewestPackage(ctx, app.ID)
		switch {
		case err == nil:
			last = sql.NullTime{Time: newest.Created, Valid: true}
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		return touchLastUpload(ctx, q, app, last)
	})
	if err != nil {
		return PackageDetail{}, err
	}
	keys := []string{PackageKey(deleted.Package)}
	for _, a := range deleted.Attached {
		keys = append(keys, AttachedKey(a))
	}
	s.deleteFiles(ctx, keys...)
	return deleted, nil
}

// detailIn はトランザクションの中で PackageDetail を読む.
func (s *Service) detailIn(ctx context.Context, q *db.Queries, p db.Package) (PackageDetail, error) {
	d := PackageDetail{Package: p}
	var err error
	if d.Tags, err = q.ListTagsByPackage(ctx, p.ID); err != nil {
		return d, err
	}
	if d.Attached, err = q.ListAttachedFiles(ctx, p.ID); err != nil {
		return d, err
	}
	counts, err := q.CountPackageInstallUsers(ctx, []int64{p.ID})
	if err != nil {
		return d, err
	}
	for _, c := range counts {
		d.InstallCount = c.Users
	}
	return d, nil
}

// GetAttachedFile はパッケージの添付ファイルを返す. 別のパッケージのものなら ErrNotFound.
func (s *Service) GetAttachedFile(ctx context.Context, pkgID, attachedID int64) (db.AttachedFile, error) {
	a, err := s.q.GetAttachedFile(ctx, attachedID)
	if err != nil {
		return a, notFound(err, "attached file")
	}
	if a.PackageID != pkgID {
		return db.AttachedFile{}, ErrNotFound
	}
	return a, nil
}

// Attachment はパッケージに添付するファイル.
type Attachment struct {
	Path string // ローカルのファイル
	Name string // 元のファイル名
	Size int64
	// Type は attached_file.file_type. 空なら拡張子から決める (apk なら "apk").
	Type string
}

// 添付ファイルの種類 (attached_file.file_type).
const (
	FileTypeAPK  = "apk"
	FileTypeDSYM = "dSYM"
)

func attachedTypeFromName(name string) string {
	if pkgfile.Ext(name) == "apk" {
		return FileTypeAPK
	}
	return ""
}

// AttachFile はパッケージにファイルを添付する.
func (s *Service) AttachFile(ctx context.Context, pkg db.Package, file Attachment) (db.AttachedFile, error) {
	if file.Type == "" {
		file.Type = attachedTypeFromName(file.Name)
	}
	tmp := newTempName(pkgfile.Ext(file.Name))
	if err := s.storage.PutFile(ctx, tempKey(tmp), file.Path); err != nil {
		return db.AttachedFile{}, err
	}
	var af db.AttachedFile
	err := s.inUploadTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetApplicationForUpdate(ctx, pkg.AppID); err != nil {
			return notFound(err, "application")
		}
		// 同時に消されたパッケージに添付しない (行とファイルが残ってしまう)
		p, err := q.GetPackageForUpdate(ctx, pkg.ID)
		if err != nil {
			return notFound(err, "package")
		}
		if p.AppID != pkg.AppID {
			return fmt.Errorf("package: %w", ErrNotFound)
		}
		af, err = insertAttachedFile(ctx, q, p, file.Name, file.Size, file.Type, s.Now())
		if err != nil {
			return err
		}
		return s.storage.Move(ctx, tempKey(tmp), AttachedKey(af))
	})
	if err != nil {
		s.deleteFiles(ctx, tempKey(tmp))
		if af.ID != 0 {
			s.deleteFiles(ctx, AttachedKey(af))
		}
		return db.AttachedFile{}, err
	}
	return af, nil
}

func insertAttachedFile(ctx context.Context, q *db.Queries, pkg db.Package, name string, size int64, fileType string, now time.Time) (db.AttachedFile, error) {
	af := db.AttachedFile{
		AppID:            pkg.AppID,
		PackageID:        pkg.ID,
		FileName:         attachedFileName(name),
		OriginalFileName: name,
		FileSize:         size,
		FileType:         fileType,
		Created:          now,
	}
	id, err := q.InsertAttachedFile(ctx, db.InsertAttachedFileParams{
		AppID: af.AppID, PackageID: af.PackageID, FileName: af.FileName, OriginalFileName: af.OriginalFileName,
		FileSize: af.FileSize, FileType: af.FileType, Created: af.Created,
	})
	af.ID = id
	return af, err
}

// DeleteAttachedFile は添付ファイルを消す.
func (s *Service) DeleteAttachedFile(ctx context.Context, af db.AttachedFile) error {
	err := s.inTx(ctx, func(q *db.Queries) error {
		if _, err := q.GetApplicationForUpdate(ctx, af.AppID); err != nil {
			return err
		}
		return q.DeleteAttachedFile(ctx, af.ID)
	})
	if err != nil {
		return err
	}
	s.deleteFiles(ctx, AttachedKey(af))
	return nil
}

// DownloadExpiry はダウンロード URL の有効期間.
const DownloadExpiry = 60 * time.Minute

// PackageDownloadURL はパッケージをダウンロードする URL. android なら, AAB は生成した APK
// (添付ファイル) を返す (無ければ AAB そのもの).
func (s *Service) PackageDownloadURL(ctx context.Context, p db.Package, android bool) (string, error) {
	if android && IsAppBundle(p) {
		files, err := s.q.ListAttachedFiles(ctx, p.ID)
		if err != nil {
			return "", err
		}
		for _, a := range files {
			if a.FileType == FileTypeAPK {
				return s.AttachedDownloadURL(ctx, a)
			}
		}
	}
	name := p.OriginalFileName.String
	if name == "" {
		name = p.FileName
	}
	return s.storage.DownloadURL(ctx, PackageKey(p), DownloadExpiry, name)
}

// AttachedDownloadURL は添付ファイルをダウンロードする URL.
func (s *Service) AttachedDownloadURL(ctx context.Context, a db.AttachedFile) (string, error) {
	return s.storage.DownloadURL(ctx, AttachedKey(a), DownloadExpiry, a.OriginalFileName)
}
