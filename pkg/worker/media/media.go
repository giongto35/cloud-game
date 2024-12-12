package media

import (
	"fmt"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/opus"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"sync"
	"time"
)

const (
	audioHz      = 48000
	sampleBufLen = 1024 * 4
)

type samples []int16

var (
	encoderOnce = sync.Once{}
	opusCoder   *opus.Encoder
	buf         = make([]int16, sampleBufLen)
)

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

type WebrtcMediaPipe struct {
	a        *opus.Encoder
	v        *encoder.Video
	onAudio  func([]byte, float32)
	audioBuf *buffer
	log      *logger.Logger

	mua sync.RWMutex
	muv sync.RWMutex

	aConf config.Audio
	vConf config.Video

	AudioSrcHz     int
	AudioFrames    []float32
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
	wmp.onAudio = func(bytes []byte, ms float32) {
		cb(bytes, int32(time.Duration(ms)*time.Millisecond))
	}
}
func (wmp *WebrtcMediaPipe) Destroy() {
	v := wmp.Video()
	if v != nil {
		v.Stop()
	}
}
func (wmp *WebrtcMediaPipe) PushAudio(audio []int16) {
	wmp.audioBuf.write(audio, wmp.encodeAudio)
}

func (wmp *WebrtcMediaPipe) Init() error {
	if err := wmp.initAudio(wmp.AudioSrcHz, wmp.AudioFrames); err != nil {
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

func (wmp *WebrtcMediaPipe) initAudio(srcHz int, frameSizes []float32) error {
	au, err := DefaultOpus()
	if err != nil {
		return fmt.Errorf("opus fail: %w", err)
	}
	wmp.log.Debug().Msgf("Opus: %v", au.GetInfo())
	wmp.SetAudio(au)
	buf, err := newBuffer(frameSizes, srcHz)
	if err != nil {
		return err
	}
	wmp.log.Debug().Msgf("Opus frames (ms): %v", frameSizes)
	dstHz, _ := au.SampleRate()
	if srcHz != dstHz {
		buf.resample(dstHz)
		wmp.log.Debug().Msgf("Resample %vHz -> %vHz", srcHz, dstHz)
	}
	wmp.audioBuf = buf
	return nil
}

func (wmp *WebrtcMediaPipe) encodeAudio(pcm samples, ms float32) {
	data, err := wmp.Audio().Encode(pcm)
	if err != nil {
		wmp.log.Error().Err(err).Msgf("opus encode fail")
		return
	}
	wmp.onAudio(data, ms)
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
