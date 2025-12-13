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
	tests := []struct {
		bufLen int
		writes []bufWrite
		expect samples
	}{
		{
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 10},
				{sample: 2, len: 20},
				{sample: 3, len: 30},
			},
			expect: samples{
				2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3,
				3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
				3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
			},
		},
		{
			bufLen: 2000,
			writes: []bufWrite{
				{sample: 1, len: 3},
				{sample: 2, len: 18},
				{sample: 3, len: 2},
			},
			expect: samples{1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		},
	}

	for _, test := range tests {
		var lastResult samples
		buf, err := newBuffer([]float32{10, 5}, test.bufLen)
		if err != nil {
			t.Fatalf("oof, %v", err)
		}
		for _, w := range test.writes {
			buf.write(samplesOf(w.sample, w.len),
				func(s samples, ms float32) { lastResult = s },
			)
		}
		if !reflect.DeepEqual(test.expect, lastResult) {
			t.Errorf("not expted buffer, %v != %v, %v", lastResult, test.expect, len(buf.buckets))
		}
	}
}

func BenchmarkBufferWrite(b *testing.B) {
	fn := func(_ samples, _ float32) {}
	l := 2000
	buf, err := newBuffer([]float32{10}, l)
	if err != nil {
		b.Fatalf("oof: %v", err)
	}
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
