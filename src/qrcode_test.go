package emlauncher

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// decodeQR は PNG の QR コードを読み取る.
func decodeQR(t *testing.T, b []byte) string {
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
	return res.GetText()
}

func TestQRPNG(t *testing.T) {
	for _, tc := range []struct {
		content string
		size    int
	}{
		{"https://emlauncher.example.com/app?id=1", 150},
		{"https://emlauncher.example.com/package/install?token=0123456789abcdefghijABCDEFGHIJ01", 250},
		{"https://emlauncher.example.com/package/guestpass_install?token=0123456789abcdefghijABCDEFGHIJ01", 300},
		{"https://emlauncher.example.com/app?id=1&q=日本語", 150},
	} {
		b, err := qrPNG(tc.content, tc.size)
		if err != nil {
			t.Fatalf("%s: %v", tc.content, err)
		}
		if got := decodeQR(t, b); got != tc.content {
			t.Errorf("decoded %q, want %q", got, tc.content)
		}
		img, _ := png.Decode(bytes.NewReader(b))
		if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != tc.size+2*qrMargin || h != w {
			t.Errorf("%s: image size %dx%d, want %dx%d", tc.content, w, h, tc.size+2*qrMargin, tc.size+2*qrMargin)
		}
	}
}

// サイズがモジュール数より小さくても読み取れる大きさで作る.
func TestQRPNGTooSmall(t *testing.T) {
	content := "https://emlauncher.example.com/package/install?token=0123456789abcdefghijABCDEFGHIJ01"
	b, err := qrPNG(content, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeQR(t, b); got != content {
		t.Errorf("decoded %q", got)
	}
}
