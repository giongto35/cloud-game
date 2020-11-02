package audio

import (
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"reflect"
	"testing"
)

type testRun struct {
	srcFq int
	dstFq int

	frameDurationMs int

	in  []int16
	out []int16
}

func TestEncoder(t *testing.T) {
	tests := []testRun{
		{
			frameDurationMs: 4,
			srcFq:           1000,
			dstFq:           3100,
			in: []int16{
				0, 0,
				1, 1,
				3, 3,
				0, 0,
			},
			out: []int16{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},
		//{
		//	target: 0,
		//	srcFq:  0,
		//	dstFq:  0,
		//},
	}

	//

	o := float64(config.AudioFrameSamplesTarget) * 33161 / float64(config.AudioFrequencyTarget)
	b := config.AudioFrequencyTarget / 1000 * config.AudioFrameDurationMs * config.AudioChannels
	c := float64(33161) / 1000 * config.AudioFrameDurationMs * config.AudioChannels

	t.Logf("%v - %v - %v\n", o, b, c)

	for _, test := range tests {
		frame := test.dstFq / 1000 * test.frameDurationMs * 2
		result := Resample(test.in, frame, test.srcFq, test.dstFq)
		if !reflect.DeepEqual(result, test.out) {
			t.Errorf("%v != %v", test.out, result)
		}
	}
}
