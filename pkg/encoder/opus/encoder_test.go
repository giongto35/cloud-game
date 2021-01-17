package opus

import (
	"math/rand"
	"testing"
	"time"
)

var resampleData []int16

func init() {
	rand.Seed(time.Now().Unix())
	l := rand.Perm(getBufferLength(32768))
	for _, n := range l {
		resampleData = append(resampleData, int16(n))
	}
}

func BenchmarkResample(b *testing.B) {
	//sr := 48000
	//bs := getBufferLength(sr)
	resampler := SoxResampler{
	}
	for i := 0; i < b.N; i++ {
		resampler.Init(32768, 48000)
		pcm := resampler.Resample(resampleData, 48000/1000*1*2)
		b.Logf("Written: %v", len(pcm))
		resampler.backend.Close()
	}
}

func getBufferLength(sampleRate int) int { return sampleRate / 1000 * 1 * 2 }
