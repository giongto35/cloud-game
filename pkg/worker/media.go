package worker

import (
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/worker/media"
)

var encoder_ *opus.Encoder

const (
	audioChannels  = 2
	audioCodec     = "opus"
	audioFrequency = 48000
)

// GetFrameSizeFor calculates audio frame size, i.e. 48k*frame/1000*2
func GetFrameSizeFor(hz int, frame int) int { return hz * frame / 1000 * audioChannels }

func (r *Room) initAudio(frequency int, onOutFrame func([]byte, time.Duration), conf conf.Audio) {
	buf := media.NewBuffer(GetFrameSizeFor(frequency, conf.Frame))
	resample, frameLen := frequency != audioFrequency, 0
	if resample {
		frameLen = GetFrameSizeFor(audioFrequency, conf.Frame)
	}

	// a garbage cache
	if encoder_ == nil {
		enc, err := opus.NewEncoder(audioFrequency, audioChannels)
		if err != nil {
			r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
		}
		encoder_ = enc
	}
	enc := *encoder_
	r.log.Debug().Msgf("OPUS: %v", enc.GetInfo())

	dur := time.Duration(conf.Frame) * time.Millisecond

	r.emulator.SetAudio(func(samples *emulator.GameAudio) {
		buf.Write(samples.Data, func(s media.Samples) {
			if resample {
				s = media.ResampleStretch(s, frameLen)
			}
			f, err := enc.Encode(s)
			media.BufOutAudioPool.Put([]int16(s))
			if err == nil {
				onOutFrame(f, dur)
			}
		})
	})
}

// initVideo processes videoFrames images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) initVideo(width, height int, onOutFrame func([]byte, time.Duration), conf conf.Video) {
	var enc encoder.Encoder
	var err error

	r.log.Info().Msgf("Video codec: %v", conf.Codec)
	if conf.Codec == string(encoder.H264) {
		r.log.Debug().Msgf("x264: build v%v", h264.LibVersion())
		enc, err = h264.NewEncoder(width, height, h264.WithOptions(h264.Options{
			Crf:      conf.H264.Crf,
			Tune:     conf.H264.Tune,
			Preset:   conf.H264.Preset,
			Profile:  conf.H264.Profile,
			LogLevel: int32(conf.H264.LogLevel),
		}))
	} else {
		enc, err = vpx.NewEncoder(width, height, vpx.WithOptions(vpx.Options{
			Bitrate:     conf.Vpx.Bitrate,
			KeyframeInt: conf.Vpx.KeyframeInterval,
		}))
	}

	if err != nil {
		r.log.Error().Err(err).Msg("couldn't create a video encoder")
		return
	}

	r.vEncoder = encoder.NewVideoEncoder(enc, width, height, r.log)

	r.emulator.SetVideo(func(frame *emulator.GameFrame) {
		if fr := r.vEncoder.Encode(frame.Data); fr != nil {
			onOutFrame(fr, frame.Duration)
		}
	})
}
