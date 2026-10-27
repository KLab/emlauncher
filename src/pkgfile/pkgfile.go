// Package pkgfile はアップロードされたパッケージ (IPA / APK / AAB) を判別し, 識別子を取り出す.
package pkgfile

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/shogo82148/androidbinary"
	"howett.net/plist"
)

const (
	PlatformAndroid = "Android"
	PlatformIOS     = "iOS"
	PlatformUnknown = "unknown"
)

// Ext はファイル名の拡張子 (ドット無し). PHP の pathinfo(PATHINFO_EXTENSION) と同じ.
func Ext(name string) string {
	e := path.Ext(path.Base(strings.ReplaceAll(name, "\\", "/")))
	return strings.TrimPrefix(e, ".")
}

// Detect は元のファイル名と中身からプラットフォームを判別する (PHP 版の PackageDb::getPackageInfo).
// zip であり, 拡張子が apk / aab なら Android, ipa なら iOS.
func Detect(originalName, filePath string) (platform, ext string, err error) {
	ext = Ext(originalName)
	f, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	isZip := n == 4 && bytes.Equal(head, []byte("PK\x03\x04"))
	switch {
	case isZip && (ext == "apk" || ext == "aab"):
		return PlatformAndroid, ext, nil
	case isZip && ext == "ipa":
		return PlatformIOS, ext, nil
	}
	return PlatformUnknown, ext, nil
}

// PHP 版の mb_ereg_match('^Payload/[^/]*.app/Info.plist$') と同じ ("." は任意の 1 文字).
var infoPlistRe = regexp.MustCompile(`^Payload/[^/]*.app/Info.plist$`)

// IOSBundleID は IPA の Info.plist から CFBundleIdentifier を返す.
func IOSBundleID(ipaPath string) (string, error) {
	zr, err := zip.OpenReader(ipaPath)
	if err != nil {
		return "", fmt.Errorf("ipa: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !infoPlistRe.MatchString(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("ipa: %w", err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", fmt.Errorf("ipa: %w", err)
		}
		var info struct {
			CFBundleIdentifier string `plist:"CFBundleIdentifier"`
		}
		if _, err := plist.Unmarshal(b, &info); err != nil {
			return "", fmt.Errorf("ipa: Info.plist: %w", err)
		}
		return info.CFBundleIdentifier, nil
	}
	return "", errors.New("ipa: Info.plist file not found")
}

// AndroidPackageName は APK の AndroidManifest.xml から package 属性を返す.
func AndroidPackageName(apkPath string) (string, error) {
	zr, err := zip.OpenReader(apkPath)
	if err != nil {
		return "", fmt.Errorf("apk: %w", err)
	}
	defer zr.Close()
	f, err := zr.Open("AndroidManifest.xml")
	if err != nil {
		return "", fmt.Errorf("apk: %w", err)
	}
	b, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return "", fmt.Errorf("apk: %w", err)
	}
	xf, err := androidbinary.NewXMLFile(bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("apk: AndroidManifest.xml: %w", err)
	}
	var m struct {
		Package string `xml:"package,attr"`
	}
	if err := xml.NewDecoder(xf.Reader()).Decode(&m); err != nil {
		return "", fmt.Errorf("apk: AndroidManifest.xml: %w", err)
	}
	if m.Package == "" {
		return "", errors.New("apk: package name not found")
	}
	return m.Package, nil
}

// BundletoolConfig は bundletool の設定 (設定ファイルの apkfile. PHP 版と同じ).
type BundletoolConfig struct {
	Java       string `yaml:"java"`
	Bundletool string `yaml:"bundletool"`
	Keystore   string `yaml:"keystore"`
	KSPass     string `yaml:"kspass"`
	KeyAlias   string `yaml:"keyalias"`
	KeyPass    string `yaml:"keypass"`
	AAPT2      string `yaml:"aapt2"`
}

// Bundletool は App Bundle から universal APK を作る.
type Bundletool struct {
	conf BundletoolConfig
}

func NewBundletool(c BundletoolConfig) *Bundletool { return &Bundletool{conf: c} }

// UniversalAPK は aabPath から universal APK を作り, 一時ファイルのパスを返す. 呼び出し側が削除すること.
func (b *Bundletool) UniversalAPK(ctx context.Context, aabPath string) (string, error) {
	dir, err := os.MkdirTemp("", "emlauncher-aab-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	apks := dir + "/out.apks"

	java := strings.Fields(b.conf.Java)
	if len(java) == 0 {
		java = []string{"java"}
	}
	args := append(java[1:], "-jar", b.conf.Bundletool, "build-apks", "--mode=universal",
		"--bundle="+aabPath,
		"--output="+apks,
		"--ks="+b.conf.Keystore,
		"--ks-pass="+b.conf.KSPass,
		"--ks-key-alias="+b.conf.KeyAlias,
		"--key-pass="+b.conf.KeyPass,
	)
	if b.conf.AAPT2 != "" {
		args = append(args, "--aapt2="+b.conf.AAPT2)
	}
	out, err := exec.CommandContext(ctx, java[0], args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bundletool error: %v: %s", err, out)
	}

	zr, err := zip.OpenReader(apks)
	if err != nil {
		return "", fmt.Errorf("bundletool output: %w", err)
	}
	defer zr.Close()
	src, err := zr.Open("universal.apk")
	if err != nil {
		return "", fmt.Errorf("bundletool output: %w", err)
	}
	defer src.Close()
	dst, err := os.CreateTemp("", "apk")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dst.Name())
		return "", err
	}
	if err := dst.Close(); err != nil {
		os.Remove(dst.Name())
		return "", err
	}
	return dst.Name(), nil
}

// KeystoreInfo は再署名に使うキーストアの情報 (keytool -list -v) を返す (/doc/keystore 用).
func (b *Bundletool) KeystoreInfo(ctx context.Context) (string, error) {
	kind, pass, _ := strings.Cut(b.conf.KSPass, ":")
	switch kind {
	case "pass":
	case "file":
		p, err := os.ReadFile(pass)
		if err != nil {
			return "", err
		}
		pass = strings.TrimRight(string(p), "\r\n")
	default:
		return "", fmt.Errorf("invalid config format: apkfile.kspass")
	}
	out, err := exec.CommandContext(ctx, "keytool", "-list", "-v",
		"-keystore", b.conf.Keystore, "-alias", b.conf.KeyAlias, "-storepass", pass).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("keytool: %v: %s", err, out)
	}
	return string(out), nil
}
