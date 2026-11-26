package service

import (
	"fmt"
	"regexp"
	"strings"

	"emlauncher/internal/db"
	"emlauncher/internal/pkgfile"
	"emlauncher/internal/randstr"
)

// ストレージのキーの構成 (PHP 版と同じ).
//
//	アイコン      app-icons/{app_id}/{rand16}.png
//	パッケージ    package/{app_id}/{pkg_id}_{package.file_name}
//	添付ファイル  package/{app_id}/{pkg_id}_{attached_id}_{attached_file.file_name}
//	一時ファイル  temp-data/{temp_name}
const tempPrefix = "temp-data/"

func newIconKey(appID int64) string {
	return fmt.Sprintf("app-icons/%d/%s.png", appID, randstr.New(16))
}

// PackageKey はパッケージのファイルのキー.
func PackageKey(p db.Package) string {
	return fmt.Sprintf("package/%d/%d_%s", p.AppID, p.ID, p.FileName)
}

// AttachedKey は添付ファイルのキー.
func AttachedKey(a db.AttachedFile) string {
	return fmt.Sprintf("package/%d/%d_%d_%s", a.AppID, a.PackageID, a.ID, a.FileName)
}

func tempKey(tempName string) string { return tempPrefix + tempName }

// packageFileName は package.file_name. "{rand16}.{拡張子}" (拡張子が無くてもドットを付ける. PHP 版と同じ).
func packageFileName(ext string) string {
	return randstr.New(16) + "." + ext
}

// attachedFileName は attached_file.file_name. "{rand16}{.拡張子}" (拡張子が無ければドットも無い).
func attachedFileName(originalName string) string {
	return randstr.New(16) + dotExt(originalName)
}

// newTempName は一時ファイルの名前. "{rand16}.{拡張子}".
func newTempName(ext string) string {
	return randstr.New(16) + "." + ext
}

func dotExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// tempNamePattern は一時ファイルとして作った名前. 画面から送られた名前はこれに一致するものだけ受け付ける
// (任意のキーのファイルを移動させないため).
var tempNamePattern = regexp.MustCompile(`^[0-9A-Za-z]{16}\.[^/\\.\x00-\x1f]*$`)

func validTempName(name string) bool {
	return tempNamePattern.MatchString(name)
}

// IsAppBundle はパッケージが Android App Bundle (aab) か.
func IsAppBundle(p db.Package) bool {
	return p.Platform == string(pkgfile.Android) && pkgfile.Ext(p.OriginalFileName.String) == "aab"
}
