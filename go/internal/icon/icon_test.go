package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestConvert(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 300, 100))
	for x := 0; x < 300; x++ {
		for y := 0; y < 100; y++ {
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
		out, err := Convert(in.Bytes())
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
	if _, err := Convert([]byte("not an image")); err == nil {
		t.Error("no error for invalid image")
	}
}
