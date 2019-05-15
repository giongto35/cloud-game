package room

import (
	"log"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/emulator"
	"github.com/giongto35/cloud-game/util"
	"gopkg.in/hraban/opus.v2"
)

func (r *Room) startAudio() {
	log.Println("Enter fan audio")

	enc, err := opus.NewEncoder(emulator.SampleRate, emulator.Channels, opus.AppAudio)

	maxBufferSize := emulator.TimeFrame * emulator.SampleRate / 1000
	pcm := make([]float32, maxBufferSize) // 640 * 1000 / 16000 == 40 ms
	idx := 0

	if err != nil {
		log.Println("[!] Cannot create audio encoder")
		return
	}

	var count byte = 0

	// fanout Audio
	for {
		sample, ok := <-r.audioChannel
		if !ok {
			// Just for guarding
			log.Println("Warn: Room ", r.ID, " audio channel closed unexpectedly")
			return
		}
		if r.Done {
			log.Println("Room ", r.ID, " audio channel closed")
			return
		}

		// TODO: Use worker pool for encoding
		pcm[idx] = sample
		idx++
		if idx == len(pcm) {
			data := make([]byte, 640)

			n, err := enc.EncodeFloat32(pcm, data)

			if err != nil {
				log.Println("[!] Failed to decode")
				continue
			}
			data = data[:n]
			data = append(data, count)

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
				//isRoomRunning = true
			}
			//r.sessionsLock.Unlock()

			idx = 0
			count = (count + 1) & 0xff
		}
	}
}

func (r *Room) startVideo() {
	size := int(float32(config.Width*config.Height) * 1.5)
	yuv := make([]byte, size, size)
	// fanout Screen
	for {
		image, ok := <-r.imageChannel
		if !ok {
			// Just for guarding, should not reached
			log.Println("Warn: Room ", r.ID, " video channel closed unexpectedly")
			return
		}
		if r.Done {
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
