package libretro

import (
	"image"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/recorder"
)

type RecordingFrontend struct {
	Emulator
	rec *recorder.Recording
}

// !to fix opaque image save

type opaque struct{ image.RGBA }

func (o *opaque) Opaque() bool { return true }

func WithRecording(fe Emulator, rec bool, user string, game string, conf config.Recording, log *logger.Logger) *RecordingFrontend {
	rr := &RecordingFrontend{Emulator: fe, rec: recorder.NewRecording(
		recorder.Meta{UserName: user},
		log,
		recorder.Options{
			Dir:                   conf.Folder,
			Game:                  game,
			ImageCompressionLevel: conf.CompressLevel,
			Name:                  conf.Name,
			Zip:                   conf.Zip,
			Vsync:                 true,
		})}
	rr.ToggleRecording(rec, user)
	return rr
}

func (r *RecordingFrontend) SetAudioCb(fn func(app.Audio)) {
	r.Emulator.SetAudioCb(func(audio app.Audio) {
		if r.IsRecording() {
			pcm := audio.Data
			// example: 1600 = x / 1000 * 48000 * 2
			l := time.Duration(float64(len(pcm)) / float64(r.AudioSampleRate()<<1) * 1000000000)
			r.rec.WriteAudio(recorder.Audio{Samples: pcm, Duration: l})
		}
		fn(audio)
	})
}

func (r *RecordingFrontend) SetVideoCb(fn func(app.Video)) {
	r.Emulator.SetVideoCb(func(v app.Video) {
		if r.IsRecording() {
			r.rec.WriteVideo(recorder.Video{Image: &opaque{v.Frame}, Duration: time.Duration(v.Duration)})
		}
		fn(v)
	})
}

func (r *RecordingFrontend) LoadGame(path string) error {
	err := r.Emulator.LoadGame(path)
	if err != nil {
		return err
	}
	r.rec.SetFramerate(float64(r.Emulator.FPS()))
	r.rec.SetAudioFrequency(r.Emulator.AudioSampleRate())
	return nil
}

func (r *RecordingFrontend) ToggleRecording(active bool, user string) {
	if r.rec != nil {
		r.rec.Set(active, user)
	}
}

func (r *RecordingFrontend) IsRecording() bool { return r.rec != nil && r.rec.Enabled() }
func (r *RecordingFrontend) Close()            { r.Emulator.Close(); r.ToggleRecording(false, "") }
