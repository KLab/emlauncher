// genfixture はテスト用のパッケージファイルを書き出す (フィクスチャの手動確認用).
//
//	go run ./cmd/genfixture -o DIR
package main

import (
	"flag"
	"image/color"
	"log"
	"os"
	"path/filepath"

	e2e "emlauncher-e2e"
)

func main() {
	out := flag.String("o", ".", "output directory")
	flag.Parse()

	files := map[string][]byte{
		"sample.ipa": e2e.IPA("Sample", "com.example.e2e.ios"),
		"sample.apk": e2e.APK("com.example.e2e.android"),
		"sample.aab": e2e.AAB("com.example.e2e.bundle"),
		"icon.png":   e2e.PNG(200, 100, color.RGBA{0x33, 0x99, 0xcc, 0xff}),
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(*out, name), data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
