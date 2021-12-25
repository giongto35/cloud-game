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

func (r *Room) startAudio(frequency int, onAudio func([]byte, error), conf conf.Audio) {
	buf := media.NewBuffer(conf.GetFrameSizeFor(frequency))
	resample, resampleSize := frequency != conf.Frequency, 0
	if resample {
		resampleSize = conf.GetFrameSize()
	}
	enc, err := opus.NewEncoder(conf.Frequency, conf.Channels)
	if err != nil {
		r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
	}
	r.log.Debug().Msgf("OPUS: %v", enc.GetInfo())

	for {
		select {
		case <-r.Done:
			r.log.Info().Msg("Audio channel has been closed")
			return
		case samples := <-r.audioChannel:
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

// startVideo processes imageChannel images with an encoder (codec) then pushes the result to WebRTC.
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

	r.vPipe = encoder.NewVideoPipe(enc, width, height, r.log)
	einput, eoutput := r.vPipe.Input, r.vPipe.Output

	go r.vPipe.Start()
	defer r.vPipe.Stop()

	for {
		select {
		case <-r.Done:
			r.log.Info().Msg("Video channel has been closed")
			return
		case frame := <-r.imageChannel:
			if r.IsRecording() {
				r.rec.WriteVideo(recorder.Video{Image: frame.Data, Duration: frame.Duration})
			}
			if len(einput) < cap(einput) {
				einput <- encoder.InFrame{Image: frame.Data, Duration: frame.Duration}
			}
		case frame := <-eoutput:
			onFrame(frame)
		}
	}
}
