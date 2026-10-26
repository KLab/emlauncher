// QR コードの画像の生成.

package emlauncher

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/makiuchi-d/gozxing/qrcode/decoder"
)

// qrMargin は QR コードの周りの余白 (px). PHP 版 (endroid/qr-code の setMargin(10)) と同じ.
const qrMargin = 10

// qrPNG は content を誤り訂正レベル H の QR コードにした PNG を返す.
// QR コード部分は size x size (モジュール数より小さければモジュール数), その周りに qrMargin の余白を付ける.
func qrPNG(content string, size int) ([]byte, error) {
	hints := map[gozxing.EncodeHintType]any{
		gozxing.EncodeHintType_ERROR_CORRECTION: decoder.ErrorCorrectionLevel_H,
		gozxing.EncodeHintType_CHARACTER_SET:    "UTF-8",
		gozxing.EncodeHintType_MARGIN:           0, // 余白は下で付ける (gozxing の余白はモジュール単位のため)
	}
	code, err := qrcode.NewQRCodeWriter().Encode(content, gozxing.BarcodeFormat_QR_CODE, size, size, hints)
	if err != nil {
		return nil, err
	}
	b := code.Bounds()
	img := image.NewGray(image.Rect(0, 0, b.Dx()+2*qrMargin, b.Dy()+2*qrMargin))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(img, b.Add(image.Pt(qrMargin, qrMargin)), code, b.Min, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
