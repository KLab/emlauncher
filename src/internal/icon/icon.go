// Package icon はアプリのアイコン画像を 144x144 の PNG にする.
package icon

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // 読める形式: png / gif / jpeg / webp
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Size はアイコンの幅と高さ (PHP 版と同じ).
const Size = 144

// maxPixels は受け付ける画像の画素数の上限 (巨大な画像で メモリを使い切らないため).
const maxPixels = 8192 * 8192

// ToPNG は画像 (PNG / GIF / JPEG / WebP) を Size x Size に拡大縮小した PNG にする.
// 縦横比は保たない (PHP 版の Imagick::scaleImage(144, 144) と同じ).
func ToPNG(data []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, fmt.Errorf("icon: image too large (%dx%d)", cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, Size, Size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	return buf.Bytes(), nil
}
