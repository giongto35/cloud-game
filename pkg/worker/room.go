package worker

import (
	"math"
	"path/filepath"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/com"
	conf "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/nanoarch"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/recorder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/storage"
	"github.com/pion/webrtc/v3/pkg/media"
)

// Room defines a gaming session.
// It manages all the user WebRTC connections.
type Room struct {
	id       string
	active   bool
	done     chan struct{}
	users    com.NetMap[*Session] // a list of users in the room
	emulator emulator.CloudEmulator
	storage  storage.CloudStorage // a cloud storage to store room state online
	onClose  func(self *Room)
	rec      *recorder.Recording
	vPipe    *encoder.VideoPipe
	log      *logger.Logger
}

func NewRoom(id string, game games.GameMetadata, storage storage.CloudStorage, onClose func(*Room),
	rec bool, recUser string, conf worker.Config, log *logger.Logger) *Room {
	if id == "" {
		id = games.GenerateRoomID(game.Name)
	}
	log = log.Extend(log.With().Str("room", id[:5]))
	log.Info().Str("game", game.Name).Send()
	room := &Room{
		id:      id,
		active:  true,
		users:   com.NewNetMap[*Session](),
		storage: storage,
		done:    make(chan struct{}),
		onClose: onClose,
		log:     log,
	}

	fe, err := nanoarch.NewFrontend(conf.Emulator, log)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	room.emulator = fe
	room.emulator.SetMainSaveName(id)
	emulatorGuess := conf.Emulator.GetEmulator(game.Type, game.Path)
	room.emulator.LoadMetadata(emulatorGuess)

	gamePath := filepath.Join(game.Base, game.Path)
	if err := room.emulator.LoadGame(gamePath); err != nil {
		log.Fatal().Msgf("couldn't load the game %v, %v", gamePath, err)
	}

	// calc output frame size and rotation
	fw, fh := room.emulator.GetFrameSize()
	w, h := room.whatsFrame(conf.Emulator, fw, fh)
	if room.emulator.Rotated() {
		w, h = h, w
	}
	room.emulator.SetViewport(w, h)

	if !room.storage.IsNoop() {
		if err := room.saveOnlineRoomToLocal(id, room.emulator.GetHashPath()); err != nil {
			log.Warn().Err(err).Msg("The room is not in the cloud")
		}
	}

	log.Info().Str("game", game.Name).Msg("The room is opened")

	if conf.Recording.Enabled {
		room.rec = recorder.NewRecording(
			recorder.Meta{UserName: recUser},
			log,
			recorder.Options{
				Dir:                   conf.Recording.Folder,
				Fps:                   float64(room.emulator.GetFps()),
				Frequency:             int(room.emulator.GetSampleRate()),
				Game:                  game.Name,
				ImageCompressionLevel: conf.Recording.CompressLevel,
				Name:                  conf.Recording.Name,
				Zip:                   conf.Recording.Zip,
				Vsync:                 true,
			})
		room.ToggleRecording(rec, recUser)
	}

	go room.startVideo(w, h, func(frame encoder.OutFrame) {
		sample := media.Sample{Data: frame.Data, Duration: frame.Duration}
		room.users.ForEach(func(u *Session) {
			if u.IsConnected() {
				_ = u.SendVideo(sample)
			}
		})
	}, conf.Encoder.Video)

	dur := time.Duration(audioFrame) * time.Millisecond
	go room.startAudio(int(room.emulator.GetSampleRate()), func(audio []byte, err error) {
		if err != nil {
			return
		}
		sample := media.Sample{Data: audio, Duration: dur}
		room.users.ForEach(func(u *Session) {
			if u.IsConnected() {
				_ = u.SendAudio(sample)
			}
		})
	})

	if conf.Emulator.AutosaveSec > 0 {
		go room.enableAutosave(conf.Emulator.AutosaveSec)
	}

	go room.emulator.Start()

	return room
}

func (r *Room) enableAutosave(periodSec int) {
	r.log.Info().Msgf("Autosave every [%vs]", periodSec)
	ticker := time.NewTicker(time.Duration(periodSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !r.active {
				continue
			}
			if err := r.emulator.SaveGameState(); err != nil {
				r.log.Error().Msgf("Autosave failed: %v", err)
			} else {
				r.log.Debug().Msgf("Autosave done")
			}
		case <-r.done:
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

func (r *Room) whatsFrame(conf conf.Emulator, w, h int) (ww int, hh int) {
	// nwidth, nheight are the WebRTC output size
	var nwidth, nheight int
	emu, ar := conf, conf.AspectRatio

	if ar.Keep {
		baseAspectRatio := float64(w) / float64(ar.Height)
		nwidth, nheight = resizeToAspect(baseAspectRatio, ar.Width, ar.Height)
		r.log.Info().Msgf("Viewport size will be changed from %dx%d (%f) -> %dx%d", ar.Width, ar.Height,
			baseAspectRatio, nwidth, nheight)
	} else {
		nwidth, nheight = w, h
		r.log.Info().Msgf("Viewport resolution: %dx%d", nwidth, nheight)
	}

	if emu.Scale > 1 {
		nwidth, nheight = nwidth*emu.Scale, nheight*emu.Scale
		r.log.Info().Msgf("Viewport size has scaled to %dx%d", nwidth, nheight)
	}

	// set game frame size considering its orientation
	ww, hh = nwidth, nheight
	return
}

func hasStateSavedLocally(path string) bool { return os.Exists(path) }

func (r *Room) PollUserInput(session *Session) {
	r.log.Debug().Msg("Start session input poll")
	session.GetPeerConn().OnMessage = func(data []byte) { r.emulator.Input(session.GetPlayerIndex(), data) }
}

func (r *Room) AddUser(user *Session) {
	r.users.Add(user)
	user.SetRoom(r)
	r.log.Debug().Str("user", string(user.Id())).Msg("User has joined the room")
}

func (r *Room) CleanupUser(user *Session) {
	user.SetRoom(nil)
	if r.HasUser(user) {
		r.users.Remove(user)
		r.log.Debug().Str("user", string(user.Id())).Msg("User has left the room")
	}
	if r.IsEmpty() {
		r.log.Debug().Msg("The room is empty")
		r.Close()
	}
}

func (r *Room) HasUser(u *Session) bool { return r != nil && r.users.Has(u.id) }

func (r *Room) Close() {
	if !r.active {
		return
	}

	r.active = false
	r.log.Debug().Msg("Closing the room")

	// Save game before quit. Only save for game which was previous saved to avoid flooding database
	if r.isRoomExisted() {
		r.log.Debug().Msg("Save game before closing room")
		// use goroutine here because SaveGameState attempt to acquire an emulator lock.
		// the lock is holding before coming to close, so it will cause deadlock if SaveGameState is synchronous
		go func() {
			// Save before close, so save can have correct state (Not sure) may again cause deadlock
			if err := r.SaveGame(); err != nil {
				r.log.Error().Err(err).Msg("couldn't save the game during close")
			}
			r.emulator.Close()
		}()
	} else {
		r.emulator.Close()
	}
	close(r.done)

	if r.onClose != nil {
		r.onClose(r)
	}

	if r.rec != nil {
		r.rec.Set(false, "")
	}
}

func (r *Room) isRoomExisted() bool {
	// Check if room is in online storage
	_, err := r.storage.Load(r.id)
	if err == nil {
		return true
	}
	return hasStateSavedLocally(r.emulator.GetHashPath())
}

// SaveGame writes save state on the disk as well as
// uploads it to a cloud storage.
func (r *Room) SaveGame() error {
	if err := r.emulator.SaveGameState(); err != nil {
		return err
	}
	if err := r.storage.Save(r.id, r.emulator.GetHashPath()); err != nil {
		return err
	}
	r.log.Debug().Msg("Cloud save is successful")
	return nil
}

// saveOnlineRoomToLocal save online room to local.
// !Supports only one file of main save state.
func (r *Room) saveOnlineRoomToLocal(roomID string, savePath string) error {
	data, err := r.storage.Load(roomID)
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

func (r *Room) LoadGame() error   { return r.emulator.LoadGameState() }
func (r *Room) ToggleMultitap()   { r.emulator.ToggleMultitap() }
func (r *Room) IsEmpty() bool     { return r.users.IsEmpty() }
func (r *Room) IsRecording() bool { return r.rec != nil && r.rec.Enabled() }

func (r *Room) ToggleRecording(active bool, user string) {
	if r.rec == nil {
		return
	}
	r.log.Debug().Msgf("[REC] set: %v, %v", active, user)
	r.rec.Set(active, user)
}
