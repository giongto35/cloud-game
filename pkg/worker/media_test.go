package worker

import (
	"image"
	"math/rand"
	"testing"

	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/vpx"
)

func TestEncoders(t *testing.T) {
	tests := []struct {
		n      int
		w, h   int
		codec  encoder.VideoCodec
		frames int
	}{
		{n: 3, w: 1920, h: 1080, codec: encoder.H264, frames: 60},
		{n: 3, w: 1920, h: 1080, codec: encoder.VP8, frames: 60},
	}

	for _, test := range tests {
		a := genTestImage(test.w, test.h, rand.New(rand.NewSource(int64(1))).Float32())
		b := genTestImage(test.w, test.h, rand.New(rand.NewSource(int64(2))).Float32())
		for i := 0; i < test.n; i++ {
			run(test.w, test.h, test.codec, test.frames, a, b, t)
		}
	}
}

func BenchmarkH264(b *testing.B) { run(1920, 1080, encoder.H264, b.N, nil, nil, b) }
func BenchmarkVP8(b *testing.B)  { run(1920, 1080, encoder.VP8, b.N, nil, nil, b) }

func run(w, h int, cod encoder.VideoCodec, count int, a *image.RGBA, b *image.RGBA, backend testing.TB) {
	var enc encoder.Encoder
	if cod == encoder.H264 {
		enc, _ = h264.NewEncoder(w, h)
	} else {
		enc, _ = vpx.NewEncoder(w, h, nil)
	}

	ve := encoder.NewVideoEncoder(enc, w, h, nil)
	defer ve.Stop()

	if a == nil {
		a = genTestImage(w, h, rand.New(rand.NewSource(int64(1))).Float32())
	}
	if b == nil {
		b = genTestImage(w, h, rand.New(rand.NewSource(int64(2))).Float32())
	}

	for i := 0; i < count; i++ {
		im := a
		if i%2 == 0 {
			im = b
		}
		out := ve.Encode(im)
		if out == nil {
			backend.Fatalf("encoder closed abnormally")
		}
	}
}

func genTestImage(w, h int, seed float32) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: w, Y: h}})
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			i := img.PixOffset(x, y)
			s := img.Pix[i : i+4 : i+4]
			s[0] = uint8(seed * 255)
			s[1] = uint8(seed * 255)
			s[2] = uint8(seed * 255)
			s[3] = 0xff
		}
	}
	return img
}
