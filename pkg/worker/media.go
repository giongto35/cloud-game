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
	"github.com/giongto35/cloud-game/v2/pkg/worker/media"
	webrtc "github.com/pion/webrtc/v3/pkg/media"
)

var (
	encoderOnce = sync.Once{}
	opusCoder   *opus.Encoder
	samplePool  sync.Pool
)

const (
	audioChannels  = 2
	audioCodec     = "opus"
	audioFrequency = 48000
)

// GetFrameSizeFor calculates audio frame size, i.e. 48k*frame/1000*2
func GetFrameSizeFor(hz int, frame int) int { return hz * frame / 1000 * audioChannels }

func (r *Room) initAudio(frequency int, conf conf.Audio) {
	buf := media.NewBuffer(GetFrameSizeFor(frequency, conf.Frame))
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

	fn := func(s media.Samples) {
		if resample {
			s = media.ResampleStretch(s, frameLen)
		}
		f, err := opusCoder.Encode(s)
		media.BufOutAudioPool.Put((*[]int16)(&s))
		if err == nil {
			r.handleSample(f, dur, func(u *Session, s *webrtc.Sample) { _ = u.SendAudio(s) })
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
		if fr := r.vEncoder.Encode(frame.Data); fr != nil {
			r.handleSample(fr, frame.Duration, func(u *Session, s *webrtc.Sample) { _ = u.SendVideo(s) })
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
