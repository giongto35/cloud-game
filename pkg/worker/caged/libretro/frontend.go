package libretro

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/graphics"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
)

type Emulator interface {
	SetAudioCb(func(app.Audio))
	SetVideoCb(func(app.Video))
	SetDataCb(func([]byte))
	LoadCore(name string)
	LoadGame(path string) error
	FPS() float64
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
	Input(player int, device byte, data []byte)
	// Scale returns set video scale factor and method
	Scale() (float64, string)
	Reset()
}

type Frontend struct {
	conf    config.Emulator
	done    chan struct{}
	doneEvt chan struct{} // closed when Start() fully exits
	closed  atomic.Bool
	log     *logger.Logger
	nano    *nanoarch.Nanoarch
	onAudio func(app.Audio)
	onData  func([]byte)
	onVideo func(app.Video)
	storage Storage
	scale   float64
	scaleM  string
	isGL    bool
	th      int // draw threads
	vw, vh  int // out frame size

	// directives

	// SpinBatch batches nanotime checks in the spin loop.
	// Higher values = check every N iterations, reducing syscall overhead.
	SpinBatch int

	// skipVideo used when new frame was too late
	skipVideo bool

	mu  sync.Mutex
	mui sync.Mutex

	// per-frame hooks (tournaments, replays), guarded by mu
	frame     uint64
	frameHook FrameHook
	inputSrc  InputSource
	srcBuf    [nanoarch.MaxPort]uint32

	DisableCanvasPool bool
	SaveOnClose       bool
	UniqueSaveDir     bool
	SaveStateFs       string
}

type Device byte

const (
	RetroPad = Device(nanoarch.RetroPad)
	Keyboard = Device(nanoarch.Keyboard)
	Mouse    = Device(nanoarch.Mouse)
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
	nano.SetLogger(log.Extend(log.Level(level).With()))

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
		conf:      conf,
		done:      make(chan struct{}),
		log:       log,
		onAudio:   noAudio,
		onData:    noData,
		onVideo:   noVideo,
		storage:   store,
		th:        conf.Threads,
		SpinBatch: 2048, // ~ 3GHz 3M / 2048 calls
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

	libExt := ""
	if ar, err := f.conf.Libretro.Cores.Repo.Guess(); err == nil {
		libExt = ar.Ext
	} else {
		f.log.Warn().Err(err).Msg("system arch guesser failed")
	}

	meta := nanoarch.Metadata{
		AutoGlContext:   conf.AutoGlContext,
		FrameDup:        f.conf.Libretro.Dup,
		Hacks:           conf.Hacks,
		HasVFR:          conf.VFR,
		Hid:             conf.Hid,
		Hid4rom:         conf.Hid4rom,
		IsGlAllowed:     conf.IsGlAllowed,
		LibPath:         conf.Lib,
		Options:         conf.Options,
		Options4rom:     conf.Options4rom,
		UsesLibCo:       conf.UsesLibCo,
		CoreAspectRatio: conf.CoreAspectRatio,
		KbMouseSupport:  conf.KbMouseSupport,
		LibExt:          libExt,
	}
	f.mu.Lock()
	f.SaveStateFs = conf.SaveStateFs
	if conf.UniqueSaveDir {
		f.UniqueSaveDir = true
		f.nano.SetSaveDirSuffix(f.storage.MainPath())
		f.log.Debug().Msgf("Using unique dir for saves: %v", f.storage.MainPath())
	}
	scale := 1.0
	if conf.Scale > 1 {
		scale = conf.Scale
		f.log.Debug().Msgf("Scale: x%v", scale)
	}
	f.storage.SetNonBlocking(conf.NonBlockingSave)
	f.scale = scale
	f.scaleM = conf.ScaleMethod
	f.isGL = conf.IsGlAllowed
	f.nano.CoreLoad(meta)
	f.mu.Unlock()
}

func (f *Frontend) handleAudio(audio unsafe.Pointer, samples int) {
	fr, _ := audioPool.Get().(*app.Audio)
	if fr == nil {
		fr = new(app.Audio)
	}
	// !to look if we need a copy
	fr.Data = unsafe.Slice((*byte)(audio), samples<<1)
	// due to audio buffering for opus fixed frames and const duration up in the hierarchy,
	// we skip Duration here
	f.onAudio(*fr)
	audioPool.Put(fr)
}

func (f *Frontend) handleVideo(data []byte, delta time.Duration, fi nanoarch.FrameInfo) {
	if f.conf.SkipLateFrames && f.skipVideo {
		return
	}

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
	if lastFrame != nil {
		f.onVideo(*lastFrame)
	}
}

func (f *Frontend) Shutdown() {
	f.mu.Lock()
	f.nano.Shutdown()
	f.SetAudioCb(noAudio)
	f.SetVideoCb(noVideo)
	lastFrame = nil
	f.mu.Unlock()
	f.log.Debug().Msgf("frontend shutdown done")
}

func (f *Frontend) linkNano(nano *nanoarch.Nanoarch) {
	f.nano = nano
	if nano == nil {
		return
	}
	f.nano.WaitReady() // start only when nano is available

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

	if f.isGL {
		// don't jump between threads
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	f.mui.Lock()
	f.done = make(chan struct{})
	f.doneEvt = make(chan struct{})
	f.nano.LastFrameTime = time.Now()

	defer func() {
		// Save game on quit if it was saved before (shared or click-saved).
		if f.SaveOnClose && f.HasSave() {
			f.log.Debug().Msg("save on quit")
			if err := f.Save(); err != nil {
				f.log.Error().Err(err).Msg("save on quit failed")
			}
		}
		f.mui.Unlock()
		f.Shutdown()
		close(f.doneEvt)
	}()

	if f.HasSave() {
		// advance 1 frame for Mupen, DOSBox save states
		// loading will work if autostart is selected for DOSBox apps
		f.Tick()
		if err := f.RestoreGameState(); err != nil {
			f.log.Error().Err(err).Msg("couldn't load a save file")
		}
	}

	if f.conf.AutosaveSec > 0 {
		// !to sync both for loops, can crash if the emulator starts later
		go f.autosave(f.conf.AutosaveSec)
	}

	// The main loop of Libretro

	targetFrameTimeNs := int64(float64(time.Second) / f.nano.VideoFramerate())

	// stop sleeping and start spinning in the remaining 1ms
	const spinThresholdNs = int64(1 * time.Millisecond)

	// how many frames will be considered not normal
	const lateFramesThreshold = 3

	batch := max(f.SpinBatch, 1)

	lastFrameStart := os.Nanotime()

	for {
		select {
		case <-f.done:
			return
		default:
			f.nano.NotifyFrameTime(targetFrameTimeNs / 1000)

			f.Tick()

			now := os.Nanotime()
			elapsed := now - lastFrameStart
			sleepTimeNs := targetFrameTimeNs - elapsed
			deadline := lastFrameStart + targetFrameTimeNs

			if sleepTimeNs > 0 {
				// SLEEP
				// if we have plenty of time, sleep to save CPU and
				// wake up slightly before the target time
				if sleepTimeNs > spinThresholdNs {
					time.Sleep(time.Duration(sleepTimeNs - spinThresholdNs))
				}

				// SPIN
				// if we are close to the target,
				// burn CPU and check the clock with ns resolution.
				// SpinBatch batches the time check to reduce
				// overhead while keeping sub-us precision.
				for {
					for range batch - 1 {
					}
					if os.Nanotime() >= deadline {
						break
					}
				}
				f.skipVideo = false
			} else {
				// lagging behind the target framerate so we don't sleep
				if f.conf.LogDroppedFrames {
					// !to make some stats counter instead
					f.log.Debug().Msgf("[] Frame drop: %v", time.Duration(elapsed))
				}
				f.skipVideo = true
			}

			// timer reset
			//
			// adding targetFrameTime to the previous start
			// prevents drift, if one frame was late,
			// we try to catch up in the next frame
			lastFrameStart = lastFrameStart + targetFrameTimeNs

			// if execution was paused or heavily delayed,
			// reset lastFrameStart so we don't try to run
			// a bunch of frames instantly to catch up
			if os.Nanotime()-lastFrameStart > targetFrameTimeNs*lateFramesThreshold {
				lastFrameStart = os.Nanotime()
			}
		}
	}
}

func (f *Frontend) LoadGame(path string) error {
	if f.UniqueSaveDir {
		f.copyFsMaybe(path)
	}
	return f.nano.LoadGame(path)
}

func (f *Frontend) AspectRatio() float32          { return f.nano.AspectRatio() }
func (f *Frontend) AudioSampleRate() int          { return f.nano.AudioSampleRate() }
func (f *Frontend) FPS() float64                  { return f.nano.VideoFramerate() }
func (f *Frontend) Flipped() bool                 { return f.nano.IsGL() }
func (f *Frontend) FrameSize() (int, int)         { return f.nano.BaseWidth(), f.nano.BaseHeight() }
func (f *Frontend) HasSave() bool                 { return os.Exists(f.HashPath()) }
func (f *Frontend) HashPath() string              { return f.storage.GetSavePath() }
func (f *Frontend) IsPortrait() bool              { return f.nano.IsPortrait() }
func (f *Frontend) KbMouseSupport() bool          { return f.nano.KbMouseSupport() }
func (f *Frontend) PixFormat() uint32             { return f.nano.Video.PixFmt.C }
func (f *Frontend) Reset()                        { f.mu.Lock(); defer f.mu.Unlock(); f.nano.Reset() }
func (f *Frontend) RestoreGameState() error       { return f.Load() }
func (f *Frontend) Rotation() uint                { return f.nano.Rot }
func (f *Frontend) SRAMPath() string              { return f.storage.GetSRAMPath() }
func (f *Frontend) SaveGameState() error          { return f.Save() }
func (f *Frontend) SaveStateName() string         { return filepath.Base(f.HashPath()) }
func (f *Frontend) Scale() (float64, string)      { return f.scale, f.scaleM }
func (f *Frontend) SetAudioCb(cb func(app.Audio)) { f.onAudio = cb }
func (f *Frontend) SetSessionId(name string)      { f.storage.SetMainSaveName(name) }
func (f *Frontend) SetDataCb(cb func([]byte))     { f.onData = cb }
func (f *Frontend) SetVideoCb(ff func(app.Video)) { f.onVideo = ff }
// FrameHook is called after every emulated frame with the frame number,
// the retropad buttons applied in that frame and a live view of the work RAM.
// It runs on the emulation thread while the emulator is locked: keep it fast and
// never call Frontend methods from it.
type FrameHook func(frame uint64, input [nanoarch.MaxPort]uint16, ram []byte)

// InputSource provides the buttons for a frame instead of the live input (replays).
// ok=false means the source is exhausted: the controllers are released.
type InputSource func(frame uint64) (keys [nanoarch.MaxPort]uint16, ok bool)

func (f *Frontend) Tick() {
	f.mu.Lock()
	if f.inputSrc != nil {
		keys, ok := f.inputSrc(f.frame)
		for i := range keys {
			f.srcBuf[i] = 0
			if ok {
				f.srcBuf[i] = uint32(keys[i])
			}
		}
		f.nano.SetInputOverride(&f.srcBuf)
	}
	f.nano.Run()
	if f.frameHook != nil {
		applied := f.nano.AppliedInput()
		var input [nanoarch.MaxPort]uint16
		for i, v := range applied {
			input[i] = uint16(v)
		}
		f.frameHook(f.frame, input, nanoarch.SystemRAM())
	}
	f.frame++
	f.mu.Unlock()
}

// SetFrameHook installs (or removes with nil) the per-frame hook.
func (f *Frontend) SetFrameHook(h FrameHook) { f.mu.Lock(); f.frameHook = h; f.mu.Unlock() }

// SetInputSource replaces the live input with a scripted one (nil restores the live input).
func (f *Frontend) SetInputSource(src InputSource) {
	f.mu.Lock()
	f.inputSrc = src
	if src == nil {
		f.nano.SetInputOverride(nil)
	}
	f.mu.Unlock()
}

// Frame returns the number of frames emulated since the game was loaded (or since ResetFrames).
func (f *Frontend) Frame() uint64 { f.mu.Lock(); defer f.mu.Unlock(); return f.frame }

// ResetFrames sets the frame counter to zero (start of a recording or a playback).
func (f *Frontend) ResetFrames() { f.mu.Lock(); f.frame = 0; f.mu.Unlock() }

// SnapshotState serializes the whole emulator state between frames.
func (f *Frontend) SnapshotState() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return nanoarch.SaveState()
}

// RestoreSnapshot restores a state made by SnapshotState.
func (f *Frontend) RestoreSnapshot(st []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return nanoarch.RestoreSaveState(st)
}

// SystemRAM returns a live view of the emulated work RAM. Use it only between frames
// on the emulation thread (e.g. from a frame hook).
func (f *Frontend) SystemRAM() []byte { return nanoarch.SystemRAM() }
func (f *Frontend) ViewportRecalculate()          { f.mu.Lock(); f.vw, f.vh = f.ViewportCalc(); f.mu.Unlock() }
func (f *Frontend) ViewportSize() (int, int)      { return f.vw, f.vh }

func (f *Frontend) Input(port int, device byte, data []byte) {
	switch Device(device) {
	case RetroPad:
		f.nano.InputRetropad(port, data)
	case Keyboard:
		f.nano.InputKeyboard(port, data)
	case Mouse:
		f.nano.InputMouse(port, data)
	}
}

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
	if f.closed.Swap(true) {
		return
	}
	close(f.done)

	f.mui.Lock()
	f.nano.Close()
	f.mui.Unlock()

	if f.doneEvt != nil {
		<-f.doneEvt
	}

	f.mui.Lock()
	if f.UniqueSaveDir && !f.HasSave() {
		if err := f.nano.DeleteSaveDir(); err != nil {
			f.log.Error().Msgf("couldn't delete save dir: %v", err)
		}
	}

	f.UniqueSaveDir = false
	f.SaveStateFs = ""

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

func (f *Frontend) IsSupported() error {
	return graphics.RGFWTryInit()
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

func (f *Frontend) copyFsMaybe(path string) {
	if f.SaveStateFs == "" {
		return
	}

	fileName := f.SaveStateFs
	hasPlaceholder := strings.HasPrefix(f.SaveStateFs, "*")
	if hasPlaceholder {
		game := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		fileName = strings.Replace(f.SaveStateFs, "*", game, 1)
	}

	fullPath := filepath.Join(f.nano.SaveDir(), fileName)

	if os.Exists(fullPath) {
		return
	}

	storePath := filepath.Dir(path)
	fsPath := filepath.Join(storePath, fileName)
	if os.Exists(fsPath) {
		if err := os.CopyFile(fsPath, fullPath); err != nil {
			f.log.Error().Err(err).Msgf("fs copy fail")
		} else {
			f.log.Debug().Msgf("copied fs %v to %v", fsPath, fullPath)
		}
	}
}
