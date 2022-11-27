package nanoarch

import (
	"sync"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type Frontend struct {
	imageChannel chan<- emulator.GameFrame
	audioChannel chan<- emulator.GameAudio

	meta            emulator.Metadata
	gamePath        string
	roomID          string
	gameName        string
	isSavingLoading bool
	storage         Storage

	// out frame size
	vw, vh int

	// draw threads
	th int

	input GameSessionInput

	done chan struct{}
	log  *logger.Logger

	mu sync.Mutex
}

// NewFrontend implements CloudEmulator interface for a Libretro frontend.
func NewFrontend(roomID string, storage Storage, conf config.LibretroCoreConfig, threads int, log *logger.Logger) (*Frontend, chan emulator.GameFrame, chan emulator.GameAudio) {
	imageChannel := make(chan emulator.GameFrame, 6)
	audioChannel := make(chan emulator.GameAudio, 6)

	log = log.Extend(log.With().Str("[m]", "Libretro"))
	SetLibretroLogger(log)

	f := Frontend{
		meta: emulator.Metadata{
			LibPath:       conf.Lib,
			ConfigPath:    conf.Config,
			Ratio:         conf.Ratio,
			IsGlAllowed:   conf.IsGlAllowed,
			UsesLibCo:     conf.UsesLibCo,
			HasMultitap:   conf.HasMultitap,
			AutoGlContext: conf.AutoGlContext,
		},
		storage:      storage,
		imageChannel: imageChannel,
		audioChannel: audioChannel,
		input:        NewGameSessionInput(),
		roomID:       roomID,
		done:         make(chan struct{}, 1),
		th:           threads,
		log:          log,
	}

	// set global link to the Libretro
	frontend = &f
	return &f, imageChannel, audioChannel
}

func (f *Frontend) Input(player int, data []byte) {
	f.input.setInput(player, uint16(data[1])<<8+uint16(data[0]), data)
}

func (f *Frontend) LoadMeta(path string) emulator.Metadata {
	coreLoad(f.meta)
	coreLoadGame(path)
	f.gamePath = path
	return f.meta
}

func (f *Frontend) SetViewport(width int, height int) { f.vw, f.vh = width, height }

func (f *Frontend) Start() {
	if err := f.LoadGame(); err != nil {
		f.log.Error().Err(err).Msg("couldn't load a save file")
	}

	framerate := 1 / f.meta.Fps
	f.log.Info().Msgf("framerate: %vms", framerate)
	ticker := time.NewTicker(time.Second / time.Duration(f.meta.Fps))
	defer ticker.Stop()

	lastFrameTime = time.Now().UnixNano()

	for {
		f.mu.Lock()
		nanoarchRun()
		f.mu.Unlock()

		select {
		case <-ticker.C:
			continue
		case <-f.done:
			nanoarchShutdown()
			close(f.imageChannel)
			close(f.audioChannel)
			f.log.Debug().Msg("Closed Director")
			return
		}
	}
}

func (f *Frontend) SaveGame() error { return f.Save() }

func (f *Frontend) LoadGame() error { return f.Load() }

func (f *Frontend) ToggleMultitap() { toggleMultitap() }

func (f *Frontend) GetHashPath() string { return f.storage.GetSavePath() }

func (f *Frontend) GetSRAMPath() string { return f.storage.GetSRAMPath() }

func (f *Frontend) Close() {
	f.mu.Lock()
	f.SetViewport(0, 0)
	f.mu.Unlock()
	close(f.done)
}
