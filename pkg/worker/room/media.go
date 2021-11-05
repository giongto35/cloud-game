package room

import (
	encoderConfig "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

func (r *Room) startAudio(sampleRate int, audio encoderConfig.Audio) {
	sound, err := opus.NewEncoder(
		sampleRate,
		audio.Frequency,
		audio.Channels,
		opus.SampleBuffer(audio.Frame, sampleRate != audio.Frequency),
		// we use callback on full buffer in order to
		// send data to all the clients ASAP
		opus.CallbackOnFullBuffer(r.broadcastAudio),
	)
	if err != nil {
		r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
	}
	r.log.Debug().Msgf("OPUS: %v", sound.GetInfo())
	for samples := range r.audioChannel {
		sound.BufferWrite(samples)
	}
	r.log.Info().Msg("Audio channel has been closed")
}

func (r *Room) broadcastAudio(audio []byte) {
	for _, webRTC := range r.rtcSessions {
		if webRTC.IsConnected() {
			// NOTE: can block here
			webRTC.AudioChannel <- audio
		}
	}
}

// startVideo processes imageChannel images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) startVideo(width, height int, video encoderConfig.Video) {
	var enc encoder.Encoder
	var err error

	r.log.Info().Msgf("Video codec: %v", video.Codec)
	if video.Codec == string(encoder.H264) {
		r.log.Debug().Msgf("x264: build v%v", h264.LibVersion())
		enc, err = h264.NewEncoder(width, height, h264.WithOptions(h264.Options{
			Crf:      video.H264.Crf,
			Tune:     video.H264.Tune,
			Preset:   video.H264.Preset,
			Profile:  video.H264.Profile,
			LogLevel: int32(video.H264.LogLevel),
		}))
	} else {
		enc, err = vpx.NewEncoder(width, height, vpx.WithOptions(vpx.Options{
			Bitrate:     video.Vpx.Bitrate,
			KeyframeInt: video.Vpx.KeyframeInterval,
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

	go func() {
		defer func() {
			if rc := recover(); rc != nil {
				r.log.Error().Msgf("recovered video pipe from (%v)", rc)
			}
		}()

		// fanout Screen
		for data := range eoutput {
			// TODO: r.rtcSessions is rarely updated. Lock will hold down perf
			for _, webRTC := range r.rtcSessions {
				if !webRTC.IsConnected() {
					continue
				}
				// encode frame
				// fanout imageChannel
				// NOTE: can block here
				webRTC.ImageChannel <- webrtc.WebFrame{Data: data.Data, Timestamp: data.Timestamp}
			}
		}
	}()

	for image := range r.imageChannel {
		if len(einput) < cap(einput) {
			einput <- encoder.InFrame{Image: image.Image, Timestamp: image.Timestamp}
		}
	}
	r.log.Info().Msg("Video channel has been closed")
}
