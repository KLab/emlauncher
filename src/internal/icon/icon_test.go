package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestToPNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{0x33, 0x99, 0xcc, 0xff})
		}
	}
	for name, enc := range map[string]func(*bytes.Buffer) error{
		"png":  func(b *bytes.Buffer) error { return png.Encode(b, src) },
		"jpeg": func(b *bytes.Buffer) error { return jpeg.Encode(b, src, nil) },
	} {
		var in bytes.Buffer
		if err := enc(&in); err != nil {
			t.Fatal(err)
		}
		out, err := ToPNG(in.Bytes())
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		img, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s: output is not png: %v", name, err)
		}
		if b := img.Bounds(); b.Dx() != Size || b.Dy() != Size {
			t.Errorf("%s: size = %v", name, b)
		}
	}
	if _, err := ToPNG([]byte("not an image")); err == nil {
		t.Fatal("invalid image accepted")
	}
}
