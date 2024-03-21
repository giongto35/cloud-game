package libretro

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
)

type Emulator interface {
	SetAudioCb(func(app.Audio))
	SetVideoCb(func(app.Video))
	SetDataCb(func([]byte))
	LoadCore(name string)
	LoadGame(path string) error
	FPS() int
	Flipped() bool
	Rotation() uint
	PixFormat() uint32
	AudioSampleRate() int
	IsPortrait() bool
	// Start is called after LoadGame
	Start()
	// ViewportRecalculate calculates output resolution with aspect and scale
	ViewportRecalculate()
	RestoreGameState() error
	// SetSessionId sets distinct name for the game session (in order to save/load it later)
	SetSessionId(name string)
	SaveGameState() error
	SaveStateName() string
	// HashPath returns the path emulator will save state to
	HashPath() string
	// HasSave returns true if the current ROM was saved before
	HasSave() bool
	// Close will be called when the game is done
	Close()
	// Input passes input to the emulator
	Input(player int, data []byte)
	// Scale returns set video scale factor
	Scale() float64
}

type Frontend struct {
	conf    config.Emulator
	done    chan struct{}
	input   InputState
	log     *logger.Logger
	nano    *nanoarch.Nanoarch
	onAudio func(app.Audio)
	onData  func([]byte)
	onVideo func(app.Video)
	storage Storage
	scale   float64
	th      int // draw threads
	vw, vh  int // out frame size

	mu  sync.Mutex
	mui sync.Mutex

	DisableCanvasPool bool
	SaveOnClose       bool
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
	maxPort  = 4
	dpadAxes = 4
)

var (
	audioPool sync.Pool
	noAudio   = func(app.Audio) {}
	noData    = func([]byte) {}
	noVideo   = func(app.Video) {}
	videoPool sync.Pool
	lastFrame *app.Video
)

// NewFrontend implements Emulator interface for a Libretro frontend.
func NewFrontend(conf config.Emulator, log *logger.Logger) (*Frontend, error) {
	path, err := filepath.Abs(conf.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to use emulator path: %v, %w", conf.LocalPath, err)
	}
	if err := os.CheckCreateDir(path); err != nil {
		return nil, fmt.Errorf("failed to create local path: %v, %w", conf.LocalPath, err)
	}
	log.Info().Msgf("Emulator save path is %v", path)

	// we use the global Nanoarch instance from nanoarch
	nano := nanoarch.NewNano(path)

	log = log.Extend(log.With().Str("m", "Libretro"))
	level := logger.Level(conf.Libretro.LogLevel)
	if level == logger.DebugLevel {
		level = logger.TraceLevel
		nano.SetLogger(log.Extend(log.Level(level).With()))
	} else {
		nano.SetLogger(log)
	}

	// Check if room is on local storage, if not, pull from GCS to local storage
	log.Info().Msgf("Local storage path: %v", conf.Storage)
	if err := os.CheckCreateDir(conf.Storage); err != nil {
		return nil, fmt.Errorf("failed to create local storage path: %v, %w", conf.Storage, err)
	}

	var store Storage = &StateStorage{Path: conf.Storage}
	if conf.Libretro.SaveCompression {
		store = &ZipStorage{Storage: store}
	}

	// set global link to the Libretro
	f := &Frontend{
		conf:    conf,
		done:    make(chan struct{}),
		input:   NewGameSessionInput(),
		log:     log,
		onAudio: noAudio,
		onData:  noData,
		onVideo: noVideo,
		storage: store,
		th:      conf.Threads,
	}
	f.linkNano(nano)

	if conf.Libretro.DebounceMs > 0 {
		t := time.Duration(conf.Libretro.DebounceMs) * time.Millisecond
		f.nano.SetVideoDebounce(t)
		f.log.Debug().Msgf("set debounce time: %v", t)
	}

	return f, nil
}

func (f *Frontend) LoadCore(emu string) {
	conf := f.conf.GetLibretroCoreConfig(emu)
	meta := nanoarch.Metadata{
		AutoGlContext:   conf.AutoGlContext,
		FrameDup:        f.conf.Libretro.Dup,
		Hacks:           conf.Hacks,
		HasVFR:          conf.VFR,
		Hid:             conf.Hid,
		IsGlAllowed:     conf.IsGlAllowed,
		LibPath:         conf.Lib,
		Options:         conf.Options,
		UsesLibCo:       conf.UsesLibCo,
		CoreAspectRatio: conf.CoreAspectRatio,
	}
	f.mu.Lock()
	scale := 1.0
	if conf.Scale > 1 {
		scale = conf.Scale
		f.log.Debug().Msgf("Scale: x%v", scale)
	}
	f.scale = scale
	f.nano.CoreLoad(meta)
	f.mu.Unlock()
}

func (f *Frontend) handleAudio(audio unsafe.Pointer, samples int) {
	fr, _ := audioPool.Get().(*app.Audio)
	if fr == nil {
		fr = new(app.Audio)
	}
	// !to look if we need a copy
	fr.Data = unsafe.Slice((*int16)(audio), samples)
	// due to audio buffering for opus fixed frames and const duration up in the hierarchy,
	// we skip Duration here
	f.onAudio(*fr)
	audioPool.Put(fr)
}

func (f *Frontend) handleVideo(data []byte, delta int32, fi nanoarch.FrameInfo) {
	fr, _ := videoPool.Get().(*app.Video)
	if fr == nil {
		fr = new(app.Video)
	}
	fr.Frame.Data = data
	fr.Frame.W = int(fi.W)
	fr.Frame.H = int(fi.H)
	fr.Frame.Stride = int(fi.Stride)
	fr.Duration = delta

	lastFrame = fr
	f.onVideo(*fr)

	videoPool.Put(fr)
}

func (f *Frontend) handleDup() {
	f.onVideo(*lastFrame)
}

func (f *Frontend) Shutdown() {
	f.mu.Lock()
	f.nano.Shutdown()
	f.SetAudioCb(noAudio)
	f.SetVideoCb(noVideo)
	f.mu.Unlock()
	f.log.Debug().Msgf("frontend shutdown done")
}

func (f *Frontend) linkNano(nano *nanoarch.Nanoarch) {
	f.nano = nano
	if nano == nil {
		return
	}
	f.nano.WaitReady() // start only when nano is available

	f.nano.OnKeyPress = f.input.isKeyPressed
	f.nano.OnDpad = f.input.isDpadTouched
	f.nano.OnVideo = f.handleVideo
	f.nano.OnAudio = f.handleAudio
	f.nano.OnDup = f.handleDup
}

func (f *Frontend) SetVideoChangeCb(fn func()) {
	if f.nano != nil {
		f.nano.OnSystemAvInfo = fn
	}
}

func (f *Frontend) Start() {
	f.log.Debug().Msgf("frontend start")
	if f.nano.Stopped.Load() {
		f.log.Warn().Msgf("frontend stopped during the start")
		f.mui.Lock()
		defer f.mui.Unlock()
		f.Shutdown()
		return
	}

	f.mui.Lock()
	f.done = make(chan struct{})
	f.nano.LastFrameTime = time.Now().UnixNano()

	defer func() {
		// Save game on quit if it was saved before (shared or click-saved).
		if f.SaveOnClose && f.HasSave() {
			f.log.Debug().Msg("save on quit")
			if err := f.Save(); err != nil {
				f.log.Error().Err(err).Msg("save on quit failed")
			}
		}
		f.Shutdown()
	}()
	defer f.mui.Unlock()

	if f.HasSave() {
		// advance 1 frame for Mupen, DOSBox save states
		// loading will work if autostart is selected for DOSBox apps
		f.Tick()
		if err := f.RestoreGameState(); err != nil {
			f.log.Error().Err(err).Msg("couldn't load a save file")
		}
	}

	ticker := time.NewTicker(time.Second / time.Duration(f.nano.VideoFramerate()))
	defer ticker.Stop()

	if f.conf.AutosaveSec > 0 {
		// !to sync both for loops, can crash if the emulator starts later
		go f.autosave(f.conf.AutosaveSec)
	}

	for {
		select {
		case <-ticker.C:
			f.Tick()
		case <-f.done:
			return
		}
	}
}

func (f *Frontend) AspectRatio() float32          { return f.nano.AspectRatio() }
func (f *Frontend) AudioSampleRate() int          { return f.nano.AudioSampleRate() }
func (f *Frontend) FPS() int                      { return f.nano.VideoFramerate() }
func (f *Frontend) Flipped() bool                 { return f.nano.IsGL() }
func (f *Frontend) FrameSize() (int, int)         { return f.nano.BaseWidth(), f.nano.BaseHeight() }
func (f *Frontend) HasSave() bool                 { return os.Exists(f.HashPath()) }
func (f *Frontend) HashPath() string              { return f.storage.GetSavePath() }
func (f *Frontend) Input(player int, data []byte) { f.input.setInput(player, data) }
func (f *Frontend) IsPortrait() bool              { return f.nano.IsPortrait() }
func (f *Frontend) LoadGame(path string) error    { return f.nano.LoadGame(path) }
func (f *Frontend) PixFormat() uint32             { return f.nano.Video.PixFmt.C }
func (f *Frontend) RestoreGameState() error       { return f.Load() }
func (f *Frontend) Rotation() uint                { return f.nano.Rot }
func (f *Frontend) SRAMPath() string              { return f.storage.GetSRAMPath() }
func (f *Frontend) SaveGameState() error          { return f.Save() }
func (f *Frontend) SaveStateName() string         { return filepath.Base(f.HashPath()) }
func (f *Frontend) Scale() float64                { return f.scale }
func (f *Frontend) SetAudioCb(cb func(app.Audio)) { f.onAudio = cb }
func (f *Frontend) SetSessionId(name string)      { f.storage.SetMainSaveName(name) }
func (f *Frontend) SetDataCb(cb func([]byte))     { f.onData = cb }
func (f *Frontend) SetVideoCb(ff func(app.Video)) { f.onVideo = ff }
func (f *Frontend) Tick()                         { f.mu.Lock(); f.nano.Run(); f.mu.Unlock() }
func (f *Frontend) ViewportRecalculate()          { f.mu.Lock(); f.vw, f.vh = f.ViewportCalc(); f.mu.Unlock() }
func (f *Frontend) ViewportSize() (int, int)      { return f.vw, f.vh }

func (f *Frontend) ViewportCalc() (nw int, nh int) {
	w, h := f.FrameSize()
	nw, nh = w, h

	if f.IsPortrait() {
		nw, nh = nh, nw
	}

	f.log.Debug().Msgf("viewport: %dx%d -> %dx%d", w, h, nw, nh)

	return
}

func (f *Frontend) Close() {
	f.log.Debug().Msgf("frontend close")
	close(f.done)

	f.mui.Lock()
	f.nano.Close()
	f.mui.Unlock()
	f.log.Debug().Msgf("frontend closed")
}

// Save writes the current state to the filesystem.
func (f *Frontend) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := nanoarch.SaveState()
	if err != nil {
		return err
	}
	if err := f.storage.Save(f.HashPath(), ss); err != nil {
		return err
	}
	ss = nil

	if sram := nanoarch.SaveRAM(); sram != nil {
		if err := f.storage.Save(f.SRAMPath(), sram); err != nil {
			return err
		}
		sram = nil
	}
	return nil
}

// Load restores the state from the filesystem.
func (f *Frontend) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := f.storage.Load(f.HashPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := nanoarch.RestoreSaveState(ss); err != nil {
		return err
	}

	sram, err := f.storage.Load(f.SRAMPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if sram != nil {
		nanoarch.RestoreSaveRAM(sram)
	}
	return nil
}

func (f *Frontend) autosave(periodSec int) {
	f.log.Info().Msgf("Autosave every [%vs]", periodSec)
	ticker := time.NewTicker(time.Duration(periodSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if f.nano.IsStopped() {
				return
			}
			if err := f.Save(); err != nil {
				f.log.Error().Msgf("Autosave failed: %v", err)
			} else {
				f.log.Debug().Msgf("Autosave done")
			}
		case <-f.done:
			return
		}
	}
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
