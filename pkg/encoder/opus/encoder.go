package opus

import (
	"gopkg.in/hraban/opus.v2"
)

type Encoder struct {
	*opus.Encoder

	frequency  int
	resampling bool
	// resampleRatio is destination sample rate divided by origin sample rate (i.e. 48000/44100).
	resampleRatio float32
	resampleRate  int
	bufferSize    int
}

func NewEncoder(frequency int, channels int, bufferSize int) (Encoder, error) {
	enc, err := opus.NewEncoder(
		frequency,
		channels,
		// be aware that low delay option is not optimized for voice
		opus.AppRestrictedLowdelay,
	)
	if err != nil {
		return Encoder{}, err
	}

	_ = enc.SetMaxBandwidth(opus.Fullband)
	_ = enc.SetBitrateToAuto()
	_ = enc.SetComplexity(10)

	return Encoder{Encoder: enc, bufferSize: bufferSize, frequency: frequency}, nil
}

func (e *Encoder) GetBuffer() []int16 {
	sampleRate := e.frequency
	if e.resampling {
		sampleRate = e.resampleRate
	}
	return make([]int16, e.bufferSize*sampleRate/e.frequency)
}

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	data := make([]byte, 1024)
	if e.resampling {
		pcm = e.resample(pcm, e.bufferSize)
	}
	n, err := e.Encoder.Encode(pcm, data)
	if err != nil {
		return nil, err
	}
	data = data[:n]
	return data, nil
}

func (e *Encoder) SetResample(sourceSampleRate int) {
	e.resampling = true
	e.resampleRatio = float32(e.frequency) / float32(sourceSampleRate)
	e.resampleRate = sourceSampleRate
}

// resample does a simple linear interpolation of audio samples.
func (e *Encoder) resample(pcm []int16, size int) []int16 {
	r, l, audio := make([]int16, size/2), make([]int16, size/2), make([]int16, size)
	for i, n := 0, len(pcm)-1; i < n; i += 2 {
		idx := int(float32(i/2) * e.resampleRatio)
		r[idx], l[idx] = pcm[i], pcm[i+1]
	}
	for i, n := 1, len(r); i < n; i++ {
		if r[i] == 0 {
			r[i] = r[i-1]
		}
		if l[i] == 0 {
			l[i] = l[i-1]
		}
	}
	for i := 0; i < size-1; i += 2 {
		audio[i], audio[i+1] = r[i/2], l[i/2]
	}
	return audio
}
