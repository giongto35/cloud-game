package opus

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
)

type Encoder struct {
	*LibOpusEncoder

	buf []byte
}

func NewEncoder(outFq, channels int, options ...func(*Encoder) error) (*Encoder, error) {
	encoder, err := NewOpusEncoder(outFq, channels, AppRestrictedLowDelay)
	if err != nil {
		return nil, err
	}
	enc := &Encoder{LibOpusEncoder: encoder, buf: make([]byte, 1000)}
	var result *multierror.Error
	result = multierror.Append(result,
		enc.SetMaxBandwidth(FullBand),
		enc.SetBitrate(192000),
		enc.SetComplexity(10),
	)
	for _, option := range options {
		result = multierror.Append(option(enc))
	}
	return enc, result.ErrorOrNil()
}

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	n, err := e.LibOpusEncoder.Encode(pcm, e.buf)
	// n = 1 is DTX
	if err != nil || n == 1 {
		return []byte{}, err
	}
	return e.buf[:n], nil
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
