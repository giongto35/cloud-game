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

	for i := 0; i < count; i++ {
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

func TestResampleStretch(t *testing.T) {
	type args struct {
		pcm  samples
		size int
	}
	tests := []struct {
		name string
		args args
		want []int16
	}{
		//1764:1920
		{name: "", args: args{pcm: gen(1764), size: 1920}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rez2 := tt.args.pcm.stretch(tt.args.size)
			if rez2[0] != tt.args.pcm[0] || rez2[1] != tt.args.pcm[1] ||
				rez2[len(rez2)-1] != tt.args.pcm[len(tt.args.pcm)-1] ||
				rez2[len(rez2)-2] != tt.args.pcm[len(tt.args.pcm)-2] {
				t.Logf("%v\n%v", tt.args.pcm, rez2)
				t.Errorf("2nd is wrong (2)")
			}
		})
	}
}

func BenchmarkResampler(b *testing.B) {
	pcm := samples(gen(1764))
	size := 1920
	for i := 0; i < b.N; i++ {
		pcm.stretch(size)
	}
}

func gen(l int) []int16 {
	nums := make([]int16, l)
	for i := range nums {
		nums[i] = int16(rand.IntN(10))
	}
	return nums
}

func TestFrame(t *testing.T) {
	type args struct {
		hz    int
		frame float32
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "mGBA", args: args{hz: 32768, frame: 10}, want: 656},
		{name: "mGBA", args: args{hz: 32768, frame: 5}, want: 328},
		{name: "mGBA", args: args{hz: 32768, frame: 2.5}, want: 164},
		{name: "nes", args: args{hz: 48000, frame: 2.5}, want: 240},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frame(tt.args.hz, tt.args.frame); got != tt.want {
				t.Errorf("frame() = %v, want %v", got, tt.want)
			}
		})
	}
}
