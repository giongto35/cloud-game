package worker

import (
	conf "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/media"
	"github.com/giongto35/cloud-game/v2/pkg/recorder"
)

var encoder_ *opus.Encoder

func (r *Room) startAudio(frequency int, onAudio func([]byte, error), conf conf.Audio) {
	buf := media.NewBuffer(conf.GetFrameSizeFor(frequency))
	resample, resampleSize := frequency != conf.Frequency, 0
	if resample {
		resampleSize = conf.GetFrameSize()
	}
	// a garbage cache
	if encoder_ == nil {
		enc, err := opus.NewEncoder(conf.Frequency, conf.Channels)
		if err != nil {
			r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
		}
		encoder_ = enc
	}
	enc := *encoder_
	r.log.Debug().Msgf("OPUS: %v", enc.GetInfo())

	for {
		select {
		case <-r.Done:
			r.log.Info().Msg("Audio channel has been closed")
			return
		case samples := <-r.emulator.GetAudio():
			if r.IsRecording() {
				r.rec.WriteAudio(recorder.Audio{Samples: &samples.Data, Duration: samples.Duration})
			}
			buf.Write(samples.Data, func(s media.Samples) {
				if resample {
					s = media.ResampleStretch(s, resampleSize)
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
		case <-r.Done:
			r.log.Info().Msg("Video channel has been closed")
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
