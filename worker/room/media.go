package room

import (
	"fmt"
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/util"
	"gopkg.in/hraban/opus.v2"
)

func resample(pcm []float32, targetSize int, srcSampleRate int, dstSampleRate int) []float32 {
	newPCM := make([]float32, targetSize)
	for i := 0; i < len(pcm); i++ {
		newPCM[i*dstSampleRate/srcSampleRate] = pcm[i]
	}

	return newPCM
}

func (r *Room) startAudio() {
	log.Println("Enter fan audio")
	srcSampleRate := 32768
	dstSampleRate := 48000

	enc, err := opus.NewEncoder(dstSampleRate, 2, opus.AppVoIP)

	enc.SetMaxBandwidth(opus.Fullband)
	enc.SetBitrateToAuto()
	enc.SetComplexity(10)

	dstBufferSize := 240
	srcBufferSize := dstBufferSize * srcSampleRate / dstSampleRate
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
				// Client stopped
				//if !webRTC.IsClosed() {
				//continue
				//}

				// encode frame
				// fanout audioChannel
				if webRTC.IsConnected() {
					// NOTE: can block here
					webRTC.AudioChannel <- data
				}
			}

			idx = 0
		}
	}
}

//func (r *Room) startAudio() {
//log.Println("Enter fan audio")

//enc, err := opus.NewEncoder(emulator.SampleRate, emulator.Channels, opus.AppAudio)

////maxBufferSize := emulator.TimeFrame * r.director.GetSampleRate() / 1000
//maxBufferSize := 40 * emulator.SampleRate / 1000
//pcm := make([]float32, maxBufferSize) // 640 * 1000 / 16000 == 40 ms
////timeFrame := int(40 * 32000 / 1000)
//idx := 0

//if err != nil {
//log.Println("[!] Cannot create audio encoder", err)
//return
//}

//var count byte = 0

//// fanout Audio
//fmt.Println("listening audiochanel", r.IsRunning)
//for sample := range r.audioChannel {
//if !r.IsRunning {
//log.Println("Room ", r.ID, " audio channel closed")
//return
//}

//// TODO: Use worker pool for encoding
//pcm[idx] = sample
//idx++
//if idx == len(pcm) {
//data := make([]byte, maxBufferSize)

//n, err := enc.EncodeFloat32(pcm, data)

//if err != nil {
//log.Println("[!] Failed to decode", err)

//idx = 0
//count = (count + 1) & 0xff
//continue
//}
//data = data[:n]
//data = append(data, count)

//// TODO: r.rtcSessions is rarely updated. Lock will hold down perf
////r.sessionsLock.Lock()
//for _, webRTC := range r.rtcSessions {
//// Client stopped
////if !webRTC.IsClosed() {
////continue
////}

//// encode frame
//// fanout audioChannel
//if webRTC.IsConnected() {
//// NOTE: can block here
//webRTC.AudioChannel <- data
//}
////isRoomRunning = true
//}
////r.sessionsLock.Unlock()

//idx = 0
//count = (count + 1) & 0xff
//}
//}
//}

func (r *Room) startVideo() {
	size := int(float32(config.Width*config.Height) * 1.5)
	yuv := make([]byte, size, size)
	// fanout Screen
	for image := range r.imageChannel {
		if !r.IsRunning {
			log.Println("Room ", r.ID, " video channel closed")
			return
		}

		// TODO: Use worker pool for encoding
		util.RgbaToYuvInplace(image, yuv)
		// TODO: r.rtcSessions is rarely updated. Lock will hold down perf
		//r.sessionsLock.Lock()
		for _, webRTC := range r.rtcSessions {
			// Client stopped
			//if webRTC.IsClosed() {
			//continue
			//}

			// encode frame
			// fanout imageChannel
			if webRTC.IsConnected() {
				// NOTE: can block here
				webRTC.ImageChannel <- yuv
			}
		}
		//r.sessionsLock.Unlock()
	}
}
