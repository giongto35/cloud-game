package recorder

import (
	"image"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/hashicorp/go-multierror"
)

type Recording struct {
	sync.Mutex

	enabled bool

	audio audioStream
	video videoStream

	dir     string
	saveDir string
	meta    Meta
	opts    Options
	log     *logger.Logger

	vsync []time.Duration
}

// naming regexp
var (
	reDate = regexp.MustCompile(`%date:(.*?)%`)
	reUser = regexp.MustCompile(`%user%`)
	reGame = regexp.MustCompile(`%game%`)
	reRand = regexp.MustCompile(`%rand:(\d+)%`)
)

// stream represent an output stream of the recording.
type stream interface {
	io.Closer
}

type audioStream interface {
	stream
	Write(data Audio)
}
type videoStream interface {
	stream
	Write(data Video)
}

type (
	Audio struct {
		Samples  *[]int16
		Duration time.Duration
	}
	Video struct {
		Image    image.Image
		Duration time.Duration
	}
)

func init() { rand.Seed(time.Now().UnixNano()) }

// NewRecording creates new media recorder for the emulator.
func NewRecording(meta Meta, log *logger.Logger, opts Options) *Recording {
	savePath, err := filepath.Abs(opts.Dir)
	if err != nil {
		log.Error().Err(err).Send()
	}
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		if err = os.Mkdir(savePath, os.ModeDir); err != nil {
			log.Error().Err(err).Send()
		}
	}
	return &Recording{dir: savePath, meta: meta, opts: opts, log: log, vsync: []time.Duration{}}
}

func (r *Recording) Start() {
	r.Lock()
	defer r.Unlock()
	r.enabled = true

	r.saveDir = parseName(r.opts.Name, r.opts.Game, r.meta.UserName)
	path := filepath.Join(r.dir, r.saveDir)

	r.log.Info().Msgf("[recording] path will be [%v]", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.Mkdir(path, os.ModeDir); err != nil {
			r.log.Fatal().Err(err)
		}
	}

	audio, err := newWavStream(path, r.opts)
	if err != nil {
		r.log.Fatal().Err(err)
	}
	r.audio = audio
	video, err := newPngStream(path, r.opts)
	if err != nil {
		r.log.Fatal().Err(err)
	}
	r.video = video
}

func (r *Recording) Stop() error {
	var result *multierror.Error
	r.Lock()
	defer r.Unlock()
	r.enabled = false
	result = multierror.Append(result, r.audio.Close())
	result = multierror.Append(result, r.video.Close())

	path := filepath.Join(r.dir, r.saveDir)
	// FFMPEG
	result = multierror.Append(result, createFfmpegMuxFile(path, videoFile, r.vsync, r.opts))

	if result.ErrorOrNil() == nil && r.opts.Zip && r.saveDir != "" {
		src := filepath.Join(r.dir, r.saveDir)
		dst := filepath.Join(src, "..", r.saveDir)
		go func() {
			if err := compress(src, dst); err != nil {
				r.log.Error().Err(err).Msg("error during result compress")
				return
			}
			if err := os.RemoveAll(src); err != nil {
				r.log.Error().Err(err).Msg("error during result compress")
			}
		}()
	}
	r.vsync = []time.Duration{}
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
				r.log.Error().Err(err).Msg("failed to stop recording")
			}
			r.log.Debug().Msg("recording has stopped")
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

func (r *Recording) WriteAudio(audio Audio) {
	r.audio.Write(audio)
	r.Lock()
	r.vsync = append(r.vsync, audio.Duration)
	r.Unlock()
}

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
