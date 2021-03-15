package room

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/codec"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx"
)

func BenchmarkH264(b *testing.B) {
	benchmarkEncoder(1920, 1080, codec.H264, b)
}

func BenchmarkVP8(b *testing.B) {
	benchmarkEncoder(1920, 1080, codec.VPX, b)
}

func benchmarkEncoder(w, h int, cod codec.VideoCodec, b *testing.B) {
	var enc encoder.Encoder

	if cod == codec.H264 {
		enc, _ = h264.NewEncoder(w, h)
	} else {
		enc, _ = vpx.NewEncoder(w, h)
	}

	pipe := encoder.NewVideoPipe(enc, w, h)
	go pipe.Start()
	defer pipe.Stop()

	image1 := genTestImage(w, h, rand.New(rand.NewSource(int64(1))).Float32())
	image2 := genTestImage(w, h, rand.New(rand.NewSource(int64(2))).Float32())

	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}
		pipe.Input <- encoder.InFrame{Image: im}
		select {
		case <-pipe.Output:
		case <-time.After(5 * time.Second):
			b.Fatalf("encoder didn't produce an image")
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
