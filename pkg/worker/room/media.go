package room

import (
	"fmt"
	"log"

	encoderConfig "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

func (r *Room) startVoice() {
	// broadcast voice
	go func() {
		for sample := range r.voiceInChannel {
			r.voiceOutChannel <- sample
		}
	}()

	// fanout voice
	go func() {
		for sample := range r.voiceOutChannel {
			for _, webRTC := range r.rtcSessions {
				if webRTC.IsConnected() {
					// NOTE: can block here
					webRTC.VoiceOutChannel <- sample
				}
			}
		}
		for _, webRTC := range r.rtcSessions {
			close(webRTC.VoiceOutChannel)
		}
	}()
}

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
		log.Fatalf("error: cannot create audio encoder, %v", err)
	}
	log.Printf("OPUS: %v", sound.GetInfo())

	for samples := range r.audioChannel {
		sound.BufferWrite(samples)
	}

	log.Println("Room ", r.ID, " audio channel closed")
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

	log.Println("Video codec:", video.Codec)
	if video.Codec == encoder.H264.String() {
		enc, err = h264.NewEncoder(width, height, h264.WithOptions(h264.Options{
			Crf:      video.H264.Crf,
			Tune:     video.H264.Tune,
			Preset:   video.H264.Preset,
			Profile:  video.H264.Profile,
			LogLevel: int32(video.H264.LogLevel),
		}))
	} else {
		enc, err = vpx.NewEncoder(width, height, 20, 1200, 5)
	}

	defer func() {
		enc.Stop()
	}()

	if err != nil {
		fmt.Println("error create new encoder", err)
		return
	}

	r.encoder = enc

	einput := enc.GetInputChan()
	eoutput := enc.GetOutputChan()

	// send screenshot
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered when sent to close Image Channel")
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
	log.Println("Room ", r.ID, " video channel closed")
}
