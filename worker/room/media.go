package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/encoder"
	"github.com/giongto35/cloud-game/h264encoder"
	"github.com/giongto35/cloud-game/vpx-encoder"
	"gopkg.in/hraban/opus.v2"
)

func resample(pcm []float32, targetSize int, srcSampleRate int, dstSampleRate int) []float32 {
	newPCML := make([]float32, targetSize/2)
	newPCMR := make([]float32, targetSize/2)
	newPCM := make([]float32, targetSize)
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
	//srcSampleRate := 32768
	srcSampleRate := sampleRate
	dstSampleRate := 48000

	enc, err := opus.NewEncoder(dstSampleRate, 2, opus.AppVoIP)

	enc.SetMaxBandwidth(opus.Fullband)
	enc.SetBitrateToAuto()
	enc.SetComplexity(10)

	dstBufferSize := 240
	srcBufferSize := dstBufferSize * srcSampleRate / dstSampleRate
	fmt.Println("src BufferSize", srcBufferSize)
	pcm := make([]float32, srcBufferSize) // 640 * 1000 / 16000 == 40 ms
	idx := 0

	if err != nil {
		log.Println("[!] Cannot create audio encoder", err)
		return
	}

	// fanout Audio
	fmt.Println("listening audiochanel", r.IsRunning)
	for sample := range r.audioChannel {
		if !r.IsRunning {
			log.Println("Room ", r.ID, " audio channel closed")
			return
		}

		// TODO: Use worker pool for encoding
		pcm[idx] = sample
		idx++
		if idx == len(pcm) {
			data := make([]byte, dstBufferSize)
			dstpcm := resample(pcm, dstBufferSize, srcSampleRate, dstSampleRate)
			n, err := enc.EncodeFloat32(dstpcm, data)

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

func (r *Room) startVideo(width, height int) {
	var encoder encoder.Encoder
	var err error

	if config.Codec == config.CODEC_H264 {
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
