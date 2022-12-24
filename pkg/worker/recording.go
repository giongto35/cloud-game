package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/recorder"
)

type RecordingRoom struct {
	GamingRoom
	rec *recorder.Recording
}

func WithRecording(room GamingRoom, rec bool, recUser string, game string, conf worker.Config) *RecordingRoom {
	room.GetLog().Info().Msgf("RECORD: %v %v", rec, recUser)

	rr := &RecordingRoom{GamingRoom: room, rec: recorder.NewRecording(
		recorder.Meta{UserName: recUser},
		room.GetLog(),
		recorder.Options{
			Dir:                   conf.Recording.Folder,
			Fps:                   float64(room.GetEmulator().GetFps()),
			Frequency:             int(room.GetEmulator().GetSampleRate()),
			Game:                  game,
			ImageCompressionLevel: conf.Recording.CompressLevel,
			Name:                  conf.Recording.Name,
			Zip:                   conf.Recording.Zip,
			Vsync:                 true,
		})}
	rr.ToggleRecording(rec, recUser)
	rr.captureAudio()
	rr.captureVideo()
	return rr
}

func (r *RecordingRoom) captureAudio() {
	handler := r.GetEmulator().GetAudio()
	r.GetEmulator().SetAudio(func(samples *emulator.GameAudio) {
		if r.IsRecording() {
			r.rec.WriteAudio(recorder.Audio{Samples: &samples.Data, Duration: samples.Duration})
		}
		handler(samples)
	})
}

func (r *RecordingRoom) captureVideo() {
	handler := r.GetEmulator().GetVideo()
	r.GetEmulator().SetVideo(func(frame *emulator.GameFrame) {
		if r.IsRecording() {
			r.rec.WriteVideo(recorder.Video{Image: frame.Data, Duration: frame.Duration})
		}
		handler(frame)
	})
}

func (r *RecordingRoom) ToggleRecording(active bool, user string) {
	if r.rec == nil {
		return
	}
	r.GetLog().Debug().Msgf("[REC] set: %v, %v", active, user)
	r.rec.Set(active, user)
}

func (r *RecordingRoom) IsRecording() bool { return r.rec != nil && r.rec.Enabled() }

func (r *RecordingRoom) Close() {
	r.GamingRoom.Close()
	if r.rec != nil {
		r.rec.Set(false, "")
	}
}
