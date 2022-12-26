package opus

import "fmt"

type Encoder struct {
	*Opus

	buf []byte
}

func NewEncoder(outFq int, options ...func(*Encoder) error) (enc *Encoder, err error) {
	encoder, err := NewOpusEncoder(outFq, AppRestrictedLowDelay)
	if err != nil {
		return nil, err
	}
	enc = &Encoder{Opus: encoder, buf: make([]byte, 1024)}
	err = enc.SetMaxBandwidth(FullBand)
	err = enc.SetBitrate(96000)
	err = enc.SetComplexity(5)
	for _, option := range options {
		err = option(enc)
	}
	return enc, err
}

func (e *Encoder) Reset() error { return e.ResetState() }

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	n, err := e.Opus.Encode(pcm, e.buf)
	// n = 1 is DTX
	if err != nil || n == 1 {
		return []byte{}, err
	}
	return e.buf[:n], nil
}

func (e *Encoder) GetInfo() string {
	bitrate, _ := e.Opus.Bitrate()
	complexity, _ := e.Opus.Complexity()
	dtx, _ := e.Opus.DTX()
	fec, _ := e.Opus.FEC()
	maxBandwidth, _ := e.Opus.MaxBandwidth()
	lossPercent, _ := e.Opus.PacketLossPerc()
	sampleRate, _ := e.Opus.SampleRate()
	return fmt.Sprintf(
		"%v, Bitrate: %v bps, Complexity: %v, DTX: %v, FEC: %v, Max bandwidth: *%v, Loss%%: %v, Rate: %v Hz",
		CodecVersion(), bitrate, complexity, dtx, fec, maxBandwidth, lossPercent, sampleRate,
	)
}
