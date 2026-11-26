package pkgfile

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/shogo82148/androidbinary"
)

const maxManifestSize = 16 << 20

// APKPackageName は APK の AndroidManifest.xml (バイナリ XML) から manifest 要素の package 属性を返す.
func APKPackageName(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("apk: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "AndroidManifest.xml" {
			continue
		}
		data, err := readZipFile(f, maxManifestSize)
		if err != nil {
			return "", fmt.Errorf("apk: AndroidManifest.xml: %w", err)
		}
		name, err := manifestPackage(data)
		if err != nil {
			return "", fmt.Errorf("apk: AndroidManifest.xml: %w", err)
		}
		return name, nil
	}
	return "", errors.New("apk: AndroidManifest.xml not found")
}

// manifestPackage はバイナリ XML の manifest 要素の package 属性を返す.
func manifestPackage(data []byte) (string, error) {
	xf, err := androidbinary.NewXMLFile(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	var m struct {
		Package string `xml:"package,attr"`
	}
	if err := xml.NewDecoder(xf.Reader()).Decode(&m); err != nil {
		return "", err
	}
	if m.Package == "" {
		return "", errors.New("package attribute not found")
	}
	return m.Package, nil
}
