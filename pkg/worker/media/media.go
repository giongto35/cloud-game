package media

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/h264"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/vpx"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
)

const (
	audioHz      = 48000
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

func DefaultOpus() (*opus.Encoder, error) {
	var err error
	encoderOnce.Do(func() { opusCoder, err = opus.NewEncoder(audioHz) })
	if err != nil {
		return nil, err
	}
	if err = opusCoder.Reset(); err != nil {
		return nil, err
	}
	return opusCoder, nil
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

type WebrtcMediaPipe struct {
	a        *opus.Encoder
	v        *encoder.Video
	onAudio  func([]byte)
	audioBuf buffer
	log      *logger.Logger

	aConf config.Audio
	vConf config.Video

	AudioSrcHz     int
	AudioFrame     int
	VideoW, VideoH int
	VideoScale     float64

	// keep the old settings for reinit
	oldPf   uint32
	oldRot  uint
	oldFlip bool
}

func NewWebRtcMediaPipe(ac config.Audio, vc config.Video, log *logger.Logger) *WebrtcMediaPipe {
	return &WebrtcMediaPipe{log: log, aConf: ac, vConf: vc}
}

func (wmp *WebrtcMediaPipe) SetAudioCb(cb func([]byte, int32)) {
	fr := int32(time.Duration(wmp.AudioFrame) * time.Millisecond)
	wmp.onAudio = func(bytes []byte) { cb(bytes, fr) }
}
func (wmp *WebrtcMediaPipe) Destroy() {
	if wmp.v != nil {
		wmp.v.Stop()
	}
}
func (wmp *WebrtcMediaPipe) PushAudio(audio []int16) { wmp.audioBuf.write(audio, wmp.encodeAudio) }

func (wmp *WebrtcMediaPipe) Init() error {
	if err := wmp.initAudio(wmp.AudioSrcHz, wmp.AudioFrame); err != nil {
		return err
	}
	if err := wmp.initVideo(wmp.VideoW, wmp.VideoH, wmp.VideoScale, wmp.vConf); err != nil {
		return err
	}
	return nil
}

func (wmp *WebrtcMediaPipe) initAudio(srcHz int, frameSize int) error {
	au, err := DefaultOpus()
	if err != nil {
		return fmt.Errorf("opus fail: %w", err)
	}
	wmp.log.Debug().Msgf("Opus: %v", au.GetInfo())
	wmp.a = au
	buf := newBuffer(frame(srcHz, frameSize))
	dstHz, _ := au.SampleRate()
	if srcHz != dstHz {
		buf.enableStretch(frame(dstHz, frameSize))
		wmp.log.Debug().Msgf("Resample %vHz -> %vHz", srcHz, dstHz)
	}
	wmp.audioBuf = buf
	return nil
}

func (wmp *WebrtcMediaPipe) encodeAudio(pcm samples) {
	data, err := wmp.a.Encode(pcm)
	audioPool.Put((*[]int16)(&pcm))
	if err != nil {
		wmp.log.Error().Err(err).Msgf("opus encode fail")
		return
	}
	wmp.onAudio(data)
}

func (wmp *WebrtcMediaPipe) initVideo(w, h int, scale float64, conf config.Video) error {
	var enc encoder.Encoder
	var err error

	sw, sh := round(w, scale), round(h, scale)

	wmp.log.Debug().Msgf("Scale: %vx%v -> %vx%v", w, h, sw, sh)

	wmp.log.Info().Msgf("Video codec: %v", conf.Codec)
	if conf.Codec == string(encoder.H264) {
		wmp.log.Debug().Msgf("x264: build v%v", h264.LibVersion())
		opts := h264.Options(conf.H264)
		enc, err = h264.NewEncoder(sw, sh, &opts)
	} else {
		opts := vpx.Options(conf.Vpx)
		enc, err = vpx.NewEncoder(sw, sh, &opts)
	}
	if err != nil {
		return fmt.Errorf("couldn't create a video encoder: %w", err)
	}
	wmp.v = encoder.NewVideoEncoder(enc, w, h, scale, wmp.log)
	wmp.log.Debug().Msgf("%v", wmp.v.Info())
	return nil
}

func round(x int, scale float64) int { return (int(float64(x)*scale) + 1) & ^1 }

func (wmp *WebrtcMediaPipe) ProcessVideo(v app.Video) []byte {
	return wmp.v.Encode(encoder.InFrame(v.Frame))
}

func (wmp *WebrtcMediaPipe) Reinit() error {
	wmp.v.Stop()
	if err := wmp.initVideo(wmp.VideoW, wmp.VideoH, wmp.VideoScale, wmp.vConf); err != nil {
		return err
	}
	// restore old
	wmp.SetPixFmt(wmp.oldPf)
	wmp.SetRot(wmp.oldRot)
	wmp.SetVideoFlip(wmp.oldFlip)
	return nil
}

func (wmp *WebrtcMediaPipe) SetPixFmt(f uint32)  { wmp.oldPf = f; wmp.v.SetPixFormat(f) }
func (wmp *WebrtcMediaPipe) SetVideoFlip(b bool) { wmp.oldFlip = b; wmp.v.SetFlip(b) }
func (wmp *WebrtcMediaPipe) SetRot(r uint)       { wmp.oldRot = r; wmp.v.SetRot(r) }
