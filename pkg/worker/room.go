package worker

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/nanoarch"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/recorder"
	"github.com/giongto35/cloud-game/v2/pkg/session"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog/log"
)

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	ID string

	videoFrames <-chan emulator.GameFrame
	audioFrames <-chan emulator.GameAudio

	// State of room
	IsRunning bool
	// Done channel is to fire exit event when room is closed
	Done chan struct{}
	// List of users in the room
	users Sessions
	// Director is emulator
	director emulator.CloudEmulator
	// Cloud storage to store room state online
	onlineStorage storage.CloudStorage

	onClose func(self *Room)

	rec *recorder.Recording

	vPipe *encoder.VideoPipe
	log   *logger.Logger
}

func NewRoom(id string, game games.GameMetadata, storage storage.CloudStorage, onClose func(*Room), rec bool, recUser string, conf worker.Config, log *logger.Logger) *Room {
	if id == "" {
		id = session.GenerateRoomID(game.Name)
	}
	log = log.Extend(log.With().Str("room", id[:5]))
	log.Info().Str("game", game.Name).Msg("")

	if onClose == nil {
		onClose = func(*Room) {}
	}

	room := &Room{
		ID:            id,
		users:         NewSessions(),
		IsRunning:     true,
		onlineStorage: storage,
		Done:          make(chan struct{}, 1),
		onClose:       onClose,
		log:           log,
	}

	// Check if room is on local storage, if not, pull from GCS to local storage
	go func(game games.GameMetadata, roomID string) {
		var store nanoarch.Storage = &nanoarch.StateStorage{
			Path:     conf.Emulator.Storage,
			MainSave: roomID,
		}
		if conf.Emulator.Libretro.SaveCompression {
			store = &nanoarch.ZipStorage{Storage: store}
		}

		// Check room is on local or fetch from server
		log.Info().Msg("Check if the room in the cloud")
		if err := room.saveOnlineRoomToLocal(roomID, store.GetSavePath()); err != nil {
			log.Warn().Err(err).Msg("The room is not in the cloud")
		}

		// If not then load room or create room from local.
		log.Info().Str("game", game.Name).Msg("The room is opened")

		// Spawn new emulator and plug-in all channels
		emuName := conf.Emulator.GetEmulator(game.Type, game.Path)
		libretroConfig := conf.Emulator.GetLibretroCoreConfig(emuName)

		log.Info().Msgf("Image processing threads = %v", conf.Emulator.Threads)

		room.director, room.videoFrames, room.audioFrames =
			nanoarch.NewFrontend(roomID, store, libretroConfig, conf.Emulator.Threads, log)

		gameMeta := room.director.LoadMeta(filepath.Join(game.Base, game.Path))

		// nwidth, nheight are the WebRTC output size
		var nwidth, nheight int
		emu, ar := conf.Emulator, conf.Emulator.AspectRatio

		if ar.Keep {
			baseAspectRatio := float64(gameMeta.BaseWidth) / float64(ar.Height)
			nwidth, nheight = resizeToAspect(baseAspectRatio, ar.Width, ar.Height)
			log.Info().Msgf("Viewport size will be changed from %dx%d (%f) -> %dx%d", ar.Width, ar.Height,
				baseAspectRatio, nwidth, nheight)
		} else {
			nwidth, nheight = gameMeta.BaseWidth, gameMeta.BaseHeight
			log.Info().Msgf("Viewport custom size is disabled, base size will be used instead %dx%d", nwidth, nheight)
		}

		if emu.Scale > 1 {
			nwidth, nheight = nwidth*emu.Scale, nheight*emu.Scale
			log.Info().Msgf("Viewport size has scaled to %dx%d", nwidth, nheight)
		}

		// set game frame size considering its orientation
		encoderW, encoderH := nwidth, nheight
		if gameMeta.Rotation.IsEven {
			encoderW, encoderH = nheight, nwidth
		}

		room.director.SetViewport(encoderW, encoderH)

		if conf.Recording.Enabled {
			room.rec = recorder.NewRecording(
				recorder.Meta{UserName: recUser},
				log,
				recorder.Options{
					Dir:                   conf.Recording.Folder,
					Fps:                   gameMeta.Fps,
					Frequency:             gameMeta.AudioSampleRate,
					Game:                  game.Name,
					ImageCompressionLevel: conf.Recording.CompressLevel,
					Name:                  conf.Recording.Name,
					Zip:                   conf.Recording.Zip,
					Vsync:                 true,
				})
			room.ToggleRecording(rec, recUser)
		}

		go room.startVideo(encoderW, encoderH, func(frame encoder.OutFrame) {
			sample := media.Sample{Data: frame.Data, Duration: frame.Duration}
			room.users.EachConnected(func(s *Session) { _ = s.SendVideo(sample) })
		}, conf.Encoder.Video)

		dur := time.Duration(conf.Encoder.Audio.Frame) * time.Millisecond
		go room.startAudio(gameMeta.AudioSampleRate, func(audio []byte, err error) {
			if err != nil {
				return
			}
			sample := media.Sample{Data: audio, Duration: dur}
			room.users.EachConnected(func(s *Session) { _ = s.SendAudio(sample) })
		}, conf.Encoder.Audio)

		if conf.Emulator.AutosaveSec > 0 {
			go room.enableAutosave(conf.Emulator.AutosaveSec)
		}

		room.director.Start()
	}(game, id)
	return room
}

func (r *Room) enableAutosave(periodSec int) {
	log.Info().Msgf("Autosave is enabled with the period of [%vs]", periodSec)
	ticker := time.NewTicker(time.Duration(periodSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !r.IsRunning {
				continue
			}
			if err := r.director.SaveGame(); err != nil {
				log.Error().Msgf("Autosave failed: %v", err)
			} else {
				log.Debug().Msgf("Autosave done")
			}
		case <-r.Done:
			return
		}
	}
}

func resizeToAspect(ratio float64, sw int, sh int) (dw int, dh int) {
	// ratio is always > 0
	dw = int(math.Round(float64(sh)*ratio/2) * 2)
	dh = sh
	if dw > sw {
		dw = sw
		dh = int(math.Round(float64(sw)/ratio/2) * 2)
	}
	return
}

func isGameOnLocal(path string) bool {
	file, err := os.Open(path)
	if err == nil {
		defer func() {
			_ = file.Close()
		}()
	}
	return !errors.Is(err, os.ErrNotExist)
}

func (r *Room) PollUserInput(session *Session) {
	r.log.Debug().Msg("Start session input poll")
	session.GetPeerConn().OnMessage = func(data []byte) { r.director.Input(session.GetPlayerIndex(), data) }
}

func (r *Room) AddUser(user *Session) {
	r.users.Add(user.id, user)
	user.SetRoom(r)
}

// RemoveUser removes a user from the room.
func (r *Room) RemoveUser(user *Session) {
	user.SetRoom(nil)
	r.users.Remove(user)
	uid := user.GetId()
	r.log.Debug().Str("user", uid).Msg("User has left the room")
}

func (r *Room) HasUser(u *Session) bool { return r != nil && r.users.Get(u.id) != nil }

func (r *Room) Close() {
	if !r.IsRunning {
		return
	}

	r.IsRunning = false
	r.log.Debug().Msg("Closing the room")

	// Save game before quit. Only save for game which was previous saved to avoid flooding database
	if r.isRoomExisted() {
		r.log.Debug().Msg("Save game before closing room")
		// use goroutine here because SaveGame attempt to acquire an emulator lock.
		// the lock is holding before coming to close, so it will cause deadlock if SaveGame is synchronous
		go func() {
			// Save before close, so save can have correct state (Not sure) may again cause deadlock
			if err := r.SaveGame(); err != nil {
				r.log.Error().Err(err).Msg("couldn't save the game during close")
			}
			r.director.Close()
		}()
	} else {
		r.director.Close()
	}
	close(r.Done)

	r.onClose(r)

	if r.rec != nil {
		r.rec.Set(false, "")
	}
}

func (r *Room) isRoomExisted() bool {
	// Check if room is in online storage
	_, err := r.onlineStorage.Load(r.ID)
	if err == nil {
		return true
	}
	return isGameOnLocal(r.director.GetHashPath())
}

// SaveGame writes save state on the disk as well as
// uploads it to a cloud storage.
func (r *Room) SaveGame() error {
	// TODO: Move to game view
	if err := r.director.SaveGame(); err != nil {
		return err
	}
	if err := r.onlineStorage.Save(r.ID, r.director.GetHashPath()); err != nil {
		return err
	}
	r.log.Debug().Msg("Cloud save is successful")
	return nil
}

// saveOnlineRoomToLocal save online room to local.
// !Supports only one file of main save state.
func (r *Room) saveOnlineRoomToLocal(roomID string, savePath string) error {
	data, err := r.onlineStorage.Load(roomID)
	if err != nil {
		return err
	}
	// Save the data fetched from a cloud provider to the local server
	if data != nil {
		if err := os.WriteFile(savePath, data, 0644); err != nil {
			return err
		}
		r.log.Debug().Msg("Successfully downloaded cloud save")
	}
	return nil
}

func (r *Room) LoadGame() error { return r.director.LoadGame() }

func (r *Room) ToggleMultitap() { r.director.ToggleMultitap() }

func (r *Room) IsRecording() bool { return r.rec != nil && r.rec.Enabled() }

func (r *Room) ToggleRecording(active bool, user string) {
	if r.rec == nil {
		return
	}
	r.log.Debug().Msgf("[REC] set: %v, %v", active, user)
	r.rec.Set(active, user)
}

func (r *Room) IsEmpty() bool { return r.users.IsEmpty() }

func (r *Room) HasRunningSessions() (has bool) {
	has = false
	r.users.EachConnected(func(s *Session) {
		has = true
		return
	})
	return
}
