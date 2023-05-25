package worker

import (
	"sync"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/h264"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/opus"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/vpx"
	webrtc "github.com/pion/webrtc/v3/pkg/media"
)

const (
	dstHz        = 48000
	sampleBufLen = 1024 * 4
)

// buffer is a simple non-concurrent safe ring buffer for audio samples.
type (
	buffer struct {
		s       samples
		wi      int
		dst     int
		stretch bool
	}
	samples []int16
)

var (
	encoderOnce = sync.Once{}
	opusCoder   *opus.Encoder
	samplePool  sync.Pool
	audioPool   = sync.Pool{New: func() any { b := make([]int16, sampleBufLen); return &b }}
)

func newBuffer(srcLen int) buffer { return buffer{s: make(samples, srcLen)} }

// enableStretch adds a simple stretching of buffer to a desired size before
// the onFull callback call.
func (b *buffer) enableStretch(l int) { b.stretch = true; b.dst = l }

// write fills the buffer until it's full and then passes the gathered data into a callback.
//
// There are two cases to consider:
// 1. Underflow, when the length of the written data is less than the buffer's available space.
// 2. Overflow, when the length exceeds the current available buffer space.
//
// We overwrite any previous values in the buffer and move the internal write pointer
// by the length of the written data.
// In the first case, we won't call the callback, but it will be called every time
// when the internal buffer overflows until all samples are read.
func (b *buffer) write(s samples, onFull func(samples)) (r int) {
	for r < len(s) {
		w := copy(b.s[b.wi:], s[r:])
		r += w
		b.wi += w
		if b.wi == len(b.s) {
			b.wi = 0
			if b.stretch {
				onFull(b.s.stretch(b.dst))
			} else {
				onFull(b.s)
			}
		}
	}
	return
}

// frame calculates an audio stereo frame size, i.e. 48k*frame/1000*2
func frame(hz int, frame int) int { return hz * frame / 1000 * 2 }

// stretch does a simple stretching of audio samples.
// something like: [1,2,3,4,5,6] -> [1,2,x,x,3,4,x,x,5,6,x,x] -> [1,2,1,2,3,4,3,4,5,6,5,6]
func (s samples) stretch(size int) []int16 {
	out := (*audioPool.Get().(*[]int16))[:size]
	n := len(s)
	ratio := float32(size) / float32(n)
	sPtr := unsafe.Pointer(&s[0])
	for i, l, r := 0, 0, 0; i < n; i += 2 {
		l, r = r, int(float32((i+2)>>1)*ratio)<<1 // index in src * ratio -> approximated index in dst *2 due to int16
		for j := l; j < r; j += 2 {
			*(*int32)(unsafe.Pointer(&out[j])) = *(*int32)(sPtr) // out[j] = s[i]; out[j+1] = s[i+1]
		}
		sPtr = unsafe.Add(sPtr, uintptr(4))
	}
	return out
}

func (r *Room) initAudio(srcHz int, conf config.Audio) {
	encoderOnce.Do(func() {
		enc, err := opus.NewEncoder(dstHz)
		if err != nil {
			r.log.Fatal().Err(err).Msg("couldn't create audio encoder")
		}
		opusCoder = enc
	})
	if err := opusCoder.Reset(); err != nil {
		r.log.Error().Err(err).Msgf("opus state reset fail")
	}
	r.log.Debug().Msgf("Opus: %v", opusCoder.GetInfo())

	buf := newBuffer(frame(srcHz, conf.Frame))
	if srcHz != dstHz {
		buf.enableStretch(frame(dstHz, conf.Frame))
		r.log.Debug().Msgf("Resample %vHz -> %vHz", srcHz, dstHz)
	}
	frameDur := time.Duration(conf.Frame) * time.Millisecond

	r.emulator.SetAudio(func(raw *emulator.GameAudio) {
		buf.write(*raw.Data, func(pcm samples) {
			data, err := opusCoder.Encode(pcm)
			audioPool.Put((*[]int16)(&pcm))
			if err != nil {
				r.log.Error().Err(err).Msgf("opus encode fail")
				return
			}
			r.handleSample(data, frameDur, func(u *Session, s *webrtc.Sample) {
				if err := u.SendAudio(s); err != nil {
					r.log.Error().Err(err).Send()
				}
			})
		})
	})
}

// initVideo processes videoFrames images with an encoder (codec) then pushes the result to WebRTC.
func (r *Room) initVideo(width, height int, conf config.Video) {
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
