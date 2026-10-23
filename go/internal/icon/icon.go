// Package icon はアプリのアイコン画像を 144x144 の PNG に変換する.
// PHP 版の Imagick::scaleImage(144, 144) と同じく, 縦横比は保たずに 144x144 にする.
package icon

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	// 対応する入力形式
	_ "image/gif"
	_ "image/jpeg"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const Size = 144

func Convert(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, Size, Size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
