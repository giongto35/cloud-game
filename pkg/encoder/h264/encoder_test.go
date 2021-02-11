package h264

import (
	"image"
	color2 "image/color"
	"math/rand"
	"testing"

	encoder2 "github.com/giongto35/cloud-game/v2/pkg/encoder"
)

//func TestVpxEncoder(t *testing.T) {
//	encoder, _ := vpxencoder.NewVpxEncoder(320, 240, 20, 1500, 5)
//	in, out := encoder.GetInputChan(), encoder.GetOutputChan()
//
//	encoder.Stop()
//}

func BenchmarkH264(b *testing.B) {
	w, h := 1920, 1080
	encoder, _ := NewH264Encoder(w, h, 1)
	in, out := encoder.GetInputChan(), encoder.GetOutputChan()

	image1 := genTestImage(w, h, rand.New(rand.NewSource(int64(1))).Float32())
	image2 := genTestImage(w, h, rand.New(rand.NewSource(int64(2))).Float32())

	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}
		in <- encoder2.InFrame{Image: im}
		<-out
	}
	encoder.Stop()
}

func genTestImage(w, h int, seed float32) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: w, Y: h}})
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			var color color2.Color
			color = color2.RGBA{R: uint8(seed * 255), G: uint8(seed * 255), B: uint8(seed * 255), A: 0xff}
			img.Set(x, y, color)
		}
	}
	return img
}
