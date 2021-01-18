package opus

import (
	"math/rand"
	"testing"
	"time"
)

var resampleData []int16

func init() {
	rand.Seed(time.Now().Unix())
	l := rand.Perm(getBufferLength(44000))
	for _, n := range l {
		resampleData = append(resampleData, int16(n))
	}
}

func BenchmarkResample(b *testing.B) {
	sr := 48000
	bs := getBufferLength(sr)
	for i := 0; i < b.N; i++ {
		resampleFn(resampleData, bs)
	}
}

func getBufferLength(sampleRate int) int { return sampleRate * 20 / 1000 * 2 }
