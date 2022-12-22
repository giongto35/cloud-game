package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/recorder"
)

type RecordingRoom struct {
	*Room

	rec *recorder.Recording
}

func Init(room *Room, rec bool, recUser string, game string, conf worker.Config) *RecordingRoom {
	rr := &RecordingRoom{Room: room, rec: recorder.NewRecording(
		recorder.Meta{UserName: recUser},
		room.log,
		recorder.Options{
			Dir:                   conf.Recording.Folder,
			Fps:                   float64(room.emulator.GetFps()),
			Frequency:             int(room.emulator.GetSampleRate()),
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
	handler := r.Room.emulator.GetAudio()
	r.Room.emulator.SetAudio(func(samples *emulator.GameAudio) {
		if r.IsRecording() {
			r.rec.WriteAudio(recorder.Audio{Samples: &samples.Data, Duration: samples.Duration})
		}
		handler(samples)
	})
}

func (r *RecordingRoom) captureVideo() {
	handler := r.Room.emulator.GetVideo()
	r.Room.emulator.SetVideo(func(frame *emulator.GameFrame) {
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
	r.log.Debug().Msgf("[REC] set: %v, %v", active, user)
	r.rec.Set(active, user)
}

func (r *RecordingRoom) IsRecording() bool { return r.rec != nil && r.rec.Enabled() }

func (r *RecordingRoom) Close() {
	r.Room.Close()
	if r.rec != nil {
		r.rec.Set(false, "")
	}
}
