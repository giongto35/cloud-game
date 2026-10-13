package imgutil

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, c color.Color) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.Set(x, y, c)
		}
	}
	return m
}

func TestProcessSquareAvatar(t *testing.T) {
	var in bytes.Buffer
	if err := jpeg.Encode(&in, solid(800, 400, color.RGBA{200, 30, 30, 255}), nil); err != nil {
		t.Fatal(err)
	}
	out, err := Process(in.Bytes(), Options{Square: true, MaxSide: 128})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("result is not a PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 128 {
		t.Fatalf("want 128x128, got %v", b)
	}
	r, _, _, _ := img.At(64, 64).RGBA()
	if r>>8 < 180 {
		t.Fatalf("colour lost: %d", r>>8)
	}
}

func TestProcessKeepsRatio(t *testing.T) {
	var in bytes.Buffer
	if err := png.Encode(&in, solid(1000, 500, color.White)); err != nil {
		t.Fatal(err)
	}
	out, err := Process(in.Bytes(), Options{MaxSide: 400})
	if err != nil {
		t.Fatal(err)
	}
	img, _ := png.Decode(bytes.NewReader(out))
	if b := img.Bounds(); b.Dx() != 400 || b.Dy() != 200 {
		t.Fatalf("want 400x200, got %v", b)
	}
}

func TestProcessRejectsGarbage(t *testing.T) {
	if _, err := Process([]byte("<svg onload=alert(1)>"), Options{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
