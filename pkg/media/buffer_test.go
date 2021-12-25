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
		expect Samples
	}{
		{
			bufLen: 20,
			writes: []bufWrite{
				{sample: 1, len: 10},
				{sample: 2, len: 20},
				{sample: 3, len: 30},
			},
			expect: Samples{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
		},
		{
			bufLen: 11,
			writes: []bufWrite{
				{sample: 1, len: 3},
				{sample: 2, len: 18},
				{sample: 3, len: 2},
			},
			expect: Samples{3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3},
		},
	}

	for _, test := range tests {
		var lastResult Samples
		buf := NewBuffer(test.bufLen)
		for _, w := range test.writes {
			buf.Write(samplesOf(w.sample, w.len), func(s Samples) { lastResult = s })
		}
		if !reflect.DeepEqual(test.expect, lastResult) {
			t.Errorf("not expted buffer, %v != %v", lastResult, test.expect)
		}
	}
}

func BenchmarkBufferWrite(b *testing.B) {
	fn := func(_ Samples) {}
	l := 1920
	buf := NewBuffer(l)
	samples1 := samplesOf(1, l/2)
	samples2 := samplesOf(2, l*2)
	for i := 0; i < b.N; i++ {
		buf.Write(samples1, fn)
		buf.Write(samples2, fn)
	}
}

func samplesOf(v int16, len int) (s Samples) {
	s = make(Samples, len)
	for i := range s {
		s[i] = v
	}
	return
}
