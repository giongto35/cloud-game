package recorder

import (
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	oss "github.com/giongto35/cloud-game/v3/pkg/os"
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
		Samples  []int16
		Duration time.Duration
	}
	Video struct {
		Frame    Frame
		Duration time.Duration
	}
	Frame struct {
		Data   []byte
		Stride int
		W, H   int
	}
)

// NewRecording creates new media recorder for the emulator.
func NewRecording(meta Meta, log *logger.Logger, opts Options) *Recording {
	savePath, err := filepath.Abs(opts.Dir)
	if err != nil {
		log.Error().Err(err).Send()
	}
	if err := oss.CheckCreateDir(savePath); err != nil {
		log.Error().Err(err).Send()
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

	if err := oss.CheckCreateDir(path); err != nil {
		r.log.Fatal().Err(err)
	}

	audio, err := newWavStream(path, r.opts)
	if err != nil {
		r.log.Fatal().Err(err)
		return
	}
	r.audio = audio
	video, err := newRawStream(path)
	if err != nil {
		r.log.Fatal().Err(err)
		return
	}
	r.video = video
}

func (r *Recording) Stop() (err error) {
	r.Lock()
	defer r.Unlock()
	r.enabled = false
	if r.audio != nil {
		err = r.audio.Close()
	}
	if r.video != nil {
		err = r.video.Close()
	}

	path := filepath.Join(r.dir, r.saveDir)
	// FFMPEG
	err = createFfmpegMuxFile(path, videoFile, r.vsync, r.opts)

	if err == nil && r.opts.Zip && r.saveDir != "" {
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
	return err
}

func (r *Recording) Set(enable bool, user string) {
	r.Lock()
	r.meta.UserName = user
	if !r.enabled && enable {
		r.Unlock()
		r.Start()
		r.log.Debug().Msgf("[REC] set: +, user: %v", user)
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
	r.Unlock()
}

func (r *Recording) SetFramerate(fps float64) { r.opts.Fps = fps }
func (r *Recording) SetAudioFrequency(fq int) { r.opts.Frequency = fq }

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
		b[i] = letterBytes[rand.Int64()%int64(len(letterBytes))]
	}
	return string(b)
}
