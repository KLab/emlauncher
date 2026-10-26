// アプリのアイコン画像を 144x144 の PNG に変換する.
// PHP 版の Imagick::scaleImage(144, 144) と同じく, 縦横比は保たずに 144x144 にする.

package emlauncher

import (
	"bytes"
	"fmt"
	"image"
	// 対応する入力形式
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const iconSize = 144

func convertIcon(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("icon: %w", err)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
