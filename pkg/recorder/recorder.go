package recorder

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
)

type Recording struct {
	sync.Mutex

	active bool
	User   string

	audioStream wavStream
	videoStream ffmpegStream
}

// Stream represent an output stream of the recording.
type Stream interface {
	Start()
	Stop() error
}

type (
	Audio struct {
		Samples *[]int16
	}
	Video struct {
		Image    image.Image
		Duration time.Duration
	}
)

// NewRecording creates new recorder of the emulator.
//
// FFMPEG:
//
// Example of conversion:
//    ffmpeg -f concat -i "./recording/psxtest/input.txt" \
//   		 -ac 2 -channel_layout stereo -i "./recording/psxtest/audio.wav" \
//  		 -b:a 128K -r 60 -crf 30 -preset faster -pix_fmt yuv420p out.mp4
//
func NewRecording(game string, frequency int, conf shared.Recording) Recording {
	// todo flush all files on record stop not on room close
	date := time.Now().Format("20060102")
	savePath, err := filepath.Abs(conf.Folder)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		if err = os.Mkdir(savePath, os.ModeDir); err != nil {
			log.Fatal(err)
		}
	}
	saveFolder := fmt.Sprintf("%v_%v", date, game)
	path := filepath.Join(savePath, saveFolder)

	log.Printf("[recording] path is [%v]", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.Mkdir(path, os.ModeDir); err != nil {
			log.Fatal(err)
		}
	}

	audio, err := NewWavAudioStream(path, frequency)
	if err != nil {
		log.Fatal(err)
	}
	video, err := NewFfmpegStream(path, game, frequency, conf.CompressLevel)
	if err != nil {
		log.Fatal(err)
	}

	return Recording{
		audioStream: *audio,
		videoStream: *video,
	}
}

func (r *Recording) Start() {
	r.Lock()
	defer r.Unlock()
	r.audioStream.Start()
	r.videoStream.Start()
}

func (r *Recording) Stop() (err error) {
	r.Lock()
	defer r.Unlock()
	r.active = false
	err = r.audioStream.Stop()
	err = r.videoStream.Stop()
	return
}

func (r *Recording) Set(active bool, user string) {
	r.Lock()
	r.active = active
	r.User = user
	r.Unlock()
}

func (r *Recording) IsActive() bool {
	r.Lock()
	defer r.Unlock()
	return r.active
}

func (r *Recording) WriteVideo(frame Video) { r.videoStream.buf <- frame }

func (r *Recording) WriteAudio(audio Audio) { r.audioStream.buf <- audio }
