package media

import (
	"image"
	"math/rand/v2"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

var l = logger.New(false)

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
		a := genTestImage(test.w, test.h, rand.Float32())
		b := genTestImage(test.w, test.h, rand.Float32())
		for i := 0; i < test.n; i++ {
			run(test.w, test.h, test.codec, test.frames, a, b, t)
		}
	}
}

func BenchmarkH264(b *testing.B) { run(640, 480, encoder.H264, b.N, nil, nil, b) }
func BenchmarkVP8(b *testing.B)  { run(1920, 1080, encoder.VP8, b.N, nil, nil, b) }

func run(w, h int, cod encoder.VideoCodec, count int, a *image.RGBA, b *image.RGBA, backend testing.TB) {
	conf := config.Video{
		Codec:   string(cod),
		Threads: 0,
		H264: struct {
			Mode     string
			Crf      uint8
			MaxRate  int
			BufSize  int
			LogLevel int32
			Preset   string
			Profile  string
			Tune     string
		}{
			Crf:      30,
			LogLevel: 0,
			Preset:   "ultrafast",
			Profile:  "baseline",
			Tune:     "zerolatency",
		},
		Vpx: struct {
			Bitrate          uint
			KeyframeInterval uint
		}{
			Bitrate:          1000,
			KeyframeInterval: 5,
		},
	}

	logger.SetGlobalLevel(logger.Disabled)
	ve, err := encoder.NewVideoEncoder(w, h, w, h, 1, conf, l)
	if err != nil {
		backend.Error(err)
		return
	}
	defer ve.Stop()

	if a == nil {
		a = genTestImage(w, h, rand.Float32())
	}
	if b == nil {
		b = genTestImage(w, h, rand.Float32())
	}

	for i := range count {
		im := a
		if i%2 == 0 {
			im = b
		}
		out := ve.Encode(encoder.InFrame{
			Data:   im.Pix,
			Stride: im.Stride,
			W:      im.Bounds().Dx(),
			H:      im.Bounds().Dy(),
		})
		if out == nil {
			backend.Fatalf("encoder closed abnormally")
		}
	}
}

func genTestImage(w, h int, seed float32) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: w, Y: h}})
	for x := range w {
		for y := range h {
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
