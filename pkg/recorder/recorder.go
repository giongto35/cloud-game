package recorder

import (
	"image"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

type Recording struct {
	sync.Mutex

	enabled bool

	audio AudioStream
	video VideoStream

	dir     string
	saveDir string
	meta    Meta
	opts    Options
}

// naming regexp
var (
	reDate = regexp.MustCompile(`%date:(.*?)%`)
	reUser = regexp.MustCompile(`%user%`)
	reGame = regexp.MustCompile(`%game%`)
	reRand = regexp.MustCompile(`%rand:(\d+)%`)
)

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

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewRecording creates new recorder of the emulator.
//
// FFMPEG:
//
// Example of conversion:
//    ffmpeg -r 60 -f concat -i ./recording/psxtest/input.txt \
//   		-ac 2 -channel_layout stereo -i ./recording/psxtest/audio.wav \
//  		-b:a 192K -crf 23 -pix_fmt yuv420p out.mp4
func NewRecording(meta Meta, opts Options) *Recording {
	savePath, err := filepath.Abs(opts.Dir)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		if err = os.Mkdir(savePath, os.ModeDir); err != nil {
			log.Fatal(err)
		}
	}
	return &Recording{dir: savePath, meta: meta, opts: opts}
}

func (r *Recording) Start() {
	r.Lock()
	defer r.Unlock()
	r.enabled = true

	r.saveDir = parseName(r.opts.Name, r.opts.Game, r.meta.UserName)
	path := filepath.Join(r.dir, r.saveDir)

	log.Printf("[recording] path will be [%v]", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.Mkdir(path, os.ModeDir); err != nil {
			log.Fatal(err)
		}
	}

	audio, err := NewWavStream(path, r.opts)
	if err != nil {
		log.Fatal(err)
	}
	r.audio = audio
	video, err := NewFfmpegStream(path, r.opts)
	if err != nil {
		log.Fatal(err)
	}
	r.video = video

	go r.audio.Start()
	go r.video.Start()
}

func (r *Recording) Stop() error {
	var result *multierror.Error
	r.Lock()
	defer r.Unlock()
	r.enabled = false
	result = multierror.Append(result, r.audio.Stop())
	result = multierror.Append(result, r.video.Stop())
	if result.ErrorOrNil() == nil && r.opts.Zip && r.saveDir != "" {
		src := filepath.Join(r.dir, r.saveDir)
		dst := filepath.Join(src, "..", r.saveDir)
		go func() {
			if err := compress(src, dst); err != nil {
				log.Printf("error during result compress, %v", result)
				return
			}
			if err := os.RemoveAll(src); err != nil {
				log.Printf("error during result compress, %v", result)
			}
		}()
	}
	return result.ErrorOrNil()
}

func (r *Recording) Set(enable bool, user string) {
	r.Lock()
	if !r.enabled && enable {
		r.Unlock()
		r.Start()
		r.Lock()
	} else {
		if r.enabled && !enable {
			r.Unlock()
			if err := r.Stop(); err != nil {
				log.Printf("failed to stop recording, %v", err)
			}
			r.Lock()
		}
	}
	r.enabled = enable
	r.meta.UserName = user
	r.Unlock()
}

func (r *Recording) Enabled() bool {
	r.Lock()
	defer r.Unlock()
	return r.enabled
}

func (r *Recording) WriteVideo(frame Video) { r.video.Write(frame) }
func (r *Recording) WriteAudio(audio Audio) { r.audio.Write(audio) }

func parseName(name, game, user string) (out string) {
	if d := reDate.FindStringSubmatch(name); d != nil {
		out = reDate.ReplaceAllString(name, time.Now().Format(d[1]))
	} else {
		out = name
	}
	if rnd := reRand.FindStringSubmatch(out); rnd != nil {
		out = reRand.ReplaceAllString(out, random(rnd[1]))
	}
	out = reUser.ReplaceAllString(out, user)
	out = reGame.ReplaceAllString(out, game)
	return
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func random(num string) string {
	n, err := strconv.Atoi(num)
	if err != nil {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
