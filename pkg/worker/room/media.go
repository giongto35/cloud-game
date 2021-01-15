package room

import (
	"fmt"
	"log"

	encoderConfig "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/opus"
	vpxencoder "github.com/giongto35/cloud-game/v2/pkg/encoder/vpx-encoder"
	"github.com/giongto35/cloud-game/v2/pkg/util"
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
	log.Println("Enter fan audio")

	enc, err := opus.NewEncoder(audio.Frequency, audio.Channels, audio.GetFrameDuration())
	if err != nil {
		log.Println("[!] Cannot create audio encoder", err)
	}
	if audio.Frequency != sampleRate {
		enc.SetResample(sampleRate)
	}

	pcm := enc.GetBuffer()
	idx := 0

	// fanout Audio
	for sample := range r.audioChannel {
		for i := 0; i < len(sample); {
			rem := util.MinInt(len(sample)-i, len(pcm)-idx)
			copy(pcm[idx:idx+rem], sample[i:i+rem])
			i += rem
			idx += rem

			if idx == len(pcm) {
				data, err := enc.Encode(pcm)
				if err != nil {
					log.Println("[!] Failed to encode", err)
					idx = 0
					continue
				}

				for _, webRTC := range r.rtcSessions {
					if webRTC.IsConnected() {
						// NOTE: can block here
						webRTC.AudioChannel <- data
					}
				}

				idx = 0
			}
		}

	}
	log.Println("Room ", r.ID, " audio channel closed")
}

// startVideo processes imageChannel images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) startVideo(width, height int, videoCodec encoder.VideoCodec) {
	var enc encoder.Encoder
	var err error

	log.Println("Video Encoder: ", videoCodec)
	if videoCodec == encoder.H264 {
		enc, err = h264encoder.NewH264Encoder(width, height, 1)
	} else {
		enc, err = vpxencoder.NewVpxEncoder(width, height, 20, 1200, 5)
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
