package nanoarch

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/graphics"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
)

/*
#include "libretro.h"
#include "nanoarch.h"
#include <stdlib.h>
*/
import "C"

var (
	RGBA5551    = PixFmt{C: 0, BPP: 2} // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	RGBA8888Rev = PixFmt{C: 1, BPP: 4} // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	RGB565      = PixFmt{C: 2, BPP: 2} // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

type Nanoarch struct {
	Handlers

	keyboard KeyboardState
	mouse    MouseState
	retropad InputState

	keyboardCb    *C.struct_retro_keyboard_callback
	frameTimeCb   *C.struct_retro_frame_time_callback
	LastFrameTime time.Time
	LibCo         bool
	meta          Metadata
	options       map[string]string
	options4rom   map[string]map[string]string
	reserved      chan struct{} // limits concurrent use
	Rot           uint
	serializeSize C.size_t
	Stopped       atomic.Bool
	sys           struct {
		av  C.struct_retro_system_av_info
		i   C.struct_retro_system_info
		api C.unsigned
	}
	tickTime         time.Duration
	cSaveDirectory   *C.char
	cSystemDirectory *C.char
	cUserName        *C.char
	Video            struct {
		gl struct {
			enabled bool
			autoCtx bool
		}
		hw     *C.struct_retro_hw_render_callback
		PixFmt PixFmt
	}
	vfr                      bool
	Aspect                   bool
	glCtx                    *graphics.RGFW
	hackSkipHwContextDestroy bool
	hackSkipSameThreadSave   bool
	limiter                  func(func())
	log                      *logger.Logger

	audioBuffCb    *C.struct_retro_audio_buffer_status_callback
	lastFrameDelta float64 // most recent frame delta (ns)
	audioDrift     float64 // cumulative timing drift for frameskip
}

type Handlers struct {
	OnAudio        func(ptr unsafe.Pointer, frames int)
	OnVideo        func(data []byte, delta time.Duration, fi FrameInfo)
	OnDup          func()
	OnSystemAvInfo func()
}

type FrameInfo struct {
	W      uint
	H      uint
	Stride uint
}

type Metadata struct {
	FrameDup        bool
	LibPath         string // the full path to some emulator lib
	IsGlAllowed     bool
	UsesLibCo       bool
	AutoGlContext   bool
	HasVFR          bool
	Options         map[string]string
	Options4rom     map[string]map[string]string
	Hacks           []string
	Hid             map[int][]int
	CoreAspectRatio bool
	KbMouseSupport  bool
	LibExt          string
}

type PixFmt struct {
	C   uint32
	BPP uint
}

func (p PixFmt) String() string {
	switch p.C {
	case 0:
		return "RGBA5551/2"
	case 1:
		return "RGBA8888Rev/4"
	case 2:
		return "RGB565/2"
	default:
		return fmt.Sprintf("Unknown (%v/%v)", p.C, p.BPP)
	}
}

// Nan0 is a global link for C callbacks to Go
var Nan0 = Nanoarch{
	reserved: make(chan struct{}, 1), // this thing forbids concurrent use of the emulator
	Stopped:  atomic.Bool{},
	limiter:  func(fn func()) { fn() },
	Handlers: Handlers{
		OnAudio: func(unsafe.Pointer, int) {},
		OnVideo: func([]byte, time.Duration, FrameInfo) {},
		OnDup:   func() {},
	},
}

// init provides a global single instance lock
// !to remove when isolated properly
func init() { Nan0.reserved <- struct{}{} }

func NewNano(localPath string) *Nanoarch {
	nano := &Nan0
	nano.cSaveDirectory = C.CString(localPath + "/legacy_save")
	nano.cSystemDirectory = C.CString(localPath + "/system")
	nano.cUserName = C.CString("retro")
	return nano
}

func (n *Nanoarch) AspectRatio() float32             { return float32(n.sys.av.geometry.aspect_ratio) }
func (n *Nanoarch) AudioSampleRate() int             { return int(n.sys.av.timing.sample_rate) }
func (n *Nanoarch) VideoFramerate() float64          { return float64(n.sys.av.timing.fps) }
func (n *Nanoarch) IsPortrait() bool                 { return 90 == n.Rot%180 }
func (n *Nanoarch) KbMouseSupport() bool             { return n.meta.KbMouseSupport }
func (n *Nanoarch) BaseWidth() int                   { return int(n.sys.av.geometry.base_width) }
func (n *Nanoarch) BaseHeight() int                  { return int(n.sys.av.geometry.base_height) }
func (n *Nanoarch) WaitReady()                       { <-n.reserved }
func (n *Nanoarch) Close()                           { n.Stopped.Store(true); n.reserved <- struct{}{} }
func (n *Nanoarch) SetLogger(log *logger.Logger)     { n.log = log }
func (n *Nanoarch) SetVideoDebounce(t time.Duration) { n.limiter = NewLimit(t) }
func (n *Nanoarch) SaveDir() string {
	if n.cSaveDirectory == nil {
		return ""
	}
	return C.GoString(n.cSaveDirectory)
}
func (n *Nanoarch) SetSaveDirSuffix(sx string) {
	dir := C.GoString(n.cSaveDirectory) + "/" + sx
	err := os.CheckCreateDir(dir)
	if err != nil {
		n.log.Error().Msgf("couldn't create %v, %v", dir, err)
	}
	if n.cSaveDirectory != nil {
		C.free(unsafe.Pointer(n.cSaveDirectory))
	}
	n.cSaveDirectory = C.CString(dir)
}
func (n *Nanoarch) DeleteSaveDir() error {
	if n.cSaveDirectory == nil {
		return nil
	}

	dir := C.GoString(n.cSaveDirectory)
	return os.RemoveAll(dir)
}

func (n *Nanoarch) freeCStrings() {
	if n.cUserName != nil {
		C.free(unsafe.Pointer(n.cUserName))
		n.cUserName = nil
	}
	if n.cSaveDirectory != nil {
		C.free(unsafe.Pointer(n.cSaveDirectory))
		n.cSaveDirectory = nil
	}
	if n.cSystemDirectory != nil {
		C.free(unsafe.Pointer(n.cSystemDirectory))
		n.cSystemDirectory = nil
	}
}

func (n *Nanoarch) CoreLoad(meta Metadata) {
	var err error
	n.meta = meta
	n.LibCo = meta.UsesLibCo
	n.vfr = meta.HasVFR
	n.Aspect = meta.CoreAspectRatio
	n.Video.gl.autoCtx = meta.AutoGlContext
	n.Video.gl.enabled = meta.IsGlAllowed

	Nan0.audioBuffCb = nil

	thread.SwitchGraphics(n.Video.gl.enabled)

	// hacks
	Nan0.hackSkipHwContextDestroy = meta.HasHack("skip_hw_context_destroy")
	Nan0.hackSkipSameThreadSave = meta.HasHack("skip_same_thread_save")

	// reset controllers
	n.retropad = InputState{}
	n.keyboardCb = nil
	n.keyboard = KeyboardState{}
	n.mouse = MouseState{}

	n.options = maps.Clone(meta.Options)
	n.options4rom = meta.Options4rom

	corePath := meta.LibPath + meta.LibExt
	coreLib, err = open(corePath)
	if err != nil {
		n.log.Fatal().Err(err).Msgf("core load: %s", corePath)
	}

	retroInit = coreLib.load("retro_init")
	retroDeinit = coreLib.load("retro_deinit")
	retroAPIVersion = coreLib.load("retro_api_version")
	retroGetSystemInfo = coreLib.load("retro_get_system_info")
	retroGetSystemAVInfo = coreLib.load("retro_get_system_av_info")
	retroSetEnvironment = coreLib.load("retro_set_environment")
	retroSetVideoRefresh = coreLib.load("retro_set_video_refresh")
	retroSetInputPoll = coreLib.load("retro_set_input_poll")
	retroSetInputState = coreLib.load("retro_set_input_state")
	retroSetAudioSample = coreLib.load("retro_set_audio_sample")
	retroSetAudioSampleBatch = coreLib.load("retro_set_audio_sample_batch")
	retroReset = coreLib.load("retro_reset")
	retroRun = coreLib.load("retro_run")
	retroLoadGame = coreLib.load("retro_load_game")
	retroUnloadGame = coreLib.load("retro_unload_game")
	retroSerializeSize = coreLib.load("retro_serialize_size")
	retroSerialize = coreLib.load("retro_serialize")
	retroUnserialize = coreLib.load("retro_unserialize")
	retroSetControllerPortDevice = coreLib.load("retro_set_controller_port_device")
	retroGetMemorySize = coreLib.load("retro_get_memory_size")
	retroGetMemoryData = coreLib.load("retro_get_memory_data")

	C.bridge_retro_set_environment(retroSetEnvironment, C.core_environment_cgo)
	C.bridge_retro_set_input_state(retroSetInputState, C.core_input_state_cgo)
	C.bridge_set_callback(retroSetVideoRefresh, C.core_video_refresh_cgo)
	C.bridge_set_callback(retroSetInputPoll, C.core_input_poll_cgo)
	C.bridge_set_callback(retroSetAudioSample, C.core_audio_sample_cgo)
	C.bridge_set_callback(retroSetAudioSampleBatch, C.core_audio_sample_batch_cgo)

	if n.LibCo {
		C.same_thread(retroInit)
	} else {
		C.bridge_call(retroInit)
	}

	n.sys.api = C.bridge_retro_api_version(retroAPIVersion)
	C.bridge_retro_get_system_info(retroGetSystemInfo, &n.sys.i)
	n.log.Info().Msgf("System >>> %v (%v) [%v] nfp: %v, api: %v",
		C.GoString(n.sys.i.library_name), C.GoString(n.sys.i.library_version),
		C.GoString(n.sys.i.valid_extensions), bool(n.sys.i.need_fullpath),
		uint(n.sys.api))
}

func (n *Nanoarch) LoadGame(path string) error {
	game := C.struct_retro_game_info{}

	big := bool(n.sys.i.need_fullpath) // big ROMs are loaded by cores later
	if big {
		size, err := os.StatSize(path)
		if err != nil {
			return err
		}
		game.size = C.size_t(size)
	} else {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// !to pin in 1.21
		ptr := unsafe.Pointer(C.CBytes(bytes))
		game.data = ptr
		game.size = C.size_t(len(bytes))
		defer C.free(ptr)
	}
	fp := C.CString(path)
	defer C.free(unsafe.Pointer(fp))
	game.path = fp

	n.log.Debug().Msgf("ROM - big: %v, size: %v", big, byteCountBinary(int64(game.size)))

	// maybe some custom options
	if n.options4rom != nil {
		romName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, ok := n.options4rom[romName]; ok {
			for k, v := range n.options4rom[romName] {
				n.options[k] = v
				n.log.Debug().Msgf("Replace: %v=%v", k, v)
			}
		}
	}

	if ok := C.bridge_retro_load_game(retroLoadGame, &game); !ok {
		return fmt.Errorf("core failed to load ROM: %v", path)
	}

	var av C.struct_retro_system_av_info
	C.bridge_retro_get_system_av_info(retroGetSystemAVInfo, &av)
	n.log.Info().Msgf("System A/V >>> %vx%v (%vx%v), [%vfps], AR [%v], audio [%vHz]",
		av.geometry.base_width, av.geometry.base_height,
		av.geometry.max_width, av.geometry.max_height,
		av.timing.fps, av.geometry.aspect_ratio, av.timing.sample_rate,
	)
	if isGeometryDifferent(av.geometry) {
		geometryChange(av.geometry)
	}
	n.sys.av = av

	n.serializeSize = C.bridge_retro_serialize_size(retroSerializeSize)
	n.log.Info().Msgf("Save file size: %v", byteCountBinary(int64(n.serializeSize)))

	Nan0.tickTime = time.Duration(float64(time.Second) / float64(n.sys.av.timing.fps))
	if n.vfr {
		n.log.Info().Msgf("variable framerate (VFR) is enabled")
	}

	n.Stopped.Store(false)

	if n.Video.gl.enabled {
		if n.LibCo {
			C.same_thread(C.init_video_cgo)
			C.same_thread(unsafe.Pointer(Nan0.Video.hw.context_reset))
		} else {
			runtime.LockOSThread()
			initVideo()
			C.bridge_context_reset(Nan0.Video.hw.context_reset)
			runtime.UnlockOSThread()
		}
	}

	// set default controller types on all ports
	// needed for nestopia
	for i := range maxPort {
		C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(i), C.RETRO_DEVICE_JOYPAD)
	}

	// map custom devices to ports
	for k, v := range n.meta.Hid {
		for _, device := range v {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(k), C.unsigned(device))
			n.log.Debug().Msgf("set custom port-device: %v:%v", k, device)
		}
	}

	n.LastFrameTime = time.Now()

	return nil
}

func (n *Nanoarch) Shutdown() {
	Nan0.audioBuffCb = nil
	if n.LibCo {
		thread.Main(func() {
			C.same_thread(retroUnloadGame)
			C.same_thread(retroDeinit)
			if n.Video.gl.enabled {
				C.same_thread(C.deinit_video_cgo)
			}
			C.same_thread(C.same_thread_stop)
		})
	} else {
		if n.Video.gl.enabled {
			thread.Main(func() {
				// running inside a go routine, lock the thread to make sure the OpenGL context stays current
				runtime.LockOSThread()
				if err := n.glCtx.BindContext(); err != nil {
					n.log.Error().Err(err).Msg("ctx switch fail")
				}
			})
		}
		C.bridge_call(retroUnloadGame)
		C.bridge_call(retroDeinit)
		if n.Video.gl.enabled {
			thread.Main(func() {
				deinitVideo()
				runtime.UnlockOSThread()
			})
		}
	}

	setRotation(0)
	Nan0.sys.av = C.struct_retro_system_av_info{}
	if err := coreLib.close(); err != nil {
		n.log.Error().Err(err).Msg("lib close failed")
	}
	n.options = nil
	n.options4rom = nil
	Nan0.audioBuffCb = nil
	n.freeCStrings()
}

func (n *Nanoarch) Reset() {
	C.bridge_call(retroReset)
}

func (n *Nanoarch) syncInputToCache() {
	n.retropad.SyncToCache()
	if n.keyboardCb != nil {
		n.keyboard.SyncToCache()
	}
	n.mouse.SyncToCache()
}

func (n *Nanoarch) Run() {
	n.syncInputToCache()

	if n.LibCo {
		C.same_thread(retroRun)
	} else {
		if n.Video.gl.enabled {
			runtime.LockOSThread()
			if err := n.glCtx.BindContext(); err != nil {
				n.log.Error().Err(err).Msg("ctx bind fail")
			}
		}
		C.bridge_call(retroRun)
		if n.Video.gl.enabled {
			runtime.UnlockOSThread()
		}
	}
}

func (n *Nanoarch) NotifyFrameTime(usec int64) {
	if n.frameTimeCb != nil {
		C.call_frame_time_cb(n.frameTimeCb, C.retro_usec_t(usec))
	}
}

func (n *Nanoarch) IsSupported() error                  { return graphics.RGFWTryInit() }
func (n *Nanoarch) IsGL() bool                          { return n.Video.gl.enabled }
func (n *Nanoarch) IsStopped() bool                     { return n.Stopped.Load() }
func (n *Nanoarch) InputRetropad(port int, data []byte) { n.retropad.SetInput(port, data) }
func (n *Nanoarch) InputKeyboard(_ int, data []byte) {
	if n.keyboardCb == nil {
		return
	}

	// we should preserve the state of pressed buttons for the input poll function (each retro_run)
	// and explicitly call the retro_keyboard_callback function when a keyboard event happens
	pressed, key, mod := n.keyboard.SetKey(data)
	C.bridge_retro_keyboard_callback(unsafe.Pointer(n.keyboardCb), C.bool(pressed),
		C.unsigned(key), C.uint32_t(0), C.uint16_t(mod))
}
func (n *Nanoarch) InputMouse(_ int, data []byte) {
	if len(data) == 0 {
		return
	}

	t := data[0]
	state := data[1:]
	switch t {
	case MouseMove:
		n.mouse.ShiftPos(state)
	case MouseButton:
		n.mouse.SetButtons(state[0])
	}
}

func videoSetPixelFormat(format uint32) (C.bool, error) {
	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		Nan0.Video.PixFmt = RGBA5551
		if err := graphics.SetPixelFormat(graphics.UnsignedShort5551); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		Nan0.Video.PixFmt = RGBA8888Rev
		if err := graphics.SetPixelFormat(graphics.UnsignedInt8888Rev); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
	case C.RETRO_PIXEL_FORMAT_RGB565:
		Nan0.Video.PixFmt = RGB565
		if err := graphics.SetPixelFormat(graphics.UnsignedShort565); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
	default:
		return false, fmt.Errorf("unknown pixel type %v", format)
	}
	Nan0.log.Info().Msgf("Pixel format: %v", Nan0.Video.PixFmt)

	return true, nil
}

func setRotation(rot uint) {
	Nan0.Rot = rot
	Nan0.log.Debug().Msgf("Image rotated %v°", rot)
}

func printOpenGLDriverInfo() {
	var openGLInfo strings.Builder
	openGLInfo.Grow(128)
	version, vendor, renderrer, glsl := graphics.GLInfo()
	fmt.Fprintf(&openGLInfo, "\n[OpenGL] Version: %v\n", version)
	fmt.Fprintf(&openGLInfo, "[OpenGL] Vendor: %v\n", vendor)
	fmt.Fprintf(&openGLInfo, "[OpenGL] Renderer: %v\n", renderrer)
	fmt.Fprintf(&openGLInfo, "[OpenGL] GLSL Version: %v", glsl)
	Nan0.log.Debug().Msg(openGLInfo.String())
}

// State defines any memory state of the emulator
type State []byte

type mem struct {
	ptr  unsafe.Pointer
	size uint
}

const (
	CallSerialize   = 1
	CallUnserialize = 2
)

// SaveState returns emulator internal state.
func SaveState() (State, error) {
	size := C.bridge_retro_serialize_size(retroSerializeSize)
	data := make([]byte, uint(size))
	rez := false

	if Nan0.LibCo && !Nan0.hackSkipSameThreadSave {
		rez = *(*bool)(C.same_thread_with_args2(retroSerialize, C.int(CallSerialize), unsafe.Pointer(&data[0]), unsafe.Pointer(&size)))
	} else {
		rez = bool(C.bridge_retro_serialize(retroSerialize, unsafe.Pointer(&data[0]), size))
	}

	if !rez {
		return nil, errors.New("retro_serialize failed")
	}

	return data, nil
}

// RestoreSaveState restores emulator internal state.
func RestoreSaveState(st State) error {
	if len(st) <= 0 {
		return errors.New("empty load state")
	}

	size := C.size_t(len(st))
	rez := false

	if Nan0.LibCo {
		rez = *(*bool)(C.same_thread_with_args2(retroUnserialize, C.int(CallUnserialize), unsafe.Pointer(&st[0]), unsafe.Pointer(&size)))
	} else {
		rez = bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&st[0]), size))
	}

	if !rez {
		return errors.New("retro_unserialize failed")
	}

	return nil
}

// SaveRAM returns the game save RAM (cartridge) data or a nil slice.
func SaveRAM() State {
	memory := ptSaveRAM()
	if memory == nil {
		return nil
	}
	return C.GoBytes(memory.ptr, C.int(memory.size))
}

// RestoreSaveRAM restores game save RAM.
func RestoreSaveRAM(st State) {
	if len(st) > 0 {
		if memory := ptSaveRAM(); memory != nil {
			//noinspection GoRedundantConversion
			copy(unsafe.Slice((*byte)(memory.ptr), memory.size), st)
		}
	}
}

// memorySize returns memory region size.
func memorySize(id C.uint) uint {
	return uint(C.bridge_retro_get_memory_size(retroGetMemorySize, id))
}

// memoryData returns a pointer to memory data.
func memoryData(id C.uint) unsafe.Pointer {
	return C.bridge_retro_get_memory_data(retroGetMemoryData, id)
}

// ptSaveRam return SRAM memory pointer if core supports it or nil.
func ptSaveRAM() *mem {
	ptr, size := memoryData(C.RETRO_MEMORY_SAVE_RAM), memorySize(C.RETRO_MEMORY_SAVE_RAM)
	if ptr == nil || size == 0 {
		return nil
	}
	return &mem{ptr: ptr, size: size}
}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (m Metadata) HasHack(h string) bool { return slices.Contains(m.Hacks, h) }

var (
	coreLib                      *dlib
	retroAPIVersion              unsafe.Pointer
	retroDeinit                  unsafe.Pointer
	retroGetSystemAVInfo         unsafe.Pointer
	retroGetSystemInfo           unsafe.Pointer
	retroInit                    unsafe.Pointer
	retroLoadGame                unsafe.Pointer
	retroReset                   unsafe.Pointer
	retroRun                     unsafe.Pointer
	retroSetAudioSample          unsafe.Pointer
	retroSetAudioSampleBatch     unsafe.Pointer
	retroSetControllerPortDevice unsafe.Pointer
	retroSetEnvironment          unsafe.Pointer
	retroSetInputPoll            unsafe.Pointer
	retroSetInputState           unsafe.Pointer
	retroSetVideoRefresh         unsafe.Pointer
	retroUnloadGame              unsafe.Pointer
	retroGetMemoryData           unsafe.Pointer
	retroGetMemorySize           unsafe.Pointer
	retroSerialize               unsafe.Pointer
	retroSerializeSize           unsafe.Pointer
	retroUnserialize             unsafe.Pointer
)

//export coreVideoRefresh
func coreVideoRefresh(data unsafe.Pointer, width, height uint, packed uint) {
	if Nan0.Stopped.Load() {
		return
	}

	// some frames can be rendered slower or faster than internal 1/fps core tick
	// so track actual frame render time for proper RTP packet timestamps
	// (and proper frame display time, for example: 1->1/60=16.6ms, 2->10ms, 3->23ms, 4->16.6ms)
	// this is useful only for cores with variable framerate, for the fixed framerate cores this adds stutter
	// !to find docs on Libretro refresh sync and frame times
	dt := Nan0.tickTime
	// Update frame timing only when needed: VFR mode or audio buffer callback.
	if Nan0.vfr || Nan0.audioBuffCb != nil {
		now := time.Now()
		Nan0.lastFrameDelta = float64(now.Sub(Nan0.LastFrameTime))
		Nan0.LastFrameTime = now
	}
	if Nan0.vfr {
		dt = time.Duration(Nan0.lastFrameDelta)
	}

	// when the core returns a duplicate frame
	if data == nil {
		Nan0.Handlers.OnDup()
		return
	}

	// calculate real frame width in pixels from packed data (realWidth >= width)
	// some cores or games output zero pitch, i.e. N64 Mupen
	bpp := Nan0.Video.PixFmt.BPP
	if packed == 0 {
		packed = width * bpp
	}
	// calculate space for the video frame
	bytes := packed * height

	var data_ []byte
	if data != C.RETRO_HW_FRAME_BUFFER_VALID {
		//noinspection GoRedundantConversion
		data_ = unsafe.Slice((*byte)(data), bytes)
	} else {
		// if Libretro renders frame with OpenGL context
		data_ = graphics.ReadFramebuffer(bytes, width, height)
	}

	// some cores or games have a variable output frame size, i.e. PSX Rearmed
	// also we have an option of xN output frame magnification
	// so, it may be rescaled

	Nan0.Handlers.OnVideo(data_, dt, FrameInfo{W: width, H: height, Stride: packed})
}

//export coreAudioSampleBatch
func coreAudioSampleBatch(data unsafe.Pointer, frames C.size_t) C.size_t {
	if Nan0.Stopped.Load() {
		return frames
	}
	Nan0.Handlers.OnAudio(data, int(frames)<<1)

	// Audio buffer status for the core (e.g. mGBA frameskip).
	if Nan0.audioBuffCb != nil {
		rawRate := int(Nan0.sys.av.timing.sample_rate)
		expectedNs := float64(frames) / float64(rawRate) * 1e9
		d := Nan0.lastFrameDelta - expectedNs
		Nan0.audioDrift += d

		occupancy := min(max(50-int(Nan0.audioDrift/(expectedNs*4)*50), 0), 100)

		underrun := false
		if Nan0.audioDrift > expectedNs*4 {
			underrun = true
			Nan0.audioDrift = 0
		}
		if Nan0.audioDrift < -expectedNs*4 {
			Nan0.audioDrift = -expectedNs * 4
		}

		C.call_audio_buffer_status(Nan0.audioBuffCb, true, C.unsigned(occupancy), C.bool(underrun))
	}

	return frames
}

func m(m *C.char) string { return strings.TrimRight(C.GoString(m), "\n") }

//export coreLog
func coreLog(level C.enum_retro_log_level, msg *C.char) {
	switch level {
	// with debug level cores have too much logs
	case C.RETRO_LOG_DEBUG:
		if Nan0.log.GetLevel() <= logger.DebugLevel {
			Nan0.log.Debug().MsgFunc(func() string { return m(msg) })
		}
	case C.RETRO_LOG_INFO:
		if Nan0.log.GetLevel() <= logger.InfoLevel {
			Nan0.log.Info().MsgFunc(func() string { return m(msg) })
		}
	case C.RETRO_LOG_WARN:
		if Nan0.log.GetLevel() <= logger.WarnLevel {
			Nan0.log.Warn().MsgFunc(func() string { return m(msg) })
		}
	case C.RETRO_LOG_ERROR:
		if Nan0.log.GetLevel() <= logger.ErrorLevel {
			Nan0.log.Error().MsgFunc(func() string { return m(msg) })
		}
	default:
		Nan0.log.Log().MsgFunc(func() string { return m(msg) })
		// RETRO_LOG_DUMMY = INT_MAX
	}
}

//export coreGetCurrentFramebuffer
func coreGetCurrentFramebuffer() C.uintptr_t { return (C.uintptr_t)(graphics.GlFbo()) }

//export coreGetProcAddress
func coreGetProcAddress(sym *C.char) C.retro_proc_address_t {
	return (C.retro_proc_address_t)(graphics.RGFWGlProcAddress(C.GoString(sym)))
}

//export coreEnvironment
func coreEnvironment(cmd C.unsigned, data unsafe.Pointer) C.bool {

	// see core_environment_cgo

	switch cmd {
	case C.RETRO_ENVIRONMENT_SET_SYSTEM_AV_INFO:
		Nan0.log.Debug().Msgf("retro_set_system_av_info")
		av := *(*C.struct_retro_system_av_info)(data)
		if isGeometryDifferent(av.geometry) {
			geometryChange(av.geometry)
		}
		return true
	case C.RETRO_ENVIRONMENT_SET_GEOMETRY:
		Nan0.log.Debug().Msgf("retro_set_geometry")
		geom := *(*C.struct_retro_game_geometry)(data)
		if isGeometryDifferent(geom) {
			geometryChange(geom)
		}
		return true
	case C.RETRO_ENVIRONMENT_SET_ROTATION:
		setRotation((*(*uint)(data) % 4) * 90)
		return true
	case C.RETRO_ENVIRONMENT_GET_CAN_DUPE:
		dup := C.bool(Nan0.meta.FrameDup)
		*(*C.bool)(data) = dup
		return dup
	case C.RETRO_ENVIRONMENT_GET_USERNAME:
		*(**C.char)(data) = Nan0.cUserName
		return true
	case C.RETRO_ENVIRONMENT_GET_LOG_INTERFACE:
		cb := (*C.struct_retro_log_callback)(data)
		cb.log = (C.retro_log_printf_t)(C.core_log_cgo)
		return true
	case C.RETRO_ENVIRONMENT_SET_PIXEL_FORMAT:
		res, err := videoSetPixelFormat(*(*C.enum_retro_pixel_format)(data))
		if err != nil {
			Nan0.log.Fatal().Err(err).Msg("pix format failed")
		}
		return res
	case C.RETRO_ENVIRONMENT_GET_SYSTEM_DIRECTORY:
		*(**C.char)(data) = Nan0.cSystemDirectory
		return true
	case C.RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
		*(**C.char)(data) = Nan0.cSaveDirectory
		return true
	case C.RETRO_ENVIRONMENT_SET_MESSAGE:
		// only with the Libretro debug mode
		if Nan0.log.GetLevel() < logger.InfoLevel {
			message := (*C.struct_retro_message)(data)
			msg := C.GoString(message.msg)
			Nan0.log.Debug().Msgf("message: %v", msg)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_GET_VARIABLE:
		if Nan0.options == nil {
			return false
		}
		rv := (*C.struct_retro_variable)(data)
		key := C.GoString(rv.key)
		if v, ok := Nan0.options[key]; ok {
			// make Go strings null-terminated copies ;_;
			Nan0.options[key] = v + "\x00"
			ptr := unsafe.Pointer(unsafe.StringData(Nan0.options[key]))
			var p runtime.Pinner
			p.Pin(ptr)
			defer p.Unpin()
			// cast to C string and set the value
			rv.value = (*C.char)(ptr)
			Nan0.log.Debug().Msgf("Set %v=%v", key, v)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_HW_RENDER:
		if Nan0.Video.gl.enabled {
			Nan0.Video.hw = (*C.struct_retro_hw_render_callback)(data)
			Nan0.Video.hw.get_current_framebuffer = (C.retro_hw_get_current_framebuffer_t)(C.core_get_current_framebuffer_cgo)
			Nan0.Video.hw.get_proc_address = (C.retro_hw_get_proc_address_t)(C.core_get_proc_address_cgo)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_CONTROLLER_INFO:
		if Nan0.log.GetLevel() > logger.DebugLevel {
			return false
		}

		info := (*[64]C.struct_retro_controller_info)(data)
		for c, controller := range info {
			tp := unsafe.Pointer(controller.types)
			if tp == nil {
				break
			}
			cInfo := strings.Builder{}
			fmt.Fprintf(&cInfo, "Controller [%v] ", c)
			cd := (*[32]C.struct_retro_controller_description)(tp)
			delim := ", "
			n := int(controller.num_types)
			for i := range n {
				if i == n-1 {
					delim = ""
				}
				fmt.Fprintf(&cInfo, "%v: %v%s", cd[i].id, C.GoString(cd[i].desc), delim)
			}
			//Nan0.log.Debug().Msgf("%v", cInfo.String())
		}
		return true
	case C.RETRO_ENVIRONMENT_SET_KEYBOARD_CALLBACK:
		Nan0.log.Debug().Msgf("Keyboard event callback was set")
		Nan0.keyboardCb = (*C.struct_retro_keyboard_callback)(data)
		return true
	case C.RETRO_ENVIRONMENT_SET_AUDIO_BUFFER_STATUS_CALLBACK:
		if data == nil {
			Nan0.audioBuffCb = nil
			return true
		}

		src := (*C.struct_retro_audio_buffer_status_callback)(data)
		Nan0.audioBuffCb = new(C.struct_retro_audio_buffer_status_callback)
		*Nan0.audioBuffCb = *src
		Nan0.log.Info().Msg("Audio buffer status callback registered")
		return true
	case C.RETRO_ENVIRONMENT_SET_MINIMUM_AUDIO_LATENCY:
		latency := *(*C.unsigned)(data)
		Nan0.log.Info().Uint("latency_ms", uint(latency)).Msg("audio latency requested")
		return true
	case C.RETRO_ENVIRONMENT_SET_FRAME_TIME_CALLBACK:
		Nan0.frameTimeCb = (*C.struct_retro_frame_time_callback)(data)
		Nan0.log.Info().Msg("Frame time callback registered")
		return true
	}
	return false
}

//export initVideo
func initVideo() {
	var context graphics.Context
	switch Nan0.Video.hw.context_type {
	case C.RETRO_HW_CONTEXT_NONE:
		context = graphics.CtxNone
	case C.RETRO_HW_CONTEXT_OPENGL:
		context = graphics.CtxOpenGl
	case C.RETRO_HW_CONTEXT_OPENGLES2:
		context = graphics.CtxOpenGlEs2
	case C.RETRO_HW_CONTEXT_OPENGL_CORE:
		context = graphics.CtxOpenGlCore
	case C.RETRO_HW_CONTEXT_OPENGLES3:
		context = graphics.CtxOpenGlEs3
	case C.RETRO_HW_CONTEXT_OPENGLES_VERSION:
		context = graphics.CtxOpenGlEsVersion
	case C.RETRO_HW_CONTEXT_VULKAN:
		context = graphics.CtxVulkan
	case C.RETRO_HW_CONTEXT_DUMMY:
		context = graphics.CtxDummy
	default:
		context = graphics.CtxUnknown
	}

	thread.Main(func() {
		var err error
		Nan0.glCtx, err = graphics.NewRGFWContext(graphics.Config{
			Ctx:            context,
			W:              int(Nan0.sys.av.geometry.max_width),
			H:              int(Nan0.sys.av.geometry.max_height),
			GLAutoContext:  Nan0.Video.gl.autoCtx,
			GLVersionMajor: uint(Nan0.Video.hw.version_major),
			GLVersionMinor: uint(Nan0.Video.hw.version_minor),
			GLHasDepth:     bool(Nan0.Video.hw.depth),
			GLHasStencil:   bool(Nan0.Video.hw.stencil),
		})
		if err != nil {
			panic(err)
		}
	})

	if Nan0.log.GetLevel() < logger.InfoLevel {
		printOpenGLDriverInfo()
	}
}

//export deinitVideo
func deinitVideo() {
	if !Nan0.hackSkipHwContextDestroy {
		C.bridge_context_reset(Nan0.Video.hw.context_destroy)
	}
	thread.Main(func() {
		if err := Nan0.glCtx.Deinit(); err != nil {
			Nan0.log.Error().Err(err).Msg("deinit fail")
		}
	})
	Nan0.Video.gl.enabled = false
	Nan0.Video.gl.autoCtx = false
	Nan0.hackSkipHwContextDestroy = false
	Nan0.hackSkipSameThreadSave = false
	thread.SwitchGraphics(false)
}

type limit struct {
	d  time.Duration
	t  *time.Timer
	mu sync.Mutex
}

func NewLimit(d time.Duration) func(f func()) {
	l := &limit{d: d}
	return func(f func()) { l.push(f) }
}

func (d *limit) push(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.t != nil {
		d.t.Stop()
	}
	d.t = time.AfterFunc(d.d, f)
}

func geometryChange(geom C.struct_retro_game_geometry) {
	Nan0.limiter(func() {
		old := Nan0.sys.av.geometry
		Nan0.sys.av.geometry = geom

		if Nan0.Video.gl.enabled && (old.max_width != geom.max_width || old.max_height != geom.max_height) {
			// (for LRPS2) makes the max height bigger increasing SDL2 and OpenGL buffers slightly
			Nan0.sys.av.geometry.max_height = C.unsigned(float32(Nan0.sys.av.geometry.max_height) * 1.5)
			bufS := uint(geom.max_width*Nan0.sys.av.geometry.max_height) * Nan0.Video.PixFmt.BPP
			graphics.SetBuffer(int(bufS))
			Nan0.log.Debug().Msgf("OpenGL frame buffer: %v", bufS)
		}

		if Nan0.OnSystemAvInfo != nil {
			Nan0.log.Debug().Msgf(">>> geometry change %v -> %v", old, geom)
			go Nan0.OnSystemAvInfo()
		}
	})
}

func isGeometryDifferent(geom C.struct_retro_game_geometry) bool {
	return Nan0.sys.av.geometry.base_width != geom.base_width ||
		Nan0.sys.av.geometry.base_height != geom.base_height
}
