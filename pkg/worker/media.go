package worker

import (
	"sync"
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/opus"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/vpx"
	webrtc "github.com/pion/webrtc/v3/pkg/media"
)

var (
	encoderOnce = sync.Once{}
	opusCoder   *opus.Encoder
	samplePool  sync.Pool
	audioPool   = sync.Pool{New: func() any { b := make([]int16, 3000); return &b }}
)

const (
	audioChannels  = 2
	audioCodec     = "opus"
	audioFrequency = 48000
)

// Buffer is a simple non-thread safe ring buffer for audio samples.
// It should be used for 16bit PCM (LE interleaved) data.
type (
	Buffer struct {
		s  Samples
		wi int
	}
	OnFull  func(s Samples)
	Samples []int16
)

func NewBuffer(numSamples int) Buffer { return Buffer{s: make(Samples, numSamples)} }

// Write fills the buffer with data calling a callback function when
// the internal buffer fills out.
//
// Consider two cases:
//
// 1. Underflow, when the length of written data is less than the buffer's available space.
// 2. Overflow, when the length exceeds the current available buffer space.
// In the both cases we overwrite any previous values in the buffer and move the internal
// write pointer on the length of written data.
// In the first case we won't call the callback, but it will be called every time
// when the internal buffer overflows until all samples are read.
func (b *Buffer) Write(s Samples, onFull OnFull) (r int) {
	for r < len(s) {
		w := copy(b.s[b.wi:], s[r:])
		r += w
		b.wi += w
		if b.wi == len(b.s) {
			b.wi = 0
			if onFull != nil {
				onFull(b.s)
			}
		}
	}
	return
}

// GetFrameSizeFor calculates audio frame size, i.e. 48k*frame/1000*2
func GetFrameSizeFor(hz int, frame int) int { return hz * frame / 1000 * audioChannels }

func (r *Room) initAudio(frequency int, conf conf.Audio) {
	buf := NewBuffer(GetFrameSizeFor(frequency, conf.Frame))
	resample, frameLen := frequency != audioFrequency, 0
	if resample {
		frameLen = GetFrameSizeFor(audioFrequency, conf.Frame)
	}

	encoderOnce.Do(func() {
		enc, err := opus.NewEncoder(audioFrequency)
		if err != nil {
			r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
		}
		opusCoder = enc
	})
	if err := opusCoder.Reset(); err != nil {
		r.log.Error().Err(err).Msgf("opus state reset fail")
	}
	r.log.Debug().Msgf("Opus: %v", opusCoder.GetInfo())

	dur := time.Duration(conf.Frame) * time.Millisecond

	fn := func(s Samples) {
		if resample {
			s = ResampleStretchNew(s, frameLen)
		}
		f, err := opusCoder.Encode(s)
		audioPool.Put((*[]int16)(&s))
		if err == nil {
			r.handleSample(f, dur, func(u *Session, s *webrtc.Sample) {
				if err := u.SendAudio(s); err != nil {
					r.log.Error().Err(err).Send()
				}
			})
		}
	}
	r.emulator.SetAudio(func(samples *emulator.GameAudio) { buf.Write(*samples.Data, fn) })
}

// initVideo processes videoFrames images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) initVideo(width, height int, conf conf.Video) {
	var enc encoder.Encoder
	var err error

	r.log.Info().Msgf("Video codec: %v", conf.Codec)
	if conf.Codec == string(encoder.H264) {
		r.log.Debug().Msgf("x264: build v%v", h264.LibVersion())
		enc, err = h264.NewEncoder(width, height, &h264.Options{
			Crf:      conf.H264.Crf,
			Tune:     conf.H264.Tune,
			Preset:   conf.H264.Preset,
			Profile:  conf.H264.Profile,
			LogLevel: int32(conf.H264.LogLevel),
		})
	} else {
		enc, err = vpx.NewEncoder(width, height, &vpx.Options{
			Bitrate:     conf.Vpx.Bitrate,
			KeyframeInt: conf.Vpx.KeyframeInterval,
		})
	}

	if err != nil {
		r.log.Error().Err(err).Msg("couldn't create a video encoder")
		return
	}

	r.vEncoder = encoder.NewVideoEncoder(enc, width, height, conf.Concurrency, r.log)

	r.emulator.SetVideo(func(frame *emulator.GameFrame) {
		if fr := r.vEncoder.Encode(frame.Data.RGBA); fr != nil {
			r.handleSample(fr, frame.Duration, func(u *Session, s *webrtc.Sample) {
				if err := u.SendVideo(s); err != nil {
					r.log.Error().Err(err).Send()
				}
			})
		}
	})
}

func (r *Room) handleSample(b []byte, d time.Duration, fn func(*Session, *webrtc.Sample)) {
	sample, _ := samplePool.Get().(*webrtc.Sample)
	if sample == nil {
		sample = new(webrtc.Sample)
	}
	sample.Data = b
	sample.Duration = d
	r.users.ForEach(func(u *Session) {
		if u.IsConnected() {
			fn(u, sample)
		}
	})
	samplePool.Put(sample)
}

// ResampleStretchNew does a simple stretching of audio samples.
// something like: [1,2,3,4,5,6] -> [1,2,x,x,3,4,x,x,5,6,x,x] -> [1,2,1,2,3,4,3,4,5,6,5,6]
func ResampleStretchNew(pcm []int16, size int) []int16 {
	out := (*audioPool.Get().(*[]int16))[:size]
	n := len(pcm)
	ratio := float32(size) / float32(n)
	for i, l, r := 0, 0, 0; i < n; i += 2 {
		l, r = r, int(float32((i+2)>>1)*ratio)<<1
		for j := l; j < r-1; j += 2 {
			out[j] = pcm[i]
			out[j+1] = pcm[i+1]
		}
	}
	return out
}
