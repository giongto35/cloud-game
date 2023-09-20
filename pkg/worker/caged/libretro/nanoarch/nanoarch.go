package nanoarch

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/graphics"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/image"
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

type Nanoarch struct {
	Handlers
	LastFrameTime int64
	LibCo         bool
	multitap      struct {
		supported bool
		enabled   bool
		value     C.unsigned
	}
	options          *map[string]string
	reserved         chan struct{} // limits concurrent use
	Rot              *image.Rotate
	serializeSize    C.size_t
	stopped          atomic.Bool
	sysAvInfo        C.struct_retro_system_av_info
	sysInfo          C.struct_retro_system_info
	tickTime         int64
	cSaveDirectory   *C.char
	cSystemDirectory *C.char
	cUserName        *C.char
	Video            struct {
		gl struct {
			enabled bool
			autoCtx bool
		}
		BPP    uint
		hw     *C.struct_retro_hw_render_callback
		PixFmt uint32
	}
	vfr                      bool
	sdlCtx                   *graphics.SDL
	hackSkipHwContextDestroy bool
	log                      *logger.Logger
}

type Handlers struct {
	OnDpad     func(port uint, axis uint) (shift int16)
	OnKeyPress func(port uint, key int) int
	OnAudio    func(ptr unsafe.Pointer, frames int)
	OnVideo    func(data []byte, delta int32, fi FrameInfo)
}

type FrameInfo struct {
	W      uint
	H      uint
	Packed uint
}

type Metadata struct {
	LibPath         string // the full path to some emulator lib
	AudioSampleRate int
	Fps             float64
	BaseWidth       int
	BaseHeight      int
	Rotation        image.Rotate
	IsGlAllowed     bool
	UsesLibCo       bool
	AutoGlContext   bool
	HasMultitap     bool
	HasVFR          bool
	Options         map[string]string
	Hacks           []string
}

// Nan0 is a global link for C callbacks to Go
var Nan0 = Nanoarch{
	reserved: make(chan struct{}, 1), // this thing forbids concurrent use of the emulator
	stopped:  atomic.Bool{},
	Handlers: Handlers{
		OnDpad:     func(uint, uint) int16 { return 0 },
		OnKeyPress: func(uint, int) int { return 0 },
		OnAudio:    func(unsafe.Pointer, int) {},
		OnVideo:    func([]byte, int32, FrameInfo) {},
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

func (n *Nanoarch) AudioSampleRate() int { return int(n.sysAvInfo.timing.sample_rate) }
func (n *Nanoarch) VideoFramerate() int  { return int(n.sysAvInfo.timing.fps) }
func (n *Nanoarch) IsPortrait() bool     { return n.Rot != nil && n.Rot.IsEven }
func (n *Nanoarch) GeometryBase() (int, int) {
	return int(n.sysAvInfo.geometry.base_width), int(n.sysAvInfo.geometry.base_height)
}
func (n *Nanoarch) GeometryMax() (int, int) {
	return int(n.sysAvInfo.geometry.max_width), int(n.sysAvInfo.geometry.max_height)
}
func (n *Nanoarch) WaitReady()                   { <-n.reserved }
func (n *Nanoarch) Close()                       { n.stopped.Store(true); n.reserved <- struct{}{} }
func (n *Nanoarch) SetLogger(log *logger.Logger) { n.log = log }

func (n *Nanoarch) CoreLoad(meta Metadata) {
	var err error
	n.LibCo = meta.UsesLibCo
	n.vfr = meta.HasVFR
	n.Video.gl.autoCtx = meta.AutoGlContext
	n.Video.gl.enabled = meta.IsGlAllowed

	// hacks
	Nan0.hackSkipHwContextDestroy = meta.HasHack("skip_hw_context_destroy")

	n.options = &meta.Options

	n.multitap.supported = meta.HasMultitap
	n.multitap.enabled = false
	n.multitap.value = 0

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

	C.bridge_retro_get_system_info(retroGetSystemInfo, &n.sysInfo)
	n.log.Debug().Msgf("System >>> %s (%s) [%s] nfp: %v",
		C.GoString(n.sysInfo.library_name), C.GoString(n.sysInfo.library_version),
		C.GoString(n.sysInfo.valid_extensions), bool(n.sysInfo.need_fullpath))
}

func (n *Nanoarch) LoadGame(path string) error {
	game := C.struct_retro_game_info{}

	big := bool(n.sysInfo.need_fullpath) // big ROMs are loaded by cores later
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

	if ok := C.bridge_retro_load_game(retroLoadGame, &game); !ok {
		return fmt.Errorf("core failed to load ROM: %v", path)
	}

	C.bridge_retro_get_system_av_info(retroGetSystemAVInfo, &n.sysAvInfo)
	n.log.Info().Msgf("System A/V >>> %vx%v (%vx%v), [%vfps], AR [%v], audio [%vHz]",
		n.sysAvInfo.geometry.base_width, n.sysAvInfo.geometry.base_height,
		n.sysAvInfo.geometry.max_width, n.sysAvInfo.geometry.max_height,
		n.sysAvInfo.timing.fps, n.sysAvInfo.geometry.aspect_ratio, n.sysAvInfo.timing.sample_rate,
	)

	n.serializeSize = C.bridge_retro_serialize_size(retroSerializeSize)
	n.log.Info().Msgf("Save file size: %v", byteCountBinary(int64(n.serializeSize)))

	Nan0.tickTime = int64(time.Second / time.Duration(n.sysAvInfo.timing.fps))
	if n.vfr {
		n.log.Info().Msgf("variable framerate (VFR) is enabled")
	}

	n.stopped.Store(false)

	if n.Video.gl.enabled {
		// flip Y coordinates of OpenGL
		setRotation(uint(image.Flip180))
		bufS := uint(n.sysAvInfo.geometry.max_width*n.sysAvInfo.geometry.max_height) * n.Video.BPP
		graphics.SetBuffer(int(bufS))
		n.log.Info().Msgf("Set buffer: %v", byteCountBinary(int64(bufS)))
		if n.LibCo {
			C.same_thread(C.init_video_cgo)
		} else {
			runtime.LockOSThread()
			initVideo()
			runtime.UnlockOSThread()
		}
	}

	// set default controller types on all ports
	for i := 0; i < MaxPort; i++ {
		C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(i), C.RETRO_DEVICE_JOYPAD)
	}

	n.LastFrameTime = time.Now().UnixNano()

	return nil
}

// ToggleMultitap toggles multitap controller for cores.
//
// Official SNES games only support a single multitap device
// Most require it to be plugged in player 2 port and Snes9X requires it
// to be "plugged" after the game is loaded.
// Control this from the browser since player 2 will stop working in some games
// if multitap is "plugged" in.
func (n *Nanoarch) ToggleMultitap() {
	if !n.multitap.supported || n.multitap.value == 0 {
		return
	}
	mt := n.multitap.value
	if n.multitap.enabled {
		mt = C.RETRO_DEVICE_JOYPAD
	}
	C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, mt)
	n.multitap.enabled = !n.multitap.enabled
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
	if err := closeLib(coreLib); err != nil {
		n.log.Error().Err(err).Msg("lib close failed")
	}
	n.options = nil
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

func (n *Nanoarch) IsStopped() bool { return n.stopped.Load() }

func videoSetPixelFormat(format uint32) (C.bool, error) {
	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		Nan0.Video.PixFmt = image.BitFormatShort5551
		if err := graphics.SetPixelFormat(graphics.UnsignedShort5551); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
		Nan0.Video.BPP = 2
		// format is not implemented
		return false, fmt.Errorf("unsupported pixel type %v converter", format)
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		Nan0.Video.PixFmt = image.BitFormatInt8888Rev
		if err := graphics.SetPixelFormat(graphics.UnsignedInt8888Rev); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
		Nan0.Video.BPP = 4
	case C.RETRO_PIXEL_FORMAT_RGB565:
		Nan0.Video.PixFmt = image.BitFormatShort565
		if err := graphics.SetPixelFormat(graphics.UnsignedShort565); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", Nan0.Video.PixFmt)
		}
		Nan0.Video.BPP = 2
	default:
		return false, fmt.Errorf("unknown pixel type %v", format)
	}
	return true, nil
}

func setRotation(rotation uint) {
	if Nan0.Rot != nil && rotation == uint(Nan0.Rot.Angle) {
		return
	}
	if rotation > 0 {
		r := image.GetRotation(image.Angle(rotation))
		Nan0.Rot = &r
	} else {
		Nan0.Rot = nil
	}
	Nan0.log.Debug().Msgf("Image rotated %v°", map[uint]uint{0: 0, 1: 90, 2: 180, 3: 270}[rotation])
}

func printOpenGLDriverInfo() {
	var openGLInfo strings.Builder
	openGLInfo.Grow(128)
	openGLInfo.WriteString(fmt.Sprintf("\n[OpenGL] Version: %v\n", graphics.GetGLVersionInfo()))
	openGLInfo.WriteString(fmt.Sprintf("[OpenGL] Vendor: %v\n", graphics.GetGLVendorInfo()))
	// This string is often the name of the GPU.
	// In the case of Mesa3d, it would be i.e "Gallium 0.4 on NVA8".
	// It might even say "Direct3D" if the Windows Direct3D wrapper is being used.
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
	if Nan0.stopped.Load() {
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

	// some cores can return nothing
	// !to add duplicate if can dup
	if data == nil {
		return
	}

	// calculate real frame width in pixels from packed data (realWidth >= width)
	// some cores or games output zero pitch, i.e. N64 Mupen
	if packed == 0 {
		packed = width * Nan0.Video.BPP
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

	Nan0.Handlers.OnVideo(data_, int32(dt), FrameInfo{W: width, H: height, Packed: packed})
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
	if Nan0.stopped.Load() {
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
	case C.RETRO_ENVIRONMENT_SET_ROTATION:
		setRotation(*(*uint)(data) % 4)
		return true
	case C.RETRO_ENVIRONMENT_GET_CAN_DUPE:
		*(*C.bool)(data) = C.bool(true)
		return true
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
		if (*Nan0.options) == nil {
			return false
		}
		rv := (*C.struct_retro_variable)(data)
		key := C.GoString(rv.key)
		if v, ok := (*Nan0.options)[key]; ok {
			// make Go strings null-terminated copies ;_;
			(*Nan0.options)[key] = v + "\x00"
			// cast to C string and set the value
			// we hope the string won't be collected while C needs it
			rv.value = (*C.char)(unsafe.Pointer(unsafe.StringData((*Nan0.options)[key])))
			Nan0.log.Debug().Msgf("Set %s=%v", key, v)
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
		// !to rewrite
		if !Nan0.multitap.supported {
			return false
		}
		info := (*[100]C.struct_retro_controller_info)(data)
		var i C.unsigned
		for i = 0; unsafe.Pointer(info[i].types) != nil; i++ {
			var j C.unsigned
			types := (*[100]C.struct_retro_controller_description)(unsafe.Pointer(info[i].types))
			for j = 0; j < info[i].num_types; j++ {
				if C.GoString(types[j].desc) == "Multitap" {
					Nan0.multitap.value = types[j].id
					return true
				}
			}
		}
		return false
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
		W:              int(Nan0.sysAvInfo.geometry.max_width),
		H:              int(Nan0.sysAvInfo.geometry.max_height),
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

	C.bridge_context_reset(Nan0.Video.hw.context_reset)
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
}
