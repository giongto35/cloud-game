package media

import (
	"image"
	"math/rand"
	"reflect"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/vpx"
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
		enc, _ = h264.NewEncoder(w, h, nil)
	} else {
		enc, _ = vpx.NewEncoder(w, h, nil)
	}

	logger.SetGlobalLevel(logger.Disabled)
	ve := encoder.NewVideoEncoder(enc, w, h, 8, l)
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
		nums[i] = int16(rand.Intn(10))
	}
	return nums
}

type bufWrite struct {
	sample int16
	len    int
}

func TestBufferWrite(t *testing.T) {
	tests := []struct {
		bufLen int
		writes []bufWrite
		expect samples
	}{
		{
			bufLen: 20,
			writes: []bufWrite{
				{sample: 1, len: 10},
				{sample: 2, len: 20},
				{sample: 3, len: 30},
			},
			expect: samples{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
		},
		{
			bufLen: 11,
			writes: []bufWrite{
				{sample: 1, len: 3},
				{sample: 2, len: 18},
				{sample: 3, len: 2},
			},
			expect: samples{3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3},
		},
	}

	for _, test := range tests {
		var lastResult samples
		buf := newBuffer(test.bufLen)
		for _, w := range test.writes {
			buf.write(samplesOf(w.sample, w.len), func(s samples) { lastResult = s })
		}
		if !reflect.DeepEqual(test.expect, lastResult) {
			t.Errorf("not expted buffer, %v != %v", lastResult, test.expect)
		}
	}
}

func BenchmarkBufferWrite(b *testing.B) {
	fn := func(_ samples) {}
	l := 1920
	buf := newBuffer(l)
	samples1 := samplesOf(1, l/2)
	samples2 := samplesOf(2, l*2)
	for i := 0; i < b.N; i++ {
		buf.write(samples1, fn)
		buf.write(samples2, fn)
	}
}

func samplesOf(v int16, len int) (s samples) {
	s = make(samples, len)
	for i := range s {
		s[i] = v
	}
	return
}

func Test_frame(t *testing.T) {
	type args struct {
		hz    int
		frame int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "mGBA", args: args{hz: 32768, frame: 10}, want: 654},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frame(tt.args.hz, tt.args.frame); got != tt.want {
				t.Errorf("frame() = %v, want %v", got, tt.want)
			}
		})
	}
}
