package worker

import (
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/com"
	conf "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
)

type GamingRoom interface {
	GetId() string
	Close()
	CleanupUser(*Session)
	HasSave() bool
	StartEmulator()
	SaveGame() error
	LoadGame() error
	ToggleMultitap()
	HasUser(*Session) bool
	AddUser(*Session)
	PollUserInput(*Session)
	EnableAutosave(periodS int)
	GetEmulator() emulator.Emulator
	GetLog() *logger.Logger
}

type Room struct {
	id       string
	done     chan struct{}
	vEncoder *encoder.VideoEncoder
	users    com.NetMap[com.Uid, *Session] // a list of users in the room
	emulator emulator.Emulator
	onClose  func(self *Room)
	closed   bool
	log      *logger.Logger
}

func NewRoom(id string, game games.GameMetadata, onClose func(*Room), conf worker.Config, log *logger.Logger) *Room {
	if id == "" {
		id = games.GenerateRoomID(game.Name)
	}
	log = log.Extend(log.With().Str("room", id[:5]))
	log.Info().Str("game", game.Name).Send()
	room := &Room{id: id, users: com.NewNetMap[com.Uid, *Session](), done: make(chan struct{}), onClose: onClose, log: log}

	nano, err := libretro.NewFrontend(conf.Emulator, log)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	room.emulator = nano
	room.emulator.SetMainSaveName(id)
	room.emulator.LoadMetadata(conf.Emulator.GetEmulator(game.Type, game.Path))
	err = room.emulator.LoadGame(game.FullPath())
	if err != nil {
		log.Fatal().Err(err).Msgf("couldn't load the game %v", game)
	}
	// calc output frame size and rotation
	w, h := room.whatsFrame(conf.Emulator)
	if room.emulator.HasVerticalFrame() {
		w, h = h, w
	}
	room.emulator.SetViewport(w, h)

	room.initVideo(w, h, conf.Encoder.Video)
	room.initAudio(int(room.emulator.GetSampleRate()), conf.Encoder.Audio)

	log.Info().Str("room", room.GetId()).
		Str("game", game.Name).
		Msg("New room")
	return room
}

func (r *Room) GetEmulator() emulator.Emulator { return r.emulator }
func (r *Room) GetId() string                  { return r.id }
func (r *Room) GetLog() *logger.Logger         { return r.log }
func (r *Room) HasSave() bool                  { return os.Exists(r.emulator.GetHashPath()) }
func (r *Room) HasUser(u *Session) bool        { return r != nil && r.users.Has(u.id) }
func (r *Room) IsEmpty() bool                  { return r.users.IsEmpty() }
func (r *Room) LoadGame() error                { return r.emulator.LoadGameState() }
func (r *Room) SaveGame() error                { return r.emulator.SaveGameState() }
func (r *Room) StartEmulator()                 { go r.emulator.Start() }
func (r *Room) ToggleMultitap()                { r.emulator.ToggleMultitap() }

func (r *Room) EnableAutosave(periodSec int) {
	r.log.Info().Msgf("Autosave every [%vs]", periodSec)
	ticker := time.NewTicker(time.Duration(periodSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.closed {
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

func (r *Room) whatsFrame(conf conf.Emulator) (ww int, hh int) {
	w, h := r.emulator.GetFrameSize()
	// nwidth, nheight are the WebRTC output size
	var nwidth, nheight int
	emu, ar := conf, conf.AspectRatio

	if ar.Keep {
		baseAspectRatio := float64(w) / float64(ar.Height)
		nwidth, nheight = ar.ResizeToAspect(baseAspectRatio, ar.Width, ar.Height)
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

func (r *Room) PollUserInput(session *Session) {
	r.log.Debug().Msg("Start session input poll")
	session.GetPeerConn().OnMessage = func(data []byte) { r.emulator.Input(session.GetPlayerIndex(), data) }
}

func (r *Room) AddUser(user *Session) {
	r.users.Add(user)
	user.SetRoom(r)
	r.log.Debug().Str("user", user.Id().String()).Msg("User has joined the room")
}

func (r *Room) CleanupUser(user *Session) {
	user.SetRoom(nil)
	if r.HasUser(user) {
		r.users.Remove(user)
		r.log.Debug().Str("user", user.Id().String()).Msg("User has left the room")
	}
	if r.IsEmpty() {
		r.log.Debug().Msg("The room is empty")
		r.Close()
	}
}

func (r *Room) Close() {
	r.log.Debug().Msg("Closing the room")
	if r.closed {
		r.log.Debug().Msg("Close room skip")
		return
	}

	r.closed = true

	// Save game before quit. Only save for game which was previous saved to avoid flooding database
	if r.HasSave() {
		r.log.Debug().Msg("Save game before closing room")
		if err := r.SaveGame(); err != nil {
			r.log.Error().Err(err).Msg("couldn't save the game during close")
		}
	}
	r.emulator.Close()
	close(r.done)

	if r.vEncoder != nil {
		r.vEncoder.Stop()
	}

	if r.onClose != nil {
		r.onClose(r)
	}
}
