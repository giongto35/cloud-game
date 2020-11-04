package audio

import (
	"math/rand"
	"testing"
)

type testRun struct {
	srcFq int
	dstFq int

	frameDurationMs int

	in  []int16
	out []int16
}

func TestResampling(t *testing.T) {
	//rand.Seed(time.Now().Unix())
	//
	//n := 20
	//input := randPCM(n)
	//
	//p := NewAudioProcessor(NewOpusEncoder(config.DefaultOpusCfg()))
	//
	//k := GetSampleCount(20000, 2, 2)
	//
	//a := resample(input, k, 13231, 48000)
	//
	//if !reflect.DeepEqual(a, b) {
	//	t.Errorf("\n%v\n%v-%v\n%v\n\n%v", input, len(a), len(b), a, b)
	//}
}

func randPCM(n int) []int16 {
	result := make([]int16, n)
	i := 0
	for i < n {
		result[i] = int16(rand.Int31n(255))
		i++
	}
	return result
}

func TestEncoder(t *testing.T) {
	//tests := []testRun{
	//	{
	//		frameDurationMs: 4,
	//		srcFq:           1000,
	//		dstFq:           3100,
	//		in: []int16{
	//			0, 0,
	//			1, 1,
	//			3, 3,
	//			0, 0,
	//		},
	//		out: []int16{
	//			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	//		},
	//	},
	//	//{
	//	//	target: 0,
	//	//	srcFq:  0,
	//	//	dstFq:  0,
	//	//},
	//}

	//for _, test := range tests {
	//frame := test.dstFq / 1000 * test.frameDurationMs * 2
	//result := Resample(test.in, frame, test.srcFq, test.dstFq)
	//if !reflect.DeepEqual(result, test.out) {
	//	t.Errorf("%v != %v", test.out, result)
	//}
	//}
}
