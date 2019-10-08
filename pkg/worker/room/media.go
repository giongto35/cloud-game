package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/encoder"
	"github.com/giongto35/cloud-game/pkg/encoder/h264encoder"
	vpxencoder "github.com/giongto35/cloud-game/pkg/encoder/vpx-encoder"
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
	fmt.Println("src BufferSize", srcBufferSize)
	pcm := make([]int16, srcBufferSize) // 640 * 1000 / 16000 == 40 ms
	idx := 0

	// fanout Audio
	fmt.Println("listening audio channel", r.IsRunning)
	for sample := range r.audioChannel {
		if !r.IsRunning {
			log.Println("Room ", r.ID, " audio channel closed")
			return
		}

		// TODO: Use worker pool for encoding
		pcm[idx] = sample
		idx++
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

func (r *Room) startVideo(width, height int, videoEncoderType string) {
	var encoder encoder.Encoder
	var err error

	log.Println("Video Encoder: ", videoEncoderType)
	if videoEncoderType == config.CODEC_H264 {
		encoder, err = h264encoder.NewH264Encoder(width, height, 1)
	} else {
		encoder, err = vpxencoder.NewVpxEncoder(width, height, 20, 1200, 5)
	}

	defer func() {
		encoder.Stop()
	}()

	if err != nil {
		fmt.Println("error create new encoder", err)
		return
	}
	einput := encoder.GetInputChan()
	eoutput := encoder.GetOutputChan()

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
					webRTC.ImageChannel <- data
				}
			}
		}
	}()

	for image := range r.imageChannel {
		if !r.IsRunning {
			log.Println("Room ", r.ID, " video channel closed")
			return
		}
		if len(einput) < cap(einput) {
			einput <- image
		}
	}
}
