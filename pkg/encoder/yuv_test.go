package encoder

import (
	"image"
	"image/color"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestYuv(t *testing.T) {
	size1, size2 := 2, 4
	img := generateImage(size1, size2, randomColor())
	buf1 := NewYuvBuffer(size1, size2)
	buf2 := NewYuvBuffer(size1, size2)

	buf1.FromRGBA(img)
	old := buf1.data
	buf2.FromRGBA2(img)
	ne := buf2.data

	if !reflect.DeepEqual(old, ne) {

		t.Logf("old: %v, new: %v", old, ne)

		t.Fatalf("not right")
	}
}

func generateImage(w, h int, pixelColor color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, pixelColor)
		}
	}
	return img
}

func randomColor() color.RGBA {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	return color.RGBA{R: uint8(rnd.Intn(255)), G: uint8(rnd.Intn(255)), B: uint8(rnd.Intn(255)), A: 255}
}

func BenchmarkOld(b *testing.B) {
	benchmarkConverter(1920, 1080, 0, b)
}

func BenchmarkNew(b *testing.B) {
	benchmarkConverter(1920, 1080, 1, b)
}

func benchmarkConverter(w, h int, t int, b *testing.B) {
	buf := NewYuvBuffer(w, h)
	buf2 := NewYuvBuffer(w, h)

	image1 := genTestImage(w, h, rand.New(rand.NewSource(int64(1))).Float32())
	image2 := genTestImage(w, h, rand.New(rand.NewSource(int64(2))).Float32())

	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}

		if t == 0 {
			buf.FromRGBA(im)
		} else {
			buf2.FromRGBA2(im)
		}
	}
}

func genTestImage(w, h int, seed float32) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: w, Y: h}})
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			col := color.RGBA{R: uint8(seed * 255), G: uint8(seed * 255), B: uint8(seed * 255), A: 0xff}
			img.Set(x, y, col)
		}
	}
	return img
}
