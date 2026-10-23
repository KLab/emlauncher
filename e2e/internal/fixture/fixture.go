// Package fixture はテストでアップロードするパッケージファイル (IPA / APK / AAB) や
// アイコン画像を生成する. 外部ファイルに依存せず, 中身を決定的に作る.
package fixture

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// IPA は Payload/{name}.app/Info.plist を含む最小の IPA を返す.
func IPA(name, bundleID string) []byte {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
</dict>
</plist>
`, bundleID, name)
	return zipFiles([]zipEntry{
		{"Payload/" + name + ".app/Info.plist", []byte(plist)},
		{"Payload/" + name + ".app/" + name, []byte("dummy executable")},
	})
}

// APK はバイナリ XML の AndroidManifest.xml と空の resources.arsc を含む最小の APK を返す.
func APK(packageName string) []byte {
	return zipFiles([]zipEntry{
		{"AndroidManifest.xml", BinaryManifest(packageName)},
		{"resources.arsc", emptyResourceTable()},
		{"classes.dex", []byte("dex\n035\x00")},
	})
}

// AAB は bundletool で universal APK を生成できる最小の Android App Bundle を返す.
func AAB(packageName string) []byte {
	return zipFiles([]zipEntry{
		{"BundleConfig.pb", bundleConfigProto("1.17.2")},
		{"base/manifest/AndroidManifest.xml", protoManifest(packageName)},
	})
}

// PNG は単色の w x h の PNG 画像を返す.
func PNG(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type zipEntry struct {
	name string
	data []byte
}

func zipFiles(entries []zipEntry) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			panic(err)
		}
		if _, err := w.Write(e.data); err != nil {
			panic(err)
		}
	}
	if err := zw.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
