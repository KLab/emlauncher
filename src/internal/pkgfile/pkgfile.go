// Package pkgfile はアップロードされたパッケージファイル (IPA / APK / AAB) を調べる.
//
// プラットフォームの判定, バンドル ID / パッケージ名の取り出し, AAB から universal APK の生成を行う.
// アプリ本体 (DB やストレージ) には依存しない.
package pkgfile

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// Platform は package.platform の値.
type Platform string

const (
	Android Platform = "Android"
	IOS     Platform = "iOS"
	Unknown Platform = "unknown"
)

// Kind はファイルの種類.
type Kind int

const (
	KindUnknown Kind = iota
	KindIPA
	KindAPK
	KindAAB
)

// Platform は種類に対応するプラットフォーム.
func (k Kind) Platform() Platform {
	switch k {
	case KindIPA:
		return IOS
	case KindAPK, KindAAB:
		return Android
	}
	return Unknown
}

// Ext は元のファイル名の拡張子 (ドットを含まない). PHP の pathinfo(PATHINFO_EXTENSION) と同じ.
func Ext(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return ""
	}
	return name[i+1:]
}

// Detect はファイル名の拡張子と中身 (zip かどうか) から種類を判定する.
// 拡張子が ipa / apk / aab でも, zip でなければ KindUnknown.
func Detect(path, originalName string) (Kind, error) {
	var kind Kind
	switch Ext(originalName) {
	case "ipa":
		kind = KindIPA
	case "apk":
		kind = KindAPK
	case "aab":
		kind = KindAAB
	default:
		return KindUnknown, nil
	}
	zip, err := isZip(path)
	if err != nil {
		return KindUnknown, err
	}
	if !zip {
		return KindUnknown, nil
	}
	return kind, nil
}

var zipMagic = []byte("PK\x03\x04")

func isZip(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, len(zipMagic))
	if _, err := io.ReadFull(f, head); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(head, zipMagic), nil
}
