package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/codec"
	encoderConfig "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx"
	"github.com/giongto35/cloud-game/v2/pkg/media"
	"github.com/giongto35/cloud-game/v2/pkg/recorder"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

//func (r *Room) startVoice() {
//	// broadcast voice
//	go func() {
//		for sample := range r.voiceInChannel {
//			r.voiceOutChannel <- sample
//		}
//	}()
//
//	// fanout voice
//	go func() {
//		for sample := range r.voiceOutChannel {
//			for _, webRTC := range r.rtcSessions {
//				if webRTC.IsConnected() {
//					// NOTE: can block here
//					webRTC.VoiceOutChannel <- sample
//				}
//			}
//		}
//		for _, webRTC := range r.rtcSessions {
//			close(webRTC.VoiceOutChannel)
//		}
//	}()
//}

func (r *Room) isRecording() bool { return r.rec != nil && r.rec.Enabled() }

func (r *Room) startAudio(sampleRate int, audio encoderConfig.Audio) {
	buf := media.NewBuffer(audio.GetFrameSizeFor(sampleRate))
	resample, resampleSize := sampleRate != audio.Frequency, 0
	if resample {
		resampleSize = audio.GetFrameSize()
	}
	enc, err := opus.NewEncoder(audio.Frequency, audio.Channels)
	if err != nil {
		log.Fatalf("error: cannot create audio encoder, %v", err)
	}
	log.Printf("OPUS: %v", enc.GetInfo())

	for samples := range r.audioChannel {
		if r.isRecording() {
			r.rec.WriteAudio(recorder.Audio{Samples: &samples})
		}
		buf.Write(samples, func(s media.Samples) {
			if resample {
				s = media.ResampleStretch(s, resampleSize)
			}
			dat, err := enc.Encode(s)
			if err == nil {
				r.broadcastAudio(dat)
			}
		})
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
	if video.Codec == string(codec.H264) {
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
		fmt.Println("error create new encoder", err)
		return
	}

	r.vPipe = encoder.NewVideoPipe(enc, width, height)
	einput, eoutput := r.vPipe.Input, r.vPipe.Output

	go r.vPipe.Start()
	defer r.vPipe.Stop()

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
				webRTC.ImageChannel <- webrtc.WebFrame{Data: data.Data, Duration: data.Duration}
			}
		}
	}()

	for frame := range r.imageChannel {
		if len(einput) < cap(einput) {
			if r.isRecording() {
				go r.rec.WriteVideo(recorder.Video{Image: frame.Data, Duration: frame.Duration})
			}
			einput <- encoder.InFrame{Image: frame.Data, Duration: frame.Duration}
		}
	}
	log.Println("Room ", r.ID, " video channel closed")
}
