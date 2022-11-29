package nanoarch

import (
	"sync"
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
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
func NewFrontend(game games.GameMetadata, conf conf.Emulator, log *logger.Logger) *Frontend {
	emulatorGuess := conf.GetEmulator(game.Type, game.Path)
	libretroConf := conf.GetLibretroCoreConfig(emulatorGuess)

	log.Info().Msgf("Image processing threads = %v", conf.Threads)

	log = log.Extend(log.With().Str("[m]", "Libretro"))
	SetLibretroLogger(log)

	// Check if room is on local storage, if not, pull from GCS to local storage
	var store Storage = &StateStorage{Path: conf.Storage}
	if conf.Libretro.SaveCompression {
		store = &ZipStorage{Storage: store}
	}

	// set global link to the Libretro
	frontend = &Frontend{
		meta: emulator.Metadata{
			LibPath:       libretroConf.Lib,
			ConfigPath:    libretroConf.Config,
			IsGlAllowed:   libretroConf.IsGlAllowed,
			UsesLibCo:     libretroConf.UsesLibCo,
			HasMultitap:   libretroConf.HasMultitap,
			AutoGlContext: libretroConf.AutoGlContext,
		},
		storage: store,
		video:   make(chan emulator.GameFrame, 6),
		audio:   make(chan emulator.GameAudio, 6),
		input:   NewGameSessionInput(),
		done:    make(chan struct{}, 1),
		th:      conf.Threads,
		log:     log,
	}
	return frontend
}

func (f *Frontend) Input(player int, data []byte) { f.input.setInput(player, data) }

func (f *Frontend) LoadMeta(path string) (*emulator.Metadata, error) {
	f.mu.Lock()
	coreLoad(f.meta)
	f.mu.Unlock()
	if err := LoadGame(path); err != nil {
		return nil, err
	}
	return &f.meta, nil
}

func (f *Frontend) SetViewport(width int, height int) { f.vw, f.vh = width, height }

func (f *Frontend) SetMainSaveName(name string) { f.storage.SetMainSaveName(name) }

func (f *Frontend) Start() {
	if err := f.LoadGameState(); err != nil {
		f.log.Error().Err(err).Msg("couldn't load a save file")
	}

	ticker := time.NewTicker(time.Second / time.Duration(nano.sysAvInfo.timing.fps))
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

func (f *Frontend) SaveGameState() error { return f.Save() }

func (f *Frontend) LoadGameState() error { return f.Load() }

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
