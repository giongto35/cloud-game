package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/audio"
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/h264encoder"
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

func (r *Room) startAudio(sampleRate int) {
	conf := config.DefaultOpusCfg()
	processor := audio.NewAudioProcessor(audio.NewOpusEncoder(conf))

	log.Printf("Audio (out): %v %vHz -> %v %vHz %vCh %vms", "PCM", sampleRate, "Opus", conf.Hz, conf.Ch, conf.FrameMs)

	samples := audio.GetSampleCount(sampleRate, conf.Ch, conf.FrameMs)
	pcm := make([]int16, samples)
	idx := 0
	// audio fan-out
	for sample := range r.audioChannel {
		// some strange way to do it
		for i := 0; i < len(sample); {
			rem := util.MinInt(len(sample)-i, len(pcm)-idx)
			copy(pcm[idx:idx+rem], sample[i:i+rem])
			i += rem
			idx += rem

			if idx == len(pcm) {
				data := processor.Encode(pcm, sampleRate)
				if data == nil {
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

// startVideo listen from imageChannel and push to Encoder. The output of encoder will be pushed to webRTC
func (r *Room) startVideo(width, height int, videoEncoderType string) {
	var enc encoder.Encoder
	var err error

	log.Println("Video Encoder: ", videoEncoderType)
	if videoEncoderType == config.CODEC_H264 {
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
				// encode frame
				// fanout imageChannel
				if webRTC.IsConnected() {
					// NOTE: can block here
					webRTC.ImageChannel <- webrtc.WebFrame{Data: data.Data, Timestamp: data.Timestamp}
				}
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
