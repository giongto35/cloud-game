package nanoarch

import (
	"sync"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type Frontend struct {
	audio chan emulator.GameAudio
	video chan emulator.GameFrame
	input GameSessionInput

	meta    emulator.Metadata
	storage Storage

	// out frame size
	vw, vh int

	// draw threads
	th int

	done chan struct{}
	log  *logger.Logger

	mu sync.Mutex
}

// NewFrontend implements CloudEmulator interface for a Libretro frontend.
func NewFrontend(storage Storage, conf config.LibretroCoreConfig, threads int, log *logger.Logger) *Frontend {
	log = log.Extend(log.With().Str("[m]", "Libretro"))
	SetLibretroLogger(log)

	// set global link to the Libretro
	frontend = &Frontend{
		meta: emulator.Metadata{
			LibPath:       conf.Lib,
			ConfigPath:    conf.Config,
			Ratio:         conf.Ratio,
			IsGlAllowed:   conf.IsGlAllowed,
			UsesLibCo:     conf.UsesLibCo,
			HasMultitap:   conf.HasMultitap,
			AutoGlContext: conf.AutoGlContext,
		},
		storage: storage,
		video:   make(chan emulator.GameFrame, 6),
		audio:   make(chan emulator.GameAudio, 6),
		input:   NewGameSessionInput(),
		done:    make(chan struct{}, 1),
		th:      threads,
		log:     log,
	}
	return frontend
}

func (f *Frontend) Input(player int, data []byte) {
	f.input.setInput(player, uint16(data[1])<<8+uint16(data[0]), data)
}

func (f *Frontend) LoadMeta(path string) emulator.Metadata {
	f.mu.Lock()
	coreLoad(f.meta)
	f.mu.Unlock()
	coreLoadGame(path)
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
		run()
		f.mu.Unlock()

		select {
		case <-ticker.C:
			continue
		case <-f.done:
			nanoarchShutdown()
			close(f.video)
			close(f.audio)
			f.log.Debug().Msg("Closed Director")
			return
		}
	}
}

func (f *Frontend) GetAudio() chan emulator.GameAudio { return f.audio }

func (f *Frontend) GetVideo() chan emulator.GameFrame { return f.video }

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

// Save writes the current state to the filesystem.
func (f *Frontend) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := getSaveState()
	if err != nil {
		return err
	}
	if err := f.storage.Save(f.GetHashPath(), ss); err != nil {
		return err
	}

	if sram := getSaveRAM(); sram != nil {
		if err := f.storage.Save(f.GetSRAMPath(), sram); err != nil {
			return err
		}
	}
	return nil
}

// Load restores the state from the filesystem.
func (f *Frontend) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := f.storage.Load(f.GetHashPath())
	if err != nil {
		return err
	}
	if err := restoreSaveState(ss); err != nil {
		return err
	}

	sram, err := f.storage.Load(f.GetSRAMPath())
	if err != nil {
		return err
	}
	if sram != nil {
		restoreSaveRAM(sram)
	}
	return nil
}
