package media

import (
	"reflect"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/resampler"
)

func mustBuffer(t *testing.T, frames []float32, hz int) *buffer {
	t.Helper()
	buf, err := newBuffer(frames, hz)
	if err != nil {
		t.Fatalf("failed to create buffer: %v", err)
	}
	return buf
}

func samplesOf(v int16, n int) samples {
	s := make(samples, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func ramp(pairs int) samples {
	s := make(samples, pairs*2)
	for i := 0; i < pairs; i++ {
		s[i*2], s[i*2+1] = int16(i), int16(i)
	}
	return s
}

func TestNewBuffer(t *testing.T) {
	tests := []struct {
		name    string
		frames  []float32
		hz      int
		wantErr bool
	}{
		{"valid single", []float32{10}, 48000, false},
		{"valid multi", []float32{10, 20}, 48000, false},
		{"hz too low", []float32{10}, 1999, true},
		{"empty frames", []float32{}, 48000, true},
		{"nil frames", nil, 48000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := newBuffer(tt.frames, tt.hz)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if buf != nil {
				buf.close()
			}
		})
	}
}

func TestBufferBucketSizes(t *testing.T) {
	buf := mustBuffer(t, []float32{10, 20}, 48000)
	defer buf.close()

	if len(buf.buckets) != 2 {
		t.Fatalf("got %d buckets, want 2", len(buf.buckets))
	}
	if n := len(buf.buckets[0].mem); n != 960 {
		t.Errorf("bucket[0] = %d, want 960", n)
	}
	if n := len(buf.buckets[1].mem); n != 1920 {
		t.Errorf("bucket[1] = %d, want 1920", n)
	}
}

func TestBufferClose(t *testing.T) {
	buf := mustBuffer(t, []float32{10}, 48000)
	buf.close()
	buf.close() // idempotent
	if buf.resampler != nil {
		t.Error("resampler should be nil after close")
	}
}

func TestBufferWrite(t *testing.T) {
	tests := []struct {
		name   string
		writes []struct {
			v int16
			n int
		}
		want samples
	}{
		{
			name: "overflow triggers callback",
			writes: []struct {
				v int16
				n int
			}{{1, 10}, {2, 20}, {3, 30}},
			want: samples{
				2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
				3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
				3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
				3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
			},
		},
		{
			name: "partial fill",
			writes: []struct {
				v int16
				n int
			}{{1, 3}, {2, 18}, {3, 2}},
			want: samples{1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := mustBuffer(t, []float32{10, 5}, 2000)
			defer buf.close()

			var got samples
			for _, w := range tt.writes {
				buf.write(samplesOf(w.v, w.n), func(s samples, _ float32) { got = s })
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\ngot:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestBufferWriteExact(t *testing.T) {
	buf := mustBuffer(t, []float32{10}, 2000) // 40 samples
	defer buf.close()

	calls := 0
	buf.write(samplesOf(1, 40), func(_ samples, ms float32) {
		calls++
		if ms != 10 {
			t.Errorf("ms = %v, want 10", ms)
		}
	})
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestBufferWriteReturn(t *testing.T) {
	buf := mustBuffer(t, []float32{10}, 2000)
	defer buf.close()

	if n := buf.write(samplesOf(1, 100), func(samples, float32) {}); n != 100 {
		t.Errorf("return = %d, want 100", n)
	}
}

func TestBufferChoose(t *testing.T) {
	buf := mustBuffer(t, []float32{20, 10, 5}, 48000) // 1920, 960, 480
	defer buf.close()

	tests := []struct{ rem, want int }{
		{10000, 2}, {500, 2}, {479, 0}, {0, 0},
	}
	for _, tt := range tests {
		buf.choose(tt.rem)
		if buf.bi != tt.want {
			t.Errorf("choose(%d) = %d, want %d", tt.rem, buf.bi, tt.want)
		}
	}
}

func TestStereoSamples(t *testing.T) {
	tests := []struct {
		hz   int
		ms   float32
		want int
	}{
		{16000, 5, 160},
		{32768, 10, 656},
		{32768, 2.5, 164},
		{32768, 5, 328},
		{44100, 10, 882},
		{48000, 10, 960},
		{48000, 2.5, 240},
	}
	for _, tt := range tests {
		if got := stereoSamples(tt.hz, tt.ms); got != tt.want {
			t.Errorf("stereoSamples(%d, %.0f) = %d, want %d", tt.hz, tt.ms, got, tt.want)
		}
	}
}

func TestStretchPassthrough(t *testing.T) {
	buf := mustBuffer(t, []float32{10}, 48000)
	defer buf.close()

	src := samples{1, 2, 3, 4}
	if res := buf.stretch(src, 4); &res[0] != &src[0] {
		t.Error("expected zero-copy when sizes match")
	}
}

func TestLinear(t *testing.T) {
	t.Run("interpolation", func(t *testing.T) {
		out := make(samples, 8)
		resampler.Linear(out, samples{0, 0, 100, 100})
		if out[2] <= 0 || out[2] >= 100 {
			t.Errorf("middle value %d not interpolated", out[2])
		}
	})

	t.Run("sizes", func(t *testing.T) {
		cases := []struct{ srcPairs, dstSize int }{
			{4, 16}, {8, 8}, {4, 8},
		}
		for _, tc := range cases {
			out := make(samples, tc.dstSize)
			resampler.Linear(out, ramp(tc.srcPairs))
			if len(out) != tc.dstSize {
				t.Errorf("len = %d, want %d", len(out), tc.dstSize)
			}
		}
	})
}

func TestNearest(t *testing.T) {
	tests := []struct {
		src  samples
		want samples
	}{
		{samples{10, 20, 30, 40}, samples{10, 20, 10, 20, 30, 40, 30, 40}},
		{samples{10, 20, 30, 40, 50, 60, 70, 80}, samples{10, 20, 50, 60}},
	}
	for _, tt := range tests {
		out := make(samples, len(tt.want))
		resampler.Nearest(out, tt.src)
		if !reflect.DeepEqual(out, tt.want) {
			t.Errorf("nearest(%v) = %v, want %v", tt.src, out, tt.want)
		}
	}
}

func TestSpeex(t *testing.T) {
	buf := mustBuffer(t, []float32{10}, 48000)
	defer buf.close()

	if err := buf.resample(24000, ResampleSpeex); err != nil {
		t.Fatal(err)
	}

	t.Run("stretch", func(t *testing.T) {
		res := buf.stretch(samplesOf(1000, 960), 480)
		if len(res) != 480 {
			t.Errorf("len = %d, want 480", len(res))
		}
		for _, s := range res {
			if s != 0 {
				return
			}
		}
		t.Error("output is silent")
	})

	t.Run("write", func(t *testing.T) {
		calls := 0
		buf.write(samplesOf(5000, 960), func(s samples, ms float32) {
			calls++
			if len(s) != 480 {
				t.Errorf("len = %d, want 480", len(s))
			}
			if ms != 10 {
				t.Errorf("ms = %v, want 10", ms)
			}
		})
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})
}

func BenchmarkStretch(b *testing.B) {
	src := samplesOf(1000, 1920) // 20ms @ 48kHz

	b.Run("speex", func(b *testing.B) {
		buf, _ := newBuffer([]float32{20}, 48000)
		defer buf.close()
		_ = buf.resample(24000, ResampleSpeex)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.stretch(src, 960)
		}
	})

	b.Run("linear", func(b *testing.B) {
		buf, _ := newBuffer([]float32{20}, 48000)
		defer buf.close()
		_ = buf.resample(24000, ResampleLinear)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.stretch(src, 960)
		}
	})

	b.Run("nearest", func(b *testing.B) {
		buf, _ := newBuffer([]float32{20}, 48000)
		defer buf.close()
		_ = buf.resample(24000, ResampleNearest)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.stretch(src, 960)
		}
	})
}
