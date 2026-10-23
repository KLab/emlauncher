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

	"github.com/KLabTools/emlauncher/e2e/internal/fixture"
)

func main() {
	out := flag.String("o", ".", "output directory")
	flag.Parse()

	files := map[string][]byte{
		"sample.ipa": fixture.IPA("Sample", "com.example.e2e.ios"),
		"sample.apk": fixture.APK("com.example.e2e.android"),
		"sample.aab": fixture.AAB("com.example.e2e.bundle"),
		"icon.png":   fixture.PNG(200, 100, color.RGBA{0x33, 0x99, 0xcc, 0xff}),
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
