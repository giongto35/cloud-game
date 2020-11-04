package audio

import (
	"math"
	"testing"
)

func TestSampleCount(t *testing.T) {
	tests := []struct {
		sr      int
		ch      int
		time    float64
		samples int
	}{
		{sr: 48000, ch: 2, time: 2.5, samples: 240},
		{sr: 48000, ch: 2, time: 5, samples: 480},
		{sr: 48000, ch: 2, time: 10, samples: 960},
		{sr: 48000, ch: 2, time: 20, samples: 1920},
		{sr: 48000, ch: 2, time: 40, samples: 3840},
		{sr: 48000, ch: 2, time: 60, samples: 5760},
	}

	for _, test := range tests {
		samples := GetSampleCount(test.sr, test.ch, test.time)
		if samples != test.samples {
			t.Errorf("Sample count mismatch for %v %v != %v", test, samples, test.samples)
		}
	}
}

func TestDefaultResampling(t *testing.T) {
	// 10ms 100Hz 10KHz stereo sine
	wave10k := makeSinWave(100, 10000, 2, 10)
	// 10ms 100Hz 21KHz stereo sine
	wave21k := makeSinWave(100, 21000, 2, 10)
	// resampled 10kHz -> 21kHz
	waveUp := resample(wave10k, GetSampleCount(21000, 2, 10), 10000, 21000)

	t.Logf("\n%v\n%v", wave21k, waveUp)
}

// makeSinWave creates a 16Bit PCM sine wave of given duration in milliseconds.
func makeSinWave(frequency int, sampleRate int, channels int, duration int) []int16 {
	samples := make([]int16, sampleRate*duration/1000*channels)
	for i, end := 0, len(samples); i < end; i += channels {
		sample := math.Sin(2.0 * math.Pi * float64(i) / float64(sampleRate/frequency))
		samples[i] = int16(sample * math.MaxInt16)
		for c := channels - 1; c > 0; c-- {
			samples[i+c] = samples[i]
		}
	}
	return samples
}
