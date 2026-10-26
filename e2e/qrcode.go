package e2e

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// DecodeQR は PNG の QR コードを読み取り, 中身と画像の大きさを返す.
func DecodeQR(t testing.TB, b []byte) (text string, width, height int) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("not png: %v", err)
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("decode qr: %v", err)
	}
	return res.GetText(), img.Bounds().Dx(), img.Bounds().Dy()
}
