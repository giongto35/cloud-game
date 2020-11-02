package audio

import (
	"errors"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"gopkg.in/hraban/opus.v2"
)

type OpusEncoder struct {
	encoder *opus.Encoder
	conf    *config.Opus
	buffer  []byte
}

func NewOpusEncoder(config config.Opus) (Encoder, error) {
	enc, err := opus.NewEncoder(config.Hz, config.Ch, opus.AppAudio)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, errors.New("opus encoder initialization has failed")
	}

	enc.SetMaxBandwidth(opus.Fullband)
	enc.SetBitrateToAuto()
	enc.SetComplexity(10)

	return &OpusEncoder{
		encoder: enc,
		conf:    &config,
		buffer:  make([]byte, 1024*2),
	}, nil
}

func (e *OpusEncoder) encode(pcm []int16) []byte {
	n, err := e.encoder.Encode(pcm, e.buffer)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil
	}
	e.buffer = e.buffer[:n]
	return e.buffer
}

func (e *OpusEncoder) getSampleRate() int {
	return e.conf.Hz
}

func (e *OpusEncoder) getChannelCount() int {
	return e.conf.Ch
}

func (e *OpusEncoder) getFrameSize() float64 {
	return e.conf.FrameMs
}
