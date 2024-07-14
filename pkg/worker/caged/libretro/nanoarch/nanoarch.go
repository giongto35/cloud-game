package nanoarch

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/graphics"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/repo/arch"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
)

/*
#include "libretro.h"
#include "nanoarch.h"
#include <stdlib.h>

#define RETRO_ENVIRONMENT_GET_CLEAR_ALL_THREAD_WAITS_CB (3 | 0x800000)
*/
import "C"

const lastKey = int(C.RETRO_DEVICE_ID_JOYPAD_R3)

const KeyPressed = 1
const KeyReleased = 0

const MaxPort int = 4

var (
	RGBA5551    = PixFmt{C: 0, BPP: 2} // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	RGBA8888Rev = PixFmt{C: 1, BPP: 4} // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	RGB565      = PixFmt{C: 2, BPP: 2} // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

type Nanoarch struct {
	Handlers
	LastFrameTime int64
	LibCo         bool
	meta          Metadata
	options       map[string]string
	options4rom   map[string]map[string]string
	reserved      chan struct{} // limits concurrent use
	Rot           uint
	serializeSize C.size_t
	Stopped       atomic.Bool
	sys           struct {
		av C.struct_retro_system_av_info
		i  C.struct_retro_system_info
	}
	tickTime         int64
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
	sdlCtx                   *graphics.SDL
	hackSkipHwContextDestroy bool
	limiter                  func(func())
	log                      *logger.Logger
}

type Handlers struct {
	OnDpad         func(port uint, axis uint) (shift int16)
	OnKeyPress     func(port uint, key int) int
	OnAudio        func(ptr unsafe.Pointer, frames int)
	OnVideo        func(data []byte, delta int32, fi FrameInfo)
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
		OnDpad:     func(uint, uint) int16 { return 0 },
		OnKeyPress: func(uint, int) int { return 0 },
		OnAudio:    func(unsafe.Pointer, int) {},
		OnVideo:    func([]byte, int32, FrameInfo) {},
		OnDup:      func() {},
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
func (n *Nanoarch) VideoFramerate() int              { return int(n.sys.av.timing.fps) }
func (n *Nanoarch) IsPortrait() bool                 { return 90 == n.Rot%180 }
func (n *Nanoarch) BaseWidth() int                   { return int(n.sys.av.geometry.base_width) }
func (n *Nanoarch) BaseHeight() int                  { return int(n.sys.av.geometry.base_height) }
func (n *Nanoarch) WaitReady()                       { <-n.reserved }
func (n *Nanoarch) Close()                           { n.Stopped.Store(true); n.reserved <- struct{}{} }
func (n *Nanoarch) SetLogger(log *logger.Logger)     { n.log = log }
func (n *Nanoarch) SetVideoDebounce(t time.Duration) { n.limiter = NewLimit(t) }

func (n *Nanoarch) CoreLoad(meta Metadata) {
	var err error
	n.meta = meta
	n.LibCo = meta.UsesLibCo
	n.vfr = meta.HasVFR
	n.Aspect = meta.CoreAspectRatio
	n.Video.gl.autoCtx = meta.AutoGlContext
	n.Video.gl.enabled = meta.IsGlAllowed

	thread.SwitchGraphics(n.Video.gl.enabled)

	// hacks
	Nan0.hackSkipHwContextDestroy = meta.HasHack("skip_hw_context_destroy")

	n.options = maps.Clone(meta.Options)
	n.options4rom = meta.Options4rom

	filePath := meta.LibPath
	if ar, err := arch.Guess(); err == nil {
		filePath = filePath + ar.LibExt
	} else {
		n.log.Warn().Err(err).Msg("system arch guesser failed")
	}

	coreLib, err = loadLib(filePath)
	// fallback to sequential lib loader (first successfully loaded)
	if err != nil {
		n.log.Error().Err(err).Msgf("load fail: %v", filePath)
		coreLib, err = loadLibRollingRollingRolling(filePath)
		if err != nil {
			n.log.Fatal().Err(err).Msgf("core load: %s", filePath)
		}
	}

	retroInit = loadFunction(coreLib, "retro_init")
	retroDeinit = loadFunction(coreLib, "retro_deinit")
	//retroAPIVersion = loadFunction(coreLib, "retro_api_version")
	retroGetSystemInfo = loadFunction(coreLib, "retro_get_system_info")
	retroGetSystemAVInfo = loadFunction(coreLib, "retro_get_system_av_info")
	retroSetEnvironment = loadFunction(coreLib, "retro_set_environment")
	retroSetVideoRefresh = loadFunction(coreLib, "retro_set_video_refresh")
	retroSetInputPoll = loadFunction(coreLib, "retro_set_input_poll")
	retroSetInputState = loadFunction(coreLib, "retro_set_input_state")
	retroSetAudioSample = loadFunction(coreLib, "retro_set_audio_sample")
	retroSetAudioSampleBatch = loadFunction(coreLib, "retro_set_audio_sample_batch")
	retroRun = loadFunction(coreLib, "retro_run")
	retroLoadGame = loadFunction(coreLib, "retro_load_game")
	retroUnloadGame = loadFunction(coreLib, "retro_unload_game")
	retroSerializeSize = loadFunction(coreLib, "retro_serialize_size")
	retroSerialize = loadFunction(coreLib, "retro_serialize")
	retroUnserialize = loadFunction(coreLib, "retro_unserialize")
	retroSetControllerPortDevice = loadFunction(coreLib, "retro_set_controller_port_device")
	retroGetMemorySize = loadFunction(coreLib, "retro_get_memory_size")
	retroGetMemoryData = loadFunction(coreLib, "retro_get_memory_data")

	C.bridge_retro_set_environment(retroSetEnvironment, C.core_environment_cgo)
	C.bridge_retro_set_video_refresh(retroSetVideoRefresh, C.core_video_refresh_cgo)
	C.bridge_retro_set_input_poll(retroSetInputPoll, C.core_input_poll_cgo)
	C.bridge_retro_set_input_state(retroSetInputState, C.core_input_state_cgo)
	C.bridge_retro_set_audio_sample(retroSetAudioSample, C.core_audio_sample_cgo)
	C.bridge_retro_set_audio_sample_batch(retroSetAudioSampleBatch, C.core_audio_sample_batch_cgo)

	if n.LibCo {
		C.same_thread(retroInit)
	} else {
		C.bridge_retro_init(retroInit)
	}

	C.bridge_retro_get_system_info(retroGetSystemInfo, &n.sys.i)
	n.log.Debug().Msgf("System >>> %v (%v) [%v] nfp: %v",
		C.GoString(n.sys.i.library_name), C.GoString(n.sys.i.library_version),
		C.GoString(n.sys.i.valid_extensions), bool(n.sys.i.need_fullpath))
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

	Nan0.tickTime = int64(time.Second / time.Duration(n.sys.av.timing.fps))
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
	for i := range MaxPort {
		C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(i), C.RETRO_DEVICE_JOYPAD)
	}

	// map custom devices to ports
	for k, v := range n.meta.Hid {
		for _, device := range v {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(k), C.unsigned(device))
			n.log.Debug().Msgf("set custom port-device: %v:%v", k, device)
		}
	}

	n.LastFrameTime = time.Now().UnixNano()

	return nil
}

func (n *Nanoarch) Shutdown() {
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
				if err := n.sdlCtx.BindContext(); err != nil {
					n.log.Error().Err(err).Msg("ctx switch fail")
				}
			})
		}
		C.bridge_retro_unload_game(retroUnloadGame)
		C.bridge_retro_deinit(retroDeinit)
		if n.Video.gl.enabled {
			thread.Main(func() {
				deinitVideo()
				runtime.UnlockOSThread()
			})
		}
	}

	setRotation(0)
	Nan0.sys.av = C.struct_retro_system_av_info{}
	if err := closeLib(coreLib); err != nil {
		n.log.Error().Err(err).Msg("lib close failed")
	}
	n.options = nil
	n.options4rom = nil
	C.free(unsafe.Pointer(n.cUserName))
	C.free(unsafe.Pointer(n.cSaveDirectory))
	C.free(unsafe.Pointer(n.cSystemDirectory))
}

func (n *Nanoarch) Run() {
	if n.LibCo {
		C.same_thread(retroRun)
	} else {
		if n.Video.gl.enabled {
			// running inside a go routine, lock the thread to make sure the OpenGL context stays current
			runtime.LockOSThread()
			if err := n.sdlCtx.BindContext(); err != nil {
				n.log.Error().Err(err).Msg("ctx bind fail")
			}
		}
		C.bridge_retro_run(retroRun)
		if n.Video.gl.enabled {
			runtime.UnlockOSThread()
		}
	}
}

func (n *Nanoarch) IsGL() bool      { return n.Video.gl.enabled }
func (n *Nanoarch) IsStopped() bool { return n.Stopped.Load() }

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
	openGLInfo.WriteString(fmt.Sprintf("\n[OpenGL] Version: %v\n", graphics.GetGLVersionInfo()))
	openGLInfo.WriteString(fmt.Sprintf("[OpenGL] Vendor: %v\n", graphics.GetGLVendorInfo()))
	openGLInfo.WriteString(fmt.Sprintf("[OpenGL] Renderer: %v\n", graphics.GetGLRendererInfo()))
	openGLInfo.WriteString(fmt.Sprintf("[OpenGL] GLSL Version: %v", graphics.GetGLSLInfo()))
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
	data := make([]byte, uint(Nan0.serializeSize))
	rez := false
	if Nan0.LibCo {
		rez = *(*bool)(C.same_thread_with_args2(retroSerialize, C.int(CallSerialize), unsafe.Pointer(&data[0]), unsafe.Pointer(&Nan0.serializeSize)))
	} else {
		rez = bool(C.bridge_retro_serialize(retroSerialize, unsafe.Pointer(&data[0]), Nan0.serializeSize))
	}
	if !rez {
		return nil, errors.New("retro_serialize failed")
	}
	return data, nil
}

// RestoreSaveState restores emulator internal state.
func RestoreSaveState(st State) error {
	if len(st) > 0 {
		rez := false
		if Nan0.LibCo {
			rez = *(*bool)(C.same_thread_with_args2(retroUnserialize, C.int(CallUnserialize), unsafe.Pointer(&st[0]), unsafe.Pointer(&Nan0.serializeSize)))
		} else {
			rez = bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&st[0]), Nan0.serializeSize))
		}
		if !rez {
			return errors.New("retro_unserialize failed")
		}
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

// getMemorySize returns memory region size.
func getMemorySize(id C.uint) uint {
	return uint(C.bridge_retro_get_memory_size(retroGetMemorySize, id))
}

// getMemoryData returns a pointer to memory data.
func getMemoryData(id C.uint) unsafe.Pointer {
	return C.bridge_retro_get_memory_data(retroGetMemoryData, id)
}

// ptSaveRam return SRAM memory pointer if core supports it or nil.
func ptSaveRAM() *mem {
	ptr, size := getMemoryData(C.RETRO_MEMORY_SAVE_RAM), getMemorySize(C.RETRO_MEMORY_SAVE_RAM)
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

func (m Metadata) HasHack(h string) bool {
	for _, n := range m.Hacks {
		if h == n {
			return true
		}
	}
	return false
}

var (
	//retroAPIVersion              unsafe.Pointer
	retroDeinit                  unsafe.Pointer
	retroGetSystemAVInfo         unsafe.Pointer
	retroGetSystemInfo           unsafe.Pointer
	coreLib                      unsafe.Pointer
	retroInit                    unsafe.Pointer
	retroLoadGame                unsafe.Pointer
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
		Nan0.log.Warn().Msgf(">>> skip video")
		return
	}

	// some frames can be rendered slower or faster than internal 1/fps core tick
	// so track actual frame render time for proper RTP packet timestamps
	// (and proper frame display time, for example: 1->1/60=16.6ms, 2->10ms, 3->23ms, 4->16.6ms)
	// this is useful only for cores with variable framerate, for the fixed framerate cores this adds stutter
	// !to find docs on Libretro refresh sync and frame times
	t := time.Now().UnixNano()
	dt := Nan0.tickTime
	// override frame rendering with dynamic frame times
	if Nan0.vfr {
		dt = t - Nan0.LastFrameTime
	}
	Nan0.LastFrameTime = t

	// when the core returns a duplicate frame
	if data == nil {
		Nan0.Handlers.OnDup()
		return
	}

	// calculate real frame width in pixels from packed data (realWidth >= width)
	// some cores or games output zero pitch, i.e. N64 Mupen
	if packed == 0 {
		packed = width * Nan0.Video.PixFmt.BPP
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

	Nan0.Handlers.OnVideo(data_, int32(dt), FrameInfo{W: width, H: height, Stride: packed})
}

//export coreInputPoll
func coreInputPoll() {}

//export coreInputState
func coreInputState(port C.unsigned, device C.unsigned, index C.unsigned, id C.unsigned) C.int16_t {
	if uint(port) >= uint(MaxPort) {
		return KeyReleased
	}

	if device == C.RETRO_DEVICE_ANALOG {
		if index > C.RETRO_DEVICE_INDEX_ANALOG_RIGHT || id > C.RETRO_DEVICE_ID_ANALOG_Y {
			return 0
		}
		axis := index*2 + id
		value := Nan0.Handlers.OnDpad(uint(port), uint(axis))
		if value != 0 {
			return (C.int16_t)(value)
		}
	}

	key := int(id)
	if key > lastKey || index > 0 || device != C.RETRO_DEVICE_JOYPAD {
		return KeyReleased
	}
	if Nan0.Handlers.OnKeyPress(uint(port), key) == KeyPressed {
		return KeyPressed
	}
	return KeyReleased
}

//export coreAudioSample
func coreAudioSample(l, r C.int16_t) {
	frame := []C.int16_t{l, r}
	coreAudioSampleBatch(unsafe.Pointer(&frame), 1)
}

//export coreAudioSampleBatch
func coreAudioSampleBatch(data unsafe.Pointer, frames C.size_t) C.size_t {
	if Nan0.Stopped.Load() {
		if Nan0.log.GetLevel() < logger.InfoLevel {
			Nan0.log.Warn().Msgf(">>> skip %v audio frames", frames)
		}
		return frames
	}
	Nan0.Handlers.OnAudio(data, int(frames)<<1)
	return frames
}

func m(m *C.char) string { return strings.TrimRight(C.GoString(m), "\n") }

//export coreLog
func coreLog(level C.enum_retro_log_level, msg *C.char) {
	switch level {
	// with debug level cores have too much logs
	case C.RETRO_LOG_DEBUG:
		Nan0.log.Debug().MsgFunc(func() string { return m(msg) })
	case C.RETRO_LOG_INFO:
		Nan0.log.Info().MsgFunc(func() string { return m(msg) })
	case C.RETRO_LOG_WARN:
		Nan0.log.Warn().MsgFunc(func() string { return m(msg) })
	case C.RETRO_LOG_ERROR:
		Nan0.log.Error().MsgFunc(func() string { return m(msg) })
	default:
		Nan0.log.Log().MsgFunc(func() string { return m(msg) })
		// RETRO_LOG_DUMMY = INT_MAX
	}
}

//export coreGetCurrentFramebuffer
func coreGetCurrentFramebuffer() C.uintptr_t { return (C.uintptr_t)(graphics.GetGlFbo()) }

//export coreGetProcAddress
func coreGetProcAddress(sym *C.char) C.retro_proc_address_t {
	return (C.retro_proc_address_t)(graphics.GetGlProcAddress(C.GoString(sym)))
}

//export coreEnvironment
func coreEnvironment(cmd C.unsigned, data unsafe.Pointer) C.bool {
	// spammy
	switch cmd {
	case C.RETRO_ENVIRONMENT_GET_VARIABLE_UPDATE:
		return false
	case C.RETRO_ENVIRONMENT_GET_AUDIO_VIDEO_ENABLE:
		return false
	}

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
	case C.RETRO_ENVIRONMENT_SHUTDOWN:
		//window.SetShouldClose(true)
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
			cInfo.WriteString(fmt.Sprintf("Controller [%v] ", c))
			cd := (*[32]C.struct_retro_controller_description)(tp)
			delim := ", "
			n := int(controller.num_types)
			for i := range n {
				if i == n-1 {
					delim = ""
				}
				cInfo.WriteString(fmt.Sprintf("%v: %v%s", cd[i].id, C.GoString(cd[i].desc), delim))
			}
			Nan0.log.Debug().Msgf("%v", cInfo.String())
		}
		return true
	case C.RETRO_ENVIRONMENT_GET_CLEAR_ALL_THREAD_WAITS_CB:
		C.bridge_clear_all_thread_waits_cb(data)
		return true
	case C.RETRO_ENVIRONMENT_GET_SAVESTATE_CONTEXT:
		if ctx := (*C.int)(data); ctx != nil {
			*ctx = C.RETRO_SAVESTATE_CONTEXT_NORMAL
		}
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

	sdl, err := graphics.NewSDLContext(graphics.Config{
		Ctx:            context,
		W:              int(Nan0.sys.av.geometry.max_width),
		H:              int(Nan0.sys.av.geometry.max_height),
		GLAutoContext:  Nan0.Video.gl.autoCtx,
		GLVersionMajor: uint(Nan0.Video.hw.version_major),
		GLVersionMinor: uint(Nan0.Video.hw.version_minor),
		GLHasDepth:     bool(Nan0.Video.hw.depth),
		GLHasStencil:   bool(Nan0.Video.hw.stencil),
	}, Nan0.log)
	if err != nil {
		panic(err)
	}
	Nan0.sdlCtx = sdl

	if Nan0.log.GetLevel() < logger.InfoLevel {
		printOpenGLDriverInfo()
	}
}

//export deinitVideo
func deinitVideo() {
	if !Nan0.hackSkipHwContextDestroy {
		C.bridge_context_reset(Nan0.Video.hw.context_destroy)
	}
	if err := Nan0.sdlCtx.Deinit(); err != nil {
		Nan0.log.Error().Err(err).Msg("deinit fail")
	}
	Nan0.Video.gl.enabled = false
	Nan0.Video.gl.autoCtx = false
	Nan0.hackSkipHwContextDestroy = false
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
			bufS := uint(geom.max_width*geom.max_height) * Nan0.Video.PixFmt.BPP
			graphics.SetBuffer(int(bufS))
			Nan0.log.Debug().Msgf("OpenGL frame buffer: %v", byteCountBinary(int64(bufS)))
		}

		if Nan0.OnSystemAvInfo != nil {
			Nan0.log.Debug().Msgf(">>> geometry change %v -> %v", old, geom)
			if Nan0.Aspect {
				go Nan0.OnSystemAvInfo()
			}
		}
	})
}

func isGeometryDifferent(geom C.struct_retro_game_geometry) bool {
	return Nan0.sys.av.geometry.base_width != geom.base_width ||
		Nan0.sys.av.geometry.base_height != geom.base_height
}
