package opus

import (
	"fmt"
	"log"
)

type Encoder struct {
	*LibOpusEncoder

	channels     int
	inFrequency  int
	outFrequency int
	// OPUS output buffer, 1K should be enough
	outBufferSize int

	buffer          Buffer
	onFullBuffer    func(data []byte)
	resampleBufSize int
}

func NewEncoder(inputSampleRate, outputSampleRate, channels int, options ...func(*Encoder) error) (Encoder, error) {
	encoder, err := NewOpusEncoder(
		outputSampleRate,
		channels,
		// be aware that low delay option is not optimized for voice
		AppRestrictedLowdelay,
	)
	if err != nil {
		return Encoder{}, err
	}
	enc := &Encoder{
		LibOpusEncoder: encoder,
		buffer:         Buffer{Data: make([]int16, inputSampleRate*20/1000*channels)},
		channels:       channels,
		inFrequency:    inputSampleRate,
		outFrequency:   outputSampleRate,
		outBufferSize:  1024,
		onFullBuffer:   func(data []byte) {},
	}

	_ = enc.SetMaxBandwidth(FullBand)
	_ = enc.SetBitrate(192000)
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

func CallbackOnFullBuffer(fn func(_ []byte)) func(*Encoder) error {
	return func(e *Encoder) (err error) {
		e.onFullBuffer = fn
		return
	}
}

func (e *Encoder) BufferWrite(samples []int16) (written int) {
	n := len(samples)
	for k := n / len(e.buffer.Data); written < n || k >= 0; k-- {
		written += e.buffer.Write(samples[written:])
		if e.buffer.Full() {
			data, err := e.Encode(e.buffer.Data)
			if err != nil {
				log.Println("[!] Failed to encode", err)
				continue
			}
			e.onFullBuffer(data)
		}
	}
	return
}

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	if e.resampleBufSize > 0 {
		pcm = resampleFn(pcm, e.resampleBufSize)
	}
	data := make([]byte, e.outBufferSize)
	n, err := e.LibOpusEncoder.Encode(pcm, data)
	if err != nil {
		return []byte{}, err
	}
	return data[:n], nil
}

func (e *Encoder) GetInfo() string {
	bitrate, _ := e.LibOpusEncoder.Bitrate()
	complexity, _ := e.LibOpusEncoder.Complexity()
	dtx, _ := e.LibOpusEncoder.DTX()
	fec, _ := e.LibOpusEncoder.FEC()
	maxBandwidth, _ := e.LibOpusEncoder.MaxBandwidth()
	lossPercent, _ := e.LibOpusEncoder.PacketLossPerc()
	sampleRate, _ := e.LibOpusEncoder.SampleRate()
	return fmt.Sprintf(
		"%v, Bitrate: %v bps, Complexity: %v, DTX: %v, FEC: %v, Max bandwidth: *%v, Loss%%: %v, Rate: %v Hz",
		CodecVersion(), bitrate, complexity, dtx, fec, maxBandwidth, lossPercent, sampleRate,
	)
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
