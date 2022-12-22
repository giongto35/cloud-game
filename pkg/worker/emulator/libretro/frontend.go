package libretro

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	conf "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
)

type Frontend struct {
	onVideo func(*emulator.GameFrame)
	onAudio func(*emulator.GameAudio)

	input InputState

	conf    conf.Emulator
	storage Storage

	// out frame size
	vw, vh int
	// draw threads
	th int

	done chan struct{}
	log  *logger.Logger

	mu sync.Mutex
}

// InputState stores full controller state.
// It consists of:
//   - uint16 button values
//   - int16 analog stick values
type (
	InputState [maxPort]State
	State      struct {
		keys uint32
		axes [dpadAxes]int32
	}
)

const (
	maxPort     = 4
	dpadAxes    = 4
	KeyPressed  = 1
	KeyReleased = 0
)

// NewFrontend implements Emulator interface for a Libretro frontend.
func NewFrontend(conf conf.Emulator, log *logger.Logger) (*Frontend, error) {
	log = log.Extend(log.With().Str("m", "Libretro"))
	ll := log.Extend(log.Level(logger.Level(conf.Libretro.LogLevel)).With())
	SetLibretroLogger(ll)

	// Check if room is on local storage, if not, pull from GCS to local storage
	log.Info().Msgf("Local storage path: %v", conf.Storage)
	if err := os.MkdirAll(conf.Storage, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create local storage path: %v, %w", conf.Storage, err)
	}

	path, err := filepath.Abs(conf.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to use emulator path: %v, %w", conf.LocalPath, err)
	}
	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create local path: %v, %w", conf.LocalPath, err)
	}
	log.Info().Msgf("Emulator save path is %v", path)
	Init(path)

	var store Storage = &StateStorage{Path: conf.Storage}
	if conf.Libretro.SaveCompression {
		store = &ZipStorage{Storage: store}
	}

	// set global link to the Libretro
	frontend = &Frontend{
		conf:    conf,
		storage: store,
		input:   NewGameSessionInput(),
		done:    make(chan struct{}),
		th:      conf.Threads,
		log:     log,
	}
	return frontend, nil
}

func (f *Frontend) Input(player int, data []byte) { f.input.setInput(player, data) }

func (f *Frontend) LoadMetadata(emu string) {
	libretroConf := f.conf.GetLibretroCoreConfig(emu)
	f.mu.Lock()
	coreLoad(emulator.Metadata{
		LibPath:       libretroConf.Lib,
		ConfigPath:    libretroConf.Config,
		IsGlAllowed:   libretroConf.IsGlAllowed,
		UsesLibCo:     libretroConf.UsesLibCo,
		HasMultitap:   libretroConf.HasMultitap,
		AutoGlContext: libretroConf.AutoGlContext,
	})
	f.mu.Unlock()
}

func (f *Frontend) Start() {
	// start only when it is available
	<-nano.reserved

	if err := f.LoadGameState(); err != nil {
		f.log.Error().Err(err).Msg("couldn't load a save file")
	}
	ticker := time.NewTicker(time.Second / time.Duration(nano.sysAvInfo.timing.fps))

	defer func() {
		ticker.Stop()
		nanoarchShutdown()
		f.log.Debug().Msgf("run loop finished")
	}()

	//lastFrameTime = time.Now().UnixNano()
	for {
		f.mu.Lock()
		run()
		f.mu.Unlock()
		select {
		case <-ticker.C:
			continue
		case <-f.done:
			return
		}
	}
}

func (f *Frontend) GetFrameSize() (int, int) {
	return int(nano.sysAvInfo.geometry.base_width), int(nano.sysAvInfo.geometry.base_height)
}

func (f *Frontend) SetAudio(ff func(*emulator.GameAudio)) { f.onAudio = ff }
func (f *Frontend) GetAudio() func(*emulator.GameAudio)   { return f.onAudio }
func (f *Frontend) SetVideo(ff func(*emulator.GameFrame)) { f.onVideo = ff }
func (f *Frontend) GetVideo() func(*emulator.GameFrame)   { return f.onVideo }
func (f *Frontend) GetFps() uint                          { return uint(nano.sysAvInfo.timing.fps) }
func (f *Frontend) GetHashPath() string                   { return f.storage.GetSavePath() }
func (f *Frontend) GetSRAMPath() string                   { return f.storage.GetSRAMPath() }
func (f *Frontend) GetSampleRate() uint                   { return uint(nano.sysAvInfo.timing.sample_rate) }
func (f *Frontend) LoadGame(path string) error            { return LoadGame(path) }
func (f *Frontend) LoadGameState() error                  { return f.Load() }
func (f *Frontend) HasVerticalFrame() bool                { return nano.rot != nil && nano.rot.IsEven }
func (f *Frontend) SaveGameState() error                  { return f.Save() }
func (f *Frontend) SetMainSaveName(name string)           { f.storage.SetMainSaveName(name) }
func (f *Frontend) SetViewport(width int, height int)     { f.vw, f.vh = width, height }
func (f *Frontend) ToggleMultitap()                       { toggleMultitap() }

func (f *Frontend) Close() {
	f.mu.Lock()
	f.SetViewport(0, 0)
	f.mu.Unlock()
	close(f.done)
	nano.reserved <- struct{}{}
}

// Save writes the current state to the filesystem.
func (f *Frontend) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if usesLibCo {
		return nil
	}

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

	if usesLibCo {
		return nil
	}

	ss, err := f.storage.Load(f.GetHashPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := restoreSaveState(ss); err != nil {
		return err
	}

	sram, err := f.storage.Load(f.GetSRAMPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if sram != nil {
		restoreSaveRAM(sram)
	}
	return nil
}

func NewGameSessionInput() InputState { return [maxPort]State{} }

// setInput sets input state for some player in a game session.
func (s *InputState) setInput(player int, data []byte) {
	atomic.StoreUint32(&s[player].keys, uint32(uint16(data[1])<<8+uint16(data[0])))
	for i, axes := 0, len(data); i < dpadAxes && i<<1+3 < axes; i++ {
		axis := i<<1 + 2
		atomic.StoreInt32(&s[player].axes[i], int32(data[axis+1])<<8+int32(data[axis]))
	}
}

// isKeyPressed checks if some button is pressed by any player.
func (s *InputState) isKeyPressed(port uint, key int) int {
	return int((atomic.LoadUint32(&s[port].keys) >> uint(key)) & 1)
}

// isDpadTouched checks if D-pad is used by any player.
func (s *InputState) isDpadTouched(port uint, axis uint) (shift int16) {
	return int16(atomic.LoadInt32(&s[port].axes[axis]))
}
