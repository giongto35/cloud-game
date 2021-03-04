package encoder

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
	"time"
)

func TestYuv(t *testing.T) {
	size1, size2 := 128, 128
	img := generateImage(size1, size2, randomColor())
	buf := NewYuvBuffer(size1, size2)

	buf.FromRGBA(img)

	if len(buf.data) == 0 {
		t.Fatalf("couldn't convert")
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

func benchmarkConverter(w, h int, chroma ChromaPos, b *testing.B) {
	b.StopTimer()
	buf := NewYuvBuffer(w, h)
	buf.pos = chroma

	image1 := genTestImage(w, h, rand.New(rand.NewSource(int64(1))).Float32())
	image2 := genTestImage(w, h, rand.New(rand.NewSource(int64(2))).Float32())

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}
		buf.FromRGBA(im)
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
