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
	"github.com/hashicorp/go-multierror"
)

type Recording struct {
	sync.Mutex

	active bool
	User   string

	audio AudioStream
	video VideoStream
}

// Stream represent an output stream of the recording.
type Stream interface {
	Start()
	Stop() error
}

type AudioStream interface {
	Stream
	Write(data Audio)
}
type VideoStream interface {
	Stream
	Write(data Video)
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

// ffmpeg -r 60 -f concat -i "./recording/20211126_Sushi The Cat/input.txt" -ac 2 -channel_layout stereo -i "./recording
// 20211126_Sushi The Cat/audio.wav" -b:a 128K -r 60 -crf 16 -preset faster -pix_fmt yuv420p -ar 44100 -shortest out.mp4
//
func NewRecording(game string, frequency int, conf shared.Recording) *Recording {
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

	audio, err := NewWavStream(path, frequency)
	if err != nil {
		log.Fatal(err)
	}
	video, err := NewFfmpegStream(path, game, frequency, conf.CompressLevel)
	if err != nil {
		log.Fatal(err)
	}

	return &Recording{audio: audio, video: video}
}

func (r *Recording) Start() {
	r.Lock()
	defer r.Unlock()
	r.active = true
	go r.audio.Start()
	go r.video.Start()
}

func (r *Recording) Stop() error {
	var result *multierror.Error
	r.Lock()
	defer r.Unlock()
	r.active = false
	result = multierror.Append(result, r.audio.Stop())
	result = multierror.Append(result, r.video.Stop())
	return result.ErrorOrNil()
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

func (r *Recording) WriteVideo(frame Video) { r.video.Write(frame) }
func (r *Recording) WriteAudio(audio Audio) { r.audio.Write(audio) }
