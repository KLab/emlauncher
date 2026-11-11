package emlauncher

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"

	"emlauncher/pkgfile"
)

// Upload はアップロードされたファイル (一時ファイルに保存済み).
type Upload struct {
	Name        string // 元のファイル名
	Path        string // 一時ファイルのパス
	ContentType string
	Size        int64
}

// GeneratedFile はパッケージから生成した添付ファイル (AAB の universal APK).
type GeneratedFile struct {
	Name string
	Type string
	Path string
	Size int64
}

// Analysis はパッケージの判別結果.
type Analysis struct {
	Platform   string
	Ext        string
	Identifier sql.NullString
	Generated  []GeneratedFile
}

// Cleanup は生成した一時ファイルを消す.
func (a *Analysis) Cleanup() {
	for _, g := range a.Generated {
		os.Remove(g.Path)
	}
}

// Analyze はプラットフォームと識別子を調べる. AAB なら universal APK を生成する.
func (c *Core) Analyze(ctx context.Context, name, filePath string) (*Analysis, error) {
	platform, ext, err := pkgfile.Detect(name, filePath)
	if err != nil {
		return nil, err
	}
	a := &Analysis{Platform: platform, Ext: ext}
	switch {
	case platform == pkgfile.PlatformIOS:
		id, err := pkgfile.IOSBundleID(filePath)
		if err != nil {
			return nil, err
		}
		a.Identifier = sql.NullString{String: id, Valid: true}
	case platform == pkgfile.PlatformAndroid && ext == "aab":
		apk, err := c.Bundletool.UniversalAPK(ctx, filePath)
		if err != nil {
			return nil, err
		}
		a.Generated = append(a.Generated, GeneratedFile{
			Name: strings.TrimSuffix(name, "aab") + "apk",
			Type: AttachedTypeAPK,
			Path: apk,
		})
		if st, err := os.Stat(apk); err == nil {
			a.Generated[0].Size = st.Size()
		}
		id, err := pkgfile.AndroidPackageName(apk)
		if err != nil {
			a.Cleanup()
			return nil, err
		}
		a.Identifier = sql.NullString{String: id, Valid: true}
	case platform == pkgfile.PlatformAndroid:
		id, err := pkgfile.AndroidPackageName(filePath)
		if err != nil {
			return nil, err
		}
		a.Identifier = sql.NullString{String: id, Valid: true}
	}
	return a, nil
}

// PackageInfo は利用者が入力するパッケージの情報.
type PackageInfo struct {
	Title       string
	Description *string // nil なら NULL
	Tags        []string
	Protect     bool
}

// AttachUpload はパッケージと一緒にアップロードされた添付ファイル (dSYM など).
type AttachUpload struct {
	Upload
	Type string
}

// UploadPackage は API からのアップロード. file は Analyze 済みのもの.
// ファイルはトランザクションの外で temp-data/ に保存し, トランザクション内では移動だけにする
// (大きなファイルの保存中に, アプリの行のロックを持ち続けないため).
func (c *Core) UploadPackage(ctx context.Context, appID int64, info PackageInfo, file Upload, an *Analysis, extra []AttachUpload) (*Package, error) {
	var temps []string
	defer func() { c.deleteKeys(context.WithoutCancel(ctx), temps...) }()
	t := TempPackage{
		Platform:     an.Platform,
		Identifier:   an.Identifier,
		OriginalName: file.Name,
		FileSize:     sql.NullInt64{Int64: file.Size, Valid: true},
		Attached:     []TempAttached{},
	}
	var err error
	if t.TempName, err = c.saveTemp(ctx, file.Path, an.Ext); err != nil {
		return nil, err
	}
	temps = append(temps, TempDir+t.TempName)
	attach := func(name, typ, path string, size int64) error {
		temp, err := c.saveTemp(ctx, path, pkgfile.Ext(name))
		if err != nil {
			return err
		}
		temps = append(temps, TempDir+temp)
		t.Attached = append(t.Attached, TempAttached{FileName: name, TempName: temp, Type: typ, Size: size})
		return nil
	}
	for _, g := range an.Generated {
		if err := attach(g.Name, g.Type, g.Path, g.Size); err != nil {
			return nil, err
		}
	}
	for _, e := range extra {
		if err := attach(e.Name, e.Type, e.Path, e.Size); err != nil {
			return nil, err
		}
	}
	pkg, err := c.commitTemp(ctx, appID, info, t)
	if err == nil {
		temps = nil // 移動済み
	}
	return pkg, err
}

func (c *Core) insertPackage(ctx context.Context, tx *sqlx.Tx, pkg *Package, tagNames []string) error {
	tags, err := GetOrInsertTags(ctx, tx, pkg.AppID, tagNames)
	if err != nil {
		return err
	}
	if err := InsertPackage(ctx, tx, pkg); err != nil {
		return err
	}
	return SetPackageTags(ctx, tx, pkg.ID, tags)
}

func (c *Core) insertAttached(ctx context.Context, tx *sqlx.Tx, pkg *Package, name string, size int64, typ string) (*AttachedFile, error) {
	af := &AttachedFile{
		AppID:            pkg.AppID,
		PackageID:        pkg.ID,
		FileName:         randomString(16) + dotExt(name),
		OriginalFileName: name,
		FileSize:         size,
		FileType:         typ,
		Created:          c.Clock.Now(),
	}
	if err := InsertAttachedFile(ctx, tx, af); err != nil {
		return nil, err
	}
	return af, nil
}

// dotExt は PHP の strrchr($name, '.') と同じく, 最後の "." 以降を返す (無ければ空).
func dotExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// TempUpload は画面からのアップロードの前半 (一時保存) の結果.
type TempUpload struct {
	FileName   string         `json:"file_name"`
	TempName   string         `json:"temp_name"`
	Platform   string         `json:"platform"`
	Identifier *string        `json:"identifier"`
	Attached   []TempAttached `json:"attached_files"`
}

type TempAttached struct {
	FileName string `json:"file_name"`
	TempName string `json:"temp_name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
}

// UploadTemp はファイルを temp-data/ に保存し, 判別結果を返す.
func (c *Core) UploadTemp(ctx context.Context, file Upload) (*TempUpload, error) {
	platform, ext, err := pkgfile.Detect(file.Name, file.Path)
	if err != nil {
		return nil, err
	}
	t := &TempUpload{FileName: file.Name, Platform: platform, Attached: []TempAttached{}}
	t.TempName, err = c.saveTemp(ctx, file.Path, ext)
	if err != nil {
		return nil, err
	}
	an, err := c.Analyze(ctx, file.Name, file.Path)
	if err != nil {
		return nil, err
	}
	defer an.Cleanup()
	if an.Identifier.Valid {
		t.Identifier = &an.Identifier.String
	}
	for _, g := range an.Generated {
		name, err := c.saveTemp(ctx, g.Path, "apk")
		if err != nil {
			return nil, err
		}
		t.Attached = append(t.Attached, TempAttached{FileName: g.Name, TempName: name, Type: g.Type, Size: g.Size})
	}
	return t, nil
}

func (c *Core) saveTemp(ctx context.Context, filePath, ext string) (string, error) {
	name := randomString(16) + "." + ext
	if err := c.Storage.SaveFile(ctx, TempDir+name, filePath); err != nil {
		return "", err
	}
	return name, nil
}

// 一時ファイル名は UploadTemp が作ったもの (ランダム 16 文字 + 拡張子) に限る.
// PHP 版は検証しておらず, 任意のキーのファイルを移動できた.
var tempNameRe = regexp.MustCompile(`^[0-9A-Za-z]{16}\.[^/\\]*$`)

// TempPackage は画面のアップロードの後半で送られてくる内容.
type TempPackage struct {
	TempName     string
	Platform     string
	Identifier   sql.NullString
	OriginalName string
	FileSize     sql.NullInt64
	Attached     []TempAttached
}

// CommitTempPackage は一時保存したファイルをパッケージとして登録する.
func (c *Core) CommitTempPackage(ctx context.Context, appID int64, info PackageInfo, t TempPackage) (*Package, error) {
	if !tempNameRe.MatchString(t.TempName) {
		return nil, fmt.Errorf("%w: invalid temp_name %q", ErrBadRequest, t.TempName)
	}
	for _, a := range t.Attached {
		if !tempNameRe.MatchString(a.TempName) {
			return nil, fmt.Errorf("%w: invalid temp_name %q", ErrBadRequest, a.TempName)
		}
	}
	return c.commitTemp(ctx, appID, info, t)
}

// commitTemp は temp-data/ のファイルをパッケージ (と添付ファイル) として登録する.
func (c *Core) commitTemp(ctx context.Context, appID int64, info PackageInfo, t TempPackage) (*Package, error) {
	var renamed []string
	var pkg *Package
	err := c.uploadTx(ctx, func(tx *sqlx.Tx) error {
		app, err := GetApplicationForUpdate(ctx, tx, appID)
		if err != nil {
			return err
		}
		if app == nil {
			return notFound("application", appID)
		}
		now := c.Clock.Now()
		pkg = &Package{
			AppID:            appID,
			Platform:         t.Platform,
			FileName:         randomString(16) + "." + pkgfile.Ext(t.TempName),
			Title:            info.Title,
			Description:      NullString(info.Description),
			Identifier:       t.Identifier,
			OriginalFileName: sql.NullString{String: t.OriginalName, Valid: true},
			FileSize:         t.FileSize,
			Protect:          info.Protect,
			Created:          now,
		}
		if err := c.insertPackage(ctx, tx, pkg, info.Tags); err != nil {
			return err
		}
		type mv struct{ src, dst string }
		moves := []mv{{TempDir + t.TempName, pkg.FileKey()}}
		for _, a := range t.Attached {
			af, err := c.insertAttached(ctx, tx, pkg, a.FileName, a.Size, a.Type)
			if err != nil {
				return err
			}
			moves = append(moves, mv{TempDir + a.TempName, af.FileKey()})
		}
		for _, m := range moves {
			if err := c.Storage.Rename(ctx, m.src, m.dst); err != nil {
				return err
			}
			renamed = append(renamed, m.dst)
		}
		return app.UpdateLastUpload(ctx, tx, sql.NullTime{Time: pkg.Created, Valid: true})
	})
	if err != nil {
		c.deleteKeys(context.WithoutCancel(ctx), renamed...)
		return nil, err
	}
	return pkg, nil
}

// DeletePackage はパッケージを削除する. 削除したパッケージと添付ファイルを返す.
// 最新のパッケージを消した場合は application.last_upload を 1 つ前のものに戻す.
func (c *Core) DeletePackage(ctx context.Context, appID, pkgID int64) (*Package, []AttachedFile, error) {
	var pkg *Package
	var attached []AttachedFile
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		app, err := GetApplicationForUpdate(ctx, tx, appID)
		if err != nil {
			return err
		}
		pkg, err = GetPackageForUpdate(ctx, tx, pkgID)
		if err != nil {
			return err
		}
		if app == nil || pkg == nil || pkg.AppID != appID {
			return notFound("package", pkgID)
		}
		if attached, err = AttachedFilesByPackage(ctx, tx, pkgID); err != nil {
			return err
		}
		if err := pkg.Delete(ctx, tx); err != nil {
			return err
		}
		if app.LastUpload.Valid && app.LastUpload.Time.Equal(pkg.Created) {
			newest, err := NewestPackage(ctx, tx, appID)
			if err != nil {
				return err
			}
			last := sql.NullTime{}
			if newest != nil {
				last = sql.NullTime{Time: newest.Created, Valid: true}
			}
			return app.UpdateLastUpload(ctx, tx, last)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	keys := []string{pkg.FileKey()}
	for _, a := range attached {
		keys = append(keys, a.FileKey())
	}
	c.deleteKeys(ctx, keys...)
	return pkg, attached, nil
}

// PackageEdit は変更する項目 (nil は変更しない).
type PackageEdit struct {
	Title       *string
	Description *string
	Protect     *bool
	Tags        *[]string
}

func (c *Core) EditPackage(ctx context.Context, appID, pkgID int64, e PackageEdit) (*Package, error) {
	var pkg *Package
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		if _, err := GetApplicationForUpdate(ctx, tx, appID); err != nil {
			return err
		}
		var err error
		pkg, err = GetPackageForUpdate(ctx, tx, pkgID)
		if err != nil {
			return err
		}
		if pkg == nil || pkg.AppID != appID {
			return notFound("package", pkgID)
		}
		if e.Title != nil {
			pkg.Title = *e.Title
		}
		if e.Description != nil {
			pkg.Description = NullString(e.Description)
		}
		if e.Protect != nil {
			pkg.Protect = *e.Protect
		}
		if err := pkg.UpdateInfo(ctx, tx); err != nil {
			return err
		}
		if e.Tags != nil {
			tags, err := GetOrInsertTags(ctx, tx, appID, *e.Tags)
			if err != nil {
				return err
			}
			return SetPackageTags(ctx, tx, pkg.ID, tags)
		}
		return nil
	})
	return pkg, err
}

// AttachFile はパッケージにファイルを添付する.
func (c *Core) AttachFile(ctx context.Context, pkg *Package, file Upload) (*AttachedFile, error) {
	var af *AttachedFile
	saved := false
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		if _, err := GetApplicationForUpdate(ctx, tx, pkg.AppID); err != nil {
			return err
		}
		var err error
		af, err = c.insertAttached(ctx, tx, pkg, file.Name, file.Size, AttachedTypeFromExt(pkgfile.Ext(file.Name)))
		if err != nil {
			return err
		}
		if err := c.Storage.SaveFile(ctx, af.FileKey(), file.Path); err != nil {
			return err
		}
		saved = true
		return nil
	})
	if err != nil {
		if saved {
			c.deleteKeys(context.WithoutCancel(ctx), af.FileKey())
		}
		return nil, err
	}
	return af, nil
}

// DetachFile は添付ファイルを削除する.
func (c *Core) DetachFile(ctx context.Context, af *AttachedFile) error {
	err := c.tx(ctx, func(tx *sqlx.Tx) error {
		if _, err := GetApplicationForUpdate(ctx, tx, af.AppID); err != nil {
			return err
		}
		return af.Delete(ctx, tx)
	})
	if err != nil {
		return err
	}
	c.deleteKeys(ctx, af.FileKey())
	return nil
}

// AndroidDownloadTarget は Android 端末でインストールするときのファイル:
// AAB で生成した APK が添付されていればそれ, なければパッケージそのもの.
func AndroidDownloadTarget(ctx context.Context, q Q, pkg *Package) (key, name string, err error) {
	if pkg.IsAndroidAppBundle() {
		afs, err := AttachedFilesByPackage(ctx, q, pkg.ID)
		if err != nil {
			return "", "", err
		}
		for _, a := range afs {
			if a.FileType == AttachedTypeAPK {
				return a.FileKey(), a.OriginalFileName, nil
			}
		}
	}
	return pkg.FileKey(), pkg.OriginalFileName.String, nil
}
