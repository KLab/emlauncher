package pkgfile

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"regexp"

	"howett.net/plist"
)

var infoPlistPath = regexp.MustCompile(`^Payload/[^/]*\.app/Info\.plist$`)

// maxInfoPlistSize は Info.plist として読む大きさの上限.
const maxInfoPlistSize = 16 << 20

// IPABundleID は IPA の Payload/*.app/Info.plist (XML / バイナリ) から CFBundleIdentifier を返す.
func IPABundleID(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("ipa: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !infoPlistPath.MatchString(f.Name) {
			continue
		}
		data, err := readZipFile(f, maxInfoPlistSize)
		if err != nil {
			return "", fmt.Errorf("ipa: %s: %w", f.Name, err)
		}
		var info struct {
			CFBundleIdentifier string `plist:"CFBundleIdentifier"`
		}
		if _, err := plist.Unmarshal(data, &info); err != nil {
			return "", fmt.Errorf("ipa: %s: %w", f.Name, err)
		}
		// CFBundleIdentifier が無ければ空 (PHP 版と同じく identifier は NULL になる)
		return info.CFBundleIdentifier, nil
	}
	return "", errors.New("ipa: Info.plist not found")
}

func readZipFile(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("too large (%d bytes)", f.UncompressedSize64)
	}
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, limit))
}
