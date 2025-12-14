package media

import (
	"reflect"
	"testing"
)

type bufWrite struct {
	sample int16
	len    int
}

func TestBufferWrite(t *testing.T) {
	// At 2000Hz stereo:
	// 5ms = 2000 * 0.005 * 2 = 20 samples
	// 10ms = 2000 * 0.01 * 2 = 40 samples

	tests := []struct {
		frames []float32
		bufLen int
		writes []bufWrite
		expect samples
	}{
		{
			frames: []float32{5, 10},
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 20},
				{sample: 2, len: 40},
				{sample: 3, len: 60},
			},
			expect: samples(rep(1, 20).add(2, 40).add(3, 40).add(3, 20)),
		},
		{
			frames: []float32{5, 10},
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 6},
				{sample: 2, len: 36},
				{sample: 3, len: 4},
			},
			expect: samples(rep(1, 6).add(2, 34)),
		},
		{
			frames: []float32{5, 10},
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 40},
			},
			expect: samples(rep(1, 40)),
		},
		{
			frames: []float32{5, 10},
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 100},
			},
			expect: samples(rep(1, 40).add(1, 40).add(1, 20)),
		},
		{
			frames: []float32{5},
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 10},
				{sample: 2, len: 15},
			},
			expect: samples(rep(1, 10).add(2, 10)),
		},
	}

	for i, test := range tests {
		var results samples
		buf, err := newBuffer(test.frames, test.bufLen)
		if err != nil {
			t.Fatalf("test %d: %v", i, err)
		}
		for _, w := range test.writes {
			buf.write(samplesOf(w.sample, w.len), func(s samples, ms float32) {
				tmp := make(samples, len(s))
				copy(tmp, s)
				results = append(results, tmp...)
			})
		}
		if !reflect.DeepEqual(test.expect, results) {
			t.Errorf("test %d:\ngot  %v (len=%d)\nwant %v (len=%d)", i, results, len(results), test.expect, len(test.expect))
		}
	}
}

func BenchmarkBufferWrite(b *testing.B) {
	fn := func(_ samples, _ float32) {}
	buf, _ := newBuffer([]float32{10}, 2000)
	s1 := samplesOf(1, 1000)
	s2 := samplesOf(2, 4000)
	for i := 0; i < b.N; i++ {
		buf.write(s1, fn)
		buf.write(s2, fn)
	}
}

// helpers

func samplesOf(v int16, l int) samples {
	s := make(samples, l)
	for i := range s {
		s[i] = v
	}
	return s
}

type builder samples

func rep(v int16, n int) builder {
	return builder(samplesOf(v, n))
}

func (b builder) add(v int16, n int) builder {
	return append(b, samplesOf(v, n)...)
}
