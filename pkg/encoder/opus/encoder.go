package opus

import "gopkg.in/hraban/opus.v2"

type Encoder struct {
	*opus.Encoder

	buffer          Buffer
	channels        int
	inFrequency     int
	outFrequency    int
	resampleBufSize int
}

func NewEncoder(inputSampleRate, outputSampleRate, channels int, options ...func(*Encoder) error) (Encoder, error) {
	encoder, err := opus.NewEncoder(
		outputSampleRate,
		channels,
		// be aware that low delay option is not optimized for voice
		opus.AppRestrictedLowdelay,
	)
	if err != nil {
		return Encoder{}, err
	}
	enc := &Encoder{
		Encoder:      encoder,
		channels:     channels,
		inFrequency:  inputSampleRate,
		outFrequency: outputSampleRate,
	}

	_ = enc.SetMaxBandwidth(opus.Fullband)
	_ = enc.SetBitrateToAuto()
	_ = enc.SetComplexity(10)

	for _, option := range options {
		err := option(enc)
		if err != nil {
			return Encoder{}, err
		}
	}
	return *enc, nil
}

func SampleBuffer(ms int, resampling bool) func(*Encoder) error {
	return func(e *Encoder) (err error) {
		e.buffer = Buffer{Data: make([]int16, e.inFrequency*ms/1000*e.channels)}
		if resampling {
			e.resampleBufSize = e.outFrequency * ms / 1000 * e.channels
		}
		return
	}
}

func (e *Encoder) BufferWrite(samples []int16) (written int) { return e.buffer.Write(samples) }

func (e *Encoder) BufferEncode() ([]byte, error) { return e.Encode(e.buffer.Data) }

func (e *Encoder) BufferFull() bool { return e.buffer.Full() }

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	if e.resampleBufSize > 0 {
		pcm = resampleFn(pcm, e.resampleBufSize)
	}
	data := make([]byte, 1024)
	n, err := e.Encoder.Encode(pcm, data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

// resampleFn does a simple linear interpolation of audio samples.
func resampleFn(pcm []int16, size int) []int16 {
	r, l, audio := make([]int16, size/2), make([]int16, size/2), make([]int16, size)
	// ratio is basically the destination sample rate
	// divided by the origin sample rate (i.e. 48000/44100)
	ratio := float32(size) / float32(len(pcm))
	for i, n := 0, len(pcm)-1; i < n; i += 2 {
		idx := int(float32(i/2) * ratio)
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
