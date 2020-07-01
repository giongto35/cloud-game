package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/encoder"
	"github.com/giongto35/cloud-game/pkg/encoder/h264encoder"
	vpxencoder "github.com/giongto35/cloud-game/pkg/encoder/vpx-encoder"
	"github.com/giongto35/cloud-game/pkg/util"
	"github.com/giongto35/cloud-game/pkg/webrtc"
	"gopkg.in/hraban/opus.v2"
)

func resample(pcm []int16, targetSize int, srcSampleRate int, dstSampleRate int) []int16 {
	newPCML := make([]int16, targetSize/2)
	newPCMR := make([]int16, targetSize/2)
	newPCM := make([]int16, targetSize)
	for i := 0; i+1 < len(pcm); i += 2 {
		newPCML[(i/2)*dstSampleRate/srcSampleRate] = pcm[i]
		newPCMR[(i/2)*dstSampleRate/srcSampleRate] = pcm[i+1]
	}
	for i := 1; i < len(newPCML); i++ {
		if newPCML[i] == 0 {
			newPCML[i] = newPCML[i-1]
		}
	}
	for i := 1; i < len(newPCMR); i++ {
		if newPCMR[i] == 0 {
			newPCMR[i] = newPCMR[i-1]
		}
	}
	for i := 0; i+1 < targetSize; i += 2 {
		newPCM[i] = newPCML[i/2]
		newPCM[i+1] = newPCMR[i/2]
	}

	return newPCM
}

func min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

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
	log.Println("Enter fan audio")
	srcSampleRate := sampleRate

	enc, err := opus.NewEncoder(config.AUDIO_RATE, 2, opus.AppAudio)
	if err != nil {
		log.Println("[!] Cannot create audio encoder", err)
	}

	enc.SetMaxBandwidth(opus.Fullband)
	enc.SetBitrateToAuto()
	enc.SetComplexity(10)

	dstBufferSize := config.AUDIO_FRAME
	srcBufferSize := dstBufferSize * srcSampleRate / config.AUDIO_RATE
	pcm := make([]int16, srcBufferSize) // 640 * 1000 / 16000 == 40 ms
	idx := 0

	// fanout Audio
	for sample := range r.audioChannel {
		for i := 0; i < len(sample); {
			rem := util.MinInt(len(sample)-i, len(pcm)-idx)
			copy(pcm[idx:idx+rem], sample[i:i+rem])
			i += rem
			idx += rem

			if idx == len(pcm) {
				data := make([]byte, 1024*2)
				dstpcm := resample(pcm, dstBufferSize, srcSampleRate, config.AUDIO_RATE)
				n, err := enc.Encode(dstpcm, data)

				if err != nil {
					log.Println("[!] Failed to decode", err)

					idx = 0
					continue
				}
				data = data[:n]

				// TODO: r.rtcSessions is rarely updated. Lock will hold down perf
				//r.sessionsLock.Lock()
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
					webRTC.ImageChannel <- webrtc.WebFrame{ Data: data.Data, Timestamp: data.Timestamp }
				}
			}
		}
	}()

	for image := range r.imageChannel {
		if len(einput) < cap(einput) {
			einput <- encoder.InFrame{ Image: image.Image, Timestamp: image.Timestamp }
		}
	}
	log.Println("Room ", r.ID, " video channel closed")
}
