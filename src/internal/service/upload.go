package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"emlauncher/internal/db"
	"emlauncher/internal/pkgfile"
	"emlauncher/internal/storage"
)

// アップロードは 2 段階で行う.
//
//  1. ファイルを調べ (プラットフォーム, バンドル ID, AAB なら universal APK を生成),
//     ストレージの一時ファイル (temp-data/) に置く. 時間がかかるのでロックは取らない.
//  2. トランザクションの中でアプリをロックし, DB に行を作って一時ファイルを正式なキーに移す.
//
// 画面からのアップロードは 1 を ajax で, 2 をフォームの送信で行う. API は 1 と 2 を続けて行う.

// LocalFile はリクエストで受け取ったローカルのファイル.
type LocalFile struct {
	Path string
	Name string // 元のファイル名
	Size int64
}

// NewPackage はアップロードするパッケージの情報.
type NewPackage struct {
	Title       string
	Description *string // nil なら NULL
	Tags        []string
	Protect     bool
	// Notify ならアプリをインストールしたユーザに通知する. BaseURL はメールに書く URL の起点.
	Notify  bool
	BaseURL string
}

// stagedFile はストレージの一時ファイルに置いたもの.
type stagedFile struct {
	TempName string
	Name     string // 元のファイル名
	Size     int64
	Type     string // 添付ファイルの種類 (attached_file.file_type)
}

// stagedPackage は一時ファイルに置いたパッケージと添付ファイル.
type stagedPackage struct {
	Platform   pkgfile.Platform
	Ext        string
	Identifier sql.NullString
	Main       stagedFile
	Attached   []stagedFile
}

func (sp *stagedPackage) tempKeys() []string {
	var keys []string
	for _, f := range append([]stagedFile{sp.Main}, sp.Attached...) {
		if f.TempName != "" {
			keys = append(keys, tempKey(f.TempName))
		}
	}
	return keys
}

// inspected はパッケージファイルを調べた結果.
type inspected struct {
	kind       pkgfile.Kind
	identifier string
	// AAB から生成した APK (呼び出し側で消す)
	generatedAPK string
}

// inspect はパッケージを調べる. AAB なら prebuilt (送られた APK) が無ければ universal APK を生成する.
func (s *Service) inspect(ctx context.Context, f LocalFile, prebuilt *LocalFile) (inspected, error) {
	kind, err := pkgfile.Detect(f.Path, f.Name)
	if err != nil {
		return inspected{}, err
	}
	res := inspected{kind: kind}
	switch kind {
	case pkgfile.KindIPA:
		res.identifier, err = pkgfile.IPABundleID(f.Path)
	case pkgfile.KindAPK:
		res.identifier, err = pkgfile.APKPackageName(f.Path)
	case pkgfile.KindAAB:
		apk := ""
		if prebuilt != nil {
			apk = prebuilt.Path
		} else {
			apk, err = s.universalAPK(ctx, f.Path)
			if err != nil {
				return inspected{}, err
			}
			res.generatedAPK = apk
		}
		res.identifier, err = pkgfile.APKPackageName(apk)
	}
	if err != nil {
		if res.generatedAPK != "" {
			os.Remove(res.generatedAPK)
		}
		return inspected{}, &InvalidError{Msg: err.Error()}
	}
	return res, nil
}

func (s *Service) universalAPK(ctx context.Context, aab string) (string, error) {
	f, err := os.CreateTemp("", "emlauncher-*.apk")
	if err != nil {
		return "", err
	}
	f.Close()
	if err := s.bundletool.UniversalAPK(ctx, aab, f.Name()); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// stage はローカルのファイルを一時ファイルとしてストレージに置く.
func (s *Service) stage(ctx context.Context, path, name, fileType string) (stagedFile, error) {
	st, err := os.Stat(path)
	if err != nil {
		return stagedFile{}, err
	}
	tmp := newTempName(pkgfile.Ext(name))
	if err := s.storage.PutFile(ctx, tempKey(tmp), path); err != nil {
		return stagedFile{}, err
	}
	return stagedFile{TempName: tmp, Name: name, Size: st.Size(), Type: fileType}, nil
}

// apkNameForBundle は AAB から生成した APK のファイル名 (拡張子を apk にしたもの).
func apkNameForBundle(aabName string) string {
	return strings.TrimSuffix(aabName, "aab") + "apk"
}

// stagePackage はパッケージを調べて一時ファイルに置く. extra は一緒に添付するファイル.
func (s *Service) stagePackage(ctx context.Context, f LocalFile, prebuiltAPK *LocalFile, extra ...stagedSource) (sp stagedPackage, err error) {
	if prebuiltAPK != nil {
		// apk (Go 版で足した項目) は AAB と一緒に送られたときだけ使い, それ以外は無視する
		if kind, _ := pkgfile.Detect(f.Path, f.Name); kind != pkgfile.KindAAB {
			prebuiltAPK = nil
		}
	}
	info, err := s.inspect(ctx, f, prebuiltAPK)
	if err != nil {
		return sp, err
	}
	if info.generatedAPK != "" {
		defer os.Remove(info.generatedAPK)
	}
	defer func() {
		if err != nil {
			s.deleteFiles(ctx, sp.tempKeys()...)
		}
	}()

	sp = stagedPackage{Platform: info.kind.Platform(), Ext: pkgfile.Ext(f.Name), Identifier: nullString(info.identifier)}
	if sp.Main, err = s.stage(ctx, f.Path, f.Name, ""); err != nil {
		return sp, err
	}
	switch {
	case prebuiltAPK != nil:
		extra = append([]stagedSource{{prebuiltAPK.Path, prebuiltAPK.Name, FileTypeAPK}}, extra...)
	case info.generatedAPK != "":
		extra = append([]stagedSource{{info.generatedAPK, apkNameForBundle(f.Name), FileTypeAPK}}, extra...)
	}
	for _, e := range extra {
		a, err := s.stage(ctx, e.path, e.name, e.fileType)
		if err != nil {
			return sp, err
		}
		sp.Attached = append(sp.Attached, a)
	}
	return sp, nil
}

type stagedSource struct {
	path, name, fileType string
}

// UploadOptions は API のアップロードで一緒に送られるファイル.
type UploadOptions struct {
	DSYM *LocalFile // dSYM (添付ファイルにする)
	APK  *LocalFile // AAB と一緒に送られた APK (AAB から生成する代わりに添付する)
}

// Upload はパッケージをアップロードする (API).
func (s *Service) Upload(ctx context.Context, appID int64, f LocalFile, opts UploadOptions, np NewPackage) (PackageDetail, error) {
	if np.Title == "" {
		return PackageDetail{}, invalidf("title is required")
	}
	var extra []stagedSource
	if opts.DSYM != nil {
		extra = append(extra, stagedSource{opts.DSYM.Path, opts.DSYM.Name, FileTypeDSYM})
	}
	sp, err := s.stagePackage(ctx, f, opts.APK, extra...)
	if err != nil {
		return PackageDetail{}, err
	}
	return s.commitPackage(ctx, appID, sp, np)
}

// StagedUpload は画面のアップロードの 1 段階目の結果 (ajax の応答).
type StagedUpload struct {
	FileName   string
	TempName   string
	Platform   pkgfile.Platform
	Identifier *string // 取り出せなければ nil
	Attached   []StagedAttachment
}

// StagedAttachment は一時ファイルに置いた添付ファイル.
type StagedAttachment struct {
	FileName string
	TempName string
	Size     int64
	Type     string
}

// StageUpload は画面のアップロードの 1 段階目. ファイルを調べて一時ファイルに置く.
func (s *Service) StageUpload(ctx context.Context, f LocalFile) (StagedUpload, error) {
	sp, err := s.stagePackage(ctx, f, nil)
	if err != nil {
		return StagedUpload{}, err
	}
	su := StagedUpload{FileName: f.Name, TempName: sp.Main.TempName, Platform: sp.Platform}
	if sp.Identifier.Valid {
		su.Identifier = &sp.Identifier.String
	}
	for _, a := range sp.Attached {
		su.Attached = append(su.Attached, StagedAttachment{FileName: a.Name, TempName: a.TempName, Size: a.Size, Type: a.Type})
	}
	return su, nil
}

// CommitUpload は画面のアップロードの 2 段階目. StageUpload の結果 (フォームで送り返されたもの) から
// パッケージを作る. 大きさはストレージの一時ファイルから求める.
func (s *Service) CommitUpload(ctx context.Context, appID int64, su StagedUpload, np NewPackage) (db.Package, error) {
	if np.Title == "" || su.TempName == "" {
		return db.Package{}, invalidf("title and temp_name are required")
	}
	if !slices.Contains([]pkgfile.Platform{pkgfile.Android, pkgfile.IOS, pkgfile.Unknown}, su.Platform) {
		return db.Package{}, invalidf("invalid platform %q", su.Platform)
	}
	// file_name の拡張子は一時ファイルのものにする (PHP 版と同じ)
	// identifier は送られた値をそのまま保存する (PHP 版と同じ)
	sp := stagedPackage{Platform: su.Platform, Ext: pkgfile.Ext(su.TempName), Identifier: optionalString(su.Identifier)}
	var err error
	if sp.Main, err = s.stagedFile(ctx, su.TempName, su.FileName, ""); err != nil {
		return db.Package{}, err
	}
	for _, a := range su.Attached {
		sf, err := s.stagedFile(ctx, a.TempName, a.FileName, a.Type)
		if err != nil {
			return db.Package{}, err
		}
		sp.Attached = append(sp.Attached, sf)
	}
	d, err := s.commitPackage(ctx, appID, sp, np)
	return d.Package, err
}

func (s *Service) stagedFile(ctx context.Context, tempName, name, fileType string) (stagedFile, error) {
	if !validTempName(tempName) {
		return stagedFile{}, invalidf("invalid temp_name %q", tempName)
	}
	if fileType != "" && fileType != FileTypeAPK && fileType != FileTypeDSYM {
		return stagedFile{}, invalidf("invalid attached file type %q", fileType)
	}
	size, err := s.storage.Size(ctx, tempKey(tempName))
	if errors.Is(err, storage.ErrNotFound) {
		return stagedFile{}, invalidf("temporary file %q not found", tempName)
	}
	if err != nil {
		return stagedFile{}, err
	}
	return stagedFile{TempName: tempName, Name: name, Size: size, Type: fileType}, nil
}

// commitPackage はアプリをロックしてパッケージの行を作り, 一時ファイルを正式なキーに移す.
// 失敗したら一時ファイルと移したファイルを消す.
func (s *Service) commitPackage(ctx context.Context, appID int64, sp stagedPackage, np NewPackage) (PackageDetail, error) {
	var d PackageDetail
	var moved []string
	err := s.inUploadTx(ctx, func(q *db.Queries) error {
		app, err := q.GetApplicationForUpdate(ctx, appID)
		if err != nil {
			return notFound(err, "application")
		}
		tags, _, err := tagsByName(ctx, q, appID, np.Tags, true)
		if err != nil {
			return err
		}
		now := s.Now()
		p := db.Package{
			AppID:            appID,
			Platform:         string(sp.Platform),
			FileName:         packageFileName(sp.Ext),
			Title:            np.Title,
			Description:      optionalString(np.Description),
			Identifier:       sp.Identifier,
			OriginalFileName: sql.NullString{String: sp.Main.Name, Valid: true},
			FileSize:         sql.NullInt64{Int64: sp.Main.Size, Valid: true},
			Protect:          boolInt(np.Protect),
			Created:          now,
		}
		p.ID, err = q.InsertPackage(ctx, db.InsertPackageParams{
			AppID: p.AppID, Platform: p.Platform, FileName: p.FileName, Title: p.Title, Description: p.Description,
			Identifier: p.Identifier, OriginalFileName: p.OriginalFileName, FileSize: p.FileSize, Protect: p.Protect,
			Created: p.Created,
		})
		if err != nil {
			return err
		}
		if err := setPackageTags(ctx, q, p.ID, tags); err != nil {
			return err
		}
		d = PackageDetail{Package: p, Tags: tags, Attached: []db.AttachedFile{}}
		if d.Tags == nil {
			d.Tags = []db.Tag{}
		}

		if err := s.storage.Move(ctx, tempKey(sp.Main.TempName), PackageKey(p)); err != nil {
			return err
		}
		moved = append(moved, PackageKey(p))
		for _, a := range sp.Attached {
			af, err := insertAttachedFile(ctx, q, p, a.Name, a.Size, a.Type, now)
			if err != nil {
				return err
			}
			if err := s.storage.Move(ctx, tempKey(a.TempName), AttachedKey(af)); err != nil {
				return err
			}
			moved = append(moved, AttachedKey(af))
			d.Attached = append(d.Attached, af)
		}
		return touchLastUpload(ctx, q, app, sql.NullTime{Time: now, Valid: true})
	})
	if err != nil {
		s.deleteFiles(ctx, append(moved, sp.tempKeys()...)...)
		if errors.Is(err, ErrNotFound) {
			return PackageDetail{}, err
		}
		return PackageDetail{}, fmt.Errorf("upload: %w", err)
	}
	if np.Notify {
		if err := s.notifyUpload(ctx, appID, d, np.BaseURL); err != nil {
			// 通知できなくてもアップロードは成功にする
			s.log.Error("upload notification", "app_id", appID, "package_id", d.ID, "err", err)
		}
	}
	return d, nil
}
