package media

import (
	"fmt"
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/opus"
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
	buf         = make([]int16, sampleBufLen)
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
// with round(x / 2) * 2 for the closest even number
func frame(hz int, frame float32) int {
	return int(math.Round(float64(hz)*float64(frame)/1000/2) * 2 * 2)
}

// stretch does a simple stretching of audio samples.
// something like: [1,2,3,4,5,6] -> [1,2,x,x,3,4,x,x,5,6,x,x] -> [1,2,1,2,3,4,3,4,5,6,5,6]
func (s samples) stretch(size int) []int16 {
	out := buf[:size]
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

	mua sync.RWMutex
	muv sync.RWMutex

	aConf config.Audio
	vConf config.Video

	AudioSrcHz     int
	AudioFrame     float32
	VideoW, VideoH int
	VideoScale     float64

	initialized bool

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
	v := wmp.Video()
	if v != nil {
		v.Stop()
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

	a := wmp.Audio()
	v := wmp.Video()

	if v == nil || a == nil {
		return fmt.Errorf("could intit the encoders, v=%v a=%v", v != nil, a != nil)
	}

	wmp.log.Debug().Msgf("%v", v.Info())
	wmp.initialized = true
	return nil
}

func (wmp *WebrtcMediaPipe) initAudio(srcHz int, frameSize float32) error {
	au, err := DefaultOpus()
	if err != nil {
		return fmt.Errorf("opus fail: %w", err)
	}
	wmp.log.Debug().Msgf("Opus: %v", au.GetInfo())
	wmp.SetAudio(au)
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
	data, err := wmp.Audio().Encode(pcm)
	if err != nil {
		wmp.log.Error().Err(err).Msgf("opus encode fail")
		return
	}
	wmp.onAudio(data)
}

func (wmp *WebrtcMediaPipe) initVideo(w, h int, scale float64, conf config.Video) (err error) {
	sw, sh := round(w, scale), round(h, scale)
	enc, err := encoder.NewVideoEncoder(w, h, sw, sh, scale, conf, wmp.log)
	if err != nil {
		return err
	}
	if enc == nil {
		return fmt.Errorf("broken video encoder init")
	}
	wmp.SetVideo(enc)
	wmp.log.Debug().Msgf("media scale: %vx%v -> %vx%v", w, h, sw, sh)
	return err
}

func round(x int, scale float64) int { return (int(float64(x)*scale) + 1) & ^1 }

func (wmp *WebrtcMediaPipe) ProcessVideo(v app.Video) []byte {
	return wmp.Video().Encode(encoder.InFrame(v.Frame))
}

func (wmp *WebrtcMediaPipe) Reinit() error {
	if !wmp.initialized {
		return nil
	}

	wmp.Video().Stop()
	if err := wmp.initVideo(wmp.VideoW, wmp.VideoH, wmp.VideoScale, wmp.vConf); err != nil {
		return err
	}
	// restore old
	wmp.SetPixFmt(wmp.oldPf)
	wmp.SetRot(wmp.oldRot)
	wmp.SetVideoFlip(wmp.oldFlip)
	return nil
}

func (wmp *WebrtcMediaPipe) IsInitialized() bool { return wmp.initialized }
func (wmp *WebrtcMediaPipe) SetPixFmt(f uint32)  { wmp.oldPf = f; wmp.v.SetPixFormat(f) }
func (wmp *WebrtcMediaPipe) SetVideoFlip(b bool) { wmp.oldFlip = b; wmp.v.SetFlip(b) }
func (wmp *WebrtcMediaPipe) SetRot(r uint)       { wmp.oldRot = r; wmp.v.SetRot(r) }

func (wmp *WebrtcMediaPipe) Video() *encoder.Video {
	wmp.muv.RLock()
	defer wmp.muv.RUnlock()
	return wmp.v
}

func (wmp *WebrtcMediaPipe) SetVideo(e *encoder.Video) {
	wmp.muv.Lock()
	wmp.v = e
	wmp.muv.Unlock()
}

func (wmp *WebrtcMediaPipe) Audio() *opus.Encoder {
	wmp.mua.RLock()
	defer wmp.mua.RUnlock()
	return wmp.a
}

func (wmp *WebrtcMediaPipe) SetAudio(e *opus.Encoder) {
	wmp.mua.Lock()
	wmp.a = e
	wmp.mua.Unlock()
}
