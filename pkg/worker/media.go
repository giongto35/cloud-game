package worker

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/worker/media"
	"github.com/giongto35/cloud-game/v2/pkg/worker/recorder"
)

var encoder_ *opus.Encoder

const (
	audioChannels  = 2
	audioCodec     = "opus"
	audioFrequency = 48000
)

// GetFrameSizeFor calculates audio frame size, i.e. 48k*frame/1000*2
func GetFrameSizeFor(hz int, frame int) int { return hz << 1 * frame / 1000 }

func (r *Room) startAudio(frequency int, onAudio func([]byte, error), conf conf.Audio) {
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

	for {
		select {
		case <-r.done:
			return
		case samples := <-r.emulator.GetAudio():
			if r.IsRecording() {
				r.rec.WriteAudio(recorder.Audio{Samples: &samples.Data, Duration: samples.Duration})
			}
			buf.Write(samples.Data, func(s media.Samples) {
				if resample {
					s = media.ResampleStretch(s, frameLen)
				}
				onAudio(enc.Encode(s))
			})
		}
	}
}

// startVideo processes videoFrames images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) startVideo(width, height int, onFrame func(encoder.OutFrame), conf conf.Video) {
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

	// a/v processing pipe
	r.vPipe = encoder.NewVideoPipe(enc, width, height, r.log)
	go r.vPipe.Start()
	defer r.vPipe.Stop()

	for {
		select {
		case <-r.done:
			return
		case frame := <-r.emulator.GetVideo():
			if r.IsRecording() {
				r.rec.WriteVideo(recorder.Video{Image: frame.Data, Duration: frame.Duration})
			}
			select {
			case r.vPipe.Input <- encoder.InFrame{Image: frame.Data, Duration: frame.Duration}:
			default:
			}
		case frame := <-r.vPipe.Output:
			onFrame(frame)
		}
	}
}
