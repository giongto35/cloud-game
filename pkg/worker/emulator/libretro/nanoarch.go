package libretro

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/graphics"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/image"
	"github.com/giongto35/cloud-game/v2/pkg/worker/thread"
)

/*
#cgo CFLAGS: -Wall -O3

#include "libretro.h"
#include "nanoarch.h"
#include <stdlib.h>
*/
import "C"

const lastKey = int(C.RETRO_DEVICE_ID_JOYPAD_R3)

type (
	nanoarch struct {
		v         video
		multitap  multitap
		rot       *image.Rotate
		sysInfo   C.struct_retro_system_info
		sysAvInfo C.struct_retro_system_av_info
		reserved  chan struct{} // limits concurrent use
	}
	video struct {
		pixFmt        uint32
		bpp           int
		hw            *C.struct_retro_hw_render_callback
		isGl          bool
		autoGlContext bool
	}
	multitap struct {
		supported bool
		enabled   bool
		value     C.unsigned
	}
	// defines any memory state of the emulator
	state []byte
	mem   struct {
		ptr  unsafe.Pointer
		size uint
	}
)

// Global link for C callbacks to Go
var nano = nanoarch{
	// this thing forbids concurrent use of the emulator
	reserved: make(chan struct{}, 1),
}

var (
	coreConfig       *CoreProperties
	frontend         *Frontend
	lastFrameTime    int64
	libretroLogger   = logger.Default()
	sdlCtx           *graphics.SDL
	usesLibCo        bool
	cSaveDirectory   *C.char
	cSystemDirectory *C.char
	cUserName        *C.char

	initOnce sync.Once
)

const rawAudioBuffer = 4096 // 4K
var (
	rawAudioPool = sync.Pool{New: func() any { return make([]int16, rawAudioBuffer) }}
	audioPool    = sync.Pool{New: func() any { return &emulator.GameAudio{} }}
	videoPool    = sync.Pool{New: func() any { return &emulator.GameFrame{} }}
)

func init() {
	nano.reserved <- struct{}{}
	usr, err := user.Current()
	if err == nil {
		cUserName = C.CString(usr.Name)
	} else {
		cUserName = C.CString("retro")
	}
}

func Init(localPath string) {
	initOnce.Do(func() {
		cSaveDirectory = C.CString(localPath + string(os.PathSeparator) + "legacy_save")
		cSystemDirectory = C.CString(localPath + string(os.PathSeparator) + "system")
	})
}

//export coreVideoRefresh
func coreVideoRefresh(data unsafe.Pointer, width C.unsigned, height C.unsigned, pitch C.size_t) {
	// some cores can return nothing
	// !to add duplicate if can dup
	if data == nil {
		return
	}

	// calculate real frame width in pixels from packed data (realWidth >= width)
	packedWidth := int(pitch) / nano.v.bpp
	if packedWidth < 1 {
		packedWidth = int(width)
	}
	// calculate space for the video frame
	bytes := int(height) * packedWidth * nano.v.bpp

	// if Libretro renders frame with OpenGL context
	isOpenGLRender := data == C.RETRO_HW_FRAME_BUFFER_VALID
	var data_ []byte
	if isOpenGLRender {
		data_ = graphics.ReadFramebuffer(bytes, int(width), int(height))
	} else {
		data_ = (*[1 << 30]byte)(data)[:bytes:bytes]
	}

	// the image is being resized and de-rotated
	frame := image.DrawRgbaImage(
		nano.v.pixFmt,
		nano.rot,
		image.ScaleNearestNeighbour,
		isOpenGLRender,
		int(width), int(height), packedWidth, nano.v.bpp,
		data_,
		frontend.vw,
		frontend.vh,
		frontend.th,
	)

	t := time.Now().UnixNano()
	dt := time.Duration(t - lastFrameTime)
	lastFrameTime = t

	if len(frame.Pix) == 0 {
		// this should not be happening, will crash yuv
		libretroLogger.Error().Msgf("skip empty frame %v", frame.Bounds())
		return
	}

	fr := videoPool.Get().(*emulator.GameFrame)
	fr.Data = frame
	fr.Duration = dt
	frontend.onVideo(fr)
	videoPool.Put(fr)
}

//export coreInputPoll
func coreInputPoll() {}

//export coreInputState
func coreInputState(port C.unsigned, device C.unsigned, index C.unsigned, id C.unsigned) C.int16_t {
	if port >= maxPort {
		return KeyReleased
	}

	if device == C.RETRO_DEVICE_ANALOG {
		if index > C.RETRO_DEVICE_INDEX_ANALOG_RIGHT || id > C.RETRO_DEVICE_ID_ANALOG_Y {
			return 0
		}
		axis := index*2 + id
		value := frontend.input.isDpadTouched(uint(port), uint(axis))
		if value != 0 {
			return (C.int16_t)(value)
		}
	}

	key := int(id)
	if key > lastKey || index > 0 || device != C.RETRO_DEVICE_JOYPAD {
		return KeyReleased
	}
	if frontend.input.isKeyPressed(uint(port), key) == KeyPressed {
		return KeyPressed
	}
	return KeyReleased
}

func audioWrite(buf unsafe.Pointer, frames C.size_t) C.size_t {
	samples := int(frames) << 1
	src := (*[rawAudioBuffer]int16)(buf)[:samples]
	dst := rawAudioPool.Get().([]int16)[:samples]
	copy(dst, src)

	// 1600 = x / 1000 * 48000 * 2
	estimate := float64(samples) / float64(int(nano.sysAvInfo.timing.sample_rate)<<1) * 1000000000

	fr := audioPool.Get().(*emulator.GameAudio)
	fr.Data = dst
	fr.Duration = time.Duration(estimate)
	frontend.onAudio(fr)
	audioPool.Put(fr)
	rawAudioPool.Put(dst)

	return frames
}

//export coreAudioSample
func coreAudioSample(left C.int16_t, right C.int16_t) {
	buf := []C.int16_t{left, right}
	audioWrite(unsafe.Pointer(&buf), 1)
}

//export coreAudioSampleBatch
func coreAudioSampleBatch(data unsafe.Pointer, frames C.size_t) C.size_t {
	return audioWrite(data, frames)
}

func m(m *C.char) string { return strings.TrimRight(C.GoString(m), "\n") }

//export coreLog
func coreLog(level C.enum_retro_log_level, msg *C.char) {
	switch int(level) {
	// with debug level cores have too much logs
	case 0: // RETRO_LOG_DEBUG
		libretroLogger.Debug().MsgFunc(func() string { return m(msg) })
	case 1: // RETRO_LOG_INFO
		libretroLogger.Info().MsgFunc(func() string { return m(msg) })
	case 2: // RETRO_LOG_WARN
		libretroLogger.Warn().MsgFunc(func() string { return m(msg) })
	case 3: // RETRO_LOG_ERROR
		libretroLogger.Error().MsgFunc(func() string { return m(msg) })
	default:
		libretroLogger.Log().MsgFunc(func() string { return m(msg) })
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
	switch cmd {
	case C.RETRO_ENVIRONMENT_GET_USERNAME:
		*(**C.char)(data) = cUserName
	case C.RETRO_ENVIRONMENT_GET_LOG_INTERFACE:
		cb := (*C.struct_retro_log_callback)(data)
		cb.log = (C.retro_log_printf_t)(C.coreLog_cgo)
	case C.RETRO_ENVIRONMENT_GET_CAN_DUPE:
		*(*C.bool)(data) = C.bool(true)
	case C.RETRO_ENVIRONMENT_SET_PIXEL_FORMAT:
		res, err := videoSetPixelFormat(*(*C.enum_retro_pixel_format)(data))
		if err != nil {
			libretroLogger.Fatal().Err(err).Msg("pix format failed")
		}
		return res
	case C.RETRO_ENVIRONMENT_GET_SYSTEM_DIRECTORY:
		*(**C.char)(data) = cSystemDirectory
		return true
	case C.RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
		*(**C.char)(data) = cSaveDirectory
		return true
	case C.RETRO_ENVIRONMENT_SHUTDOWN:
		//window.SetShouldClose(true)
		return true
		/*
			Sets screen rotation of graphics.
			Valid values are 0, 1, 2, 3, which rotates screen by 0, 90, 180, 270 degrees
			ccw respectively.
		*/
	case C.RETRO_ENVIRONMENT_SET_ROTATION:
		setRotation(*(*uint)(data) % 4)
		return true
	case C.RETRO_ENVIRONMENT_GET_VARIABLE:
		variable := (*C.struct_retro_variable)(data)
		key := C.GoString(variable.key)
		if val, ok := coreConfig.Get(key); ok {
			variable.value = (*C.char)(val)
			libretroLogger.Debug().Msgf("Set %s=%v", key, C.GoString(variable.value))
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_HW_RENDER:
		if nano.v.isGl {
			nano.v.hw = (*C.struct_retro_hw_render_callback)(data)
			nano.v.hw.get_current_framebuffer = (C.retro_hw_get_current_framebuffer_t)(C.coreGetCurrentFramebuffer_cgo)
			nano.v.hw.get_proc_address = (C.retro_hw_get_proc_address_t)(C.coreGetProcAddress_cgo)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_CONTROLLER_INFO:
		if !nano.multitap.supported {
			return false
		}
		info := (*[100]C.struct_retro_controller_info)(data)
		var i C.unsigned
		for i = 0; unsafe.Pointer(info[i].types) != nil; i++ {
			var j C.unsigned
			types := (*[100]C.struct_retro_controller_description)(unsafe.Pointer(info[i].types))
			for j = 0; j < info[i].num_types; j++ {
				if C.GoString(types[j].desc) == "Multitap" {
					nano.multitap.value = types[j].id
					return true
				}
			}
		}
		return false
	default:
		return false
	}
	return true
}

//export initVideo
func initVideo() {
	var context graphics.Context
	switch nano.v.hw.context_type {
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
		W:              int(nano.sysAvInfo.geometry.max_width),
		H:              int(nano.sysAvInfo.geometry.max_height),
		GLAutoContext:  nano.v.autoGlContext,
		GLVersionMajor: uint(nano.v.hw.version_major),
		GLVersionMinor: uint(nano.v.hw.version_minor),
		GLHasDepth:     bool(nano.v.hw.depth),
		GLHasStencil:   bool(nano.v.hw.stencil),
	}, libretroLogger)
	if err != nil {
		panic(err)
	}
	sdlCtx = sdl

	C.bridge_context_reset(nano.v.hw.context_reset)
	if libretroLogger.GetLevel() < logger.InfoLevel {
		printOpenGLDriverInfo()
	}
}

//export deinitVideo
func deinitVideo() {
	C.bridge_context_reset(nano.v.hw.context_destroy)
	if err := sdlCtx.Deinit(); err != nil {
		libretroLogger.Error().Err(err).Msg("deinit fail")
	}
	nano.v.isGl = false
	nano.v.autoGlContext = false
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

func SetLibretroLogger(log *logger.Logger) { libretroLogger = log }

func coreLoad(meta emulator.Metadata) {
	var err error
	nano.v.isGl = meta.IsGlAllowed
	usesLibCo = meta.UsesLibCo
	nano.v.autoGlContext = meta.AutoGlContext
	coreConfig, err = ReadProperties(meta.ConfigPath)
	if err != nil {
		libretroLogger.Warn().Err(err).Msg("config scan has been failed")
	}

	nano.multitap.supported = meta.HasMultitap
	nano.multitap.enabled = false
	nano.multitap.value = 0

	filePath := meta.LibPath
	if arch, err := GetCoreExt(); err == nil {
		filePath = filePath + arch.LibExt
	} else {
		libretroLogger.Warn().Err(err).Msg("system arch guesser failed")
	}

	coreLib, err = loadLib(filePath)
	// fallback to sequential lib loader (first successfully loaded)
	if err != nil {
		coreLib, err = loadLibRollingRollingRolling(filePath)
		if err != nil {
			libretroLogger.Fatal().Err(err).Msgf("core load: %s, %v", filePath, err)
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

	C.bridge_retro_set_environment(retroSetEnvironment, C.coreEnvironment_cgo)
	C.bridge_retro_set_video_refresh(retroSetVideoRefresh, C.coreVideoRefresh_cgo)
	C.bridge_retro_set_input_poll(retroSetInputPoll, C.coreInputPoll_cgo)
	C.bridge_retro_set_input_state(retroSetInputState, C.coreInputState_cgo)
	C.bridge_retro_set_audio_sample(retroSetAudioSample, C.coreAudioSample_cgo)
	C.bridge_retro_set_audio_sample_batch(retroSetAudioSampleBatch, C.coreAudioSampleBatch_cgo)

	C.bridge_retro_init(retroInit)

	C.bridge_retro_get_system_info(retroGetSystemInfo, &nano.sysInfo)
	libretroLogger.Debug().Msgf("System >>> %s (%s) [%s] nfp: %v",
		C.GoString(nano.sysInfo.library_name), C.GoString(nano.sysInfo.library_version),
		C.GoString(nano.sysInfo.valid_extensions), bool(nano.sysInfo.need_fullpath))
}

func LoadGame(path string) error {
	lastFrameTime = 0

	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	fileSize := fi.Size()
	libretroLogger.Debug().Msgf("ROM size: %v", byteCountBinary(fileSize))

	fPath := C.CString(path)
	defer C.free(unsafe.Pointer(fPath))
	gi := C.struct_retro_game_info{path: fPath, size: C.size_t(fileSize)}

	if !bool(nano.sysInfo.need_fullpath) {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dat := C.CString(string(bytes))
		gi.data = unsafe.Pointer(dat)
		defer C.free(unsafe.Pointer(dat))
	}

	if ok := C.bridge_retro_load_game(retroLoadGame, &gi); !ok {
		return fmt.Errorf("core failed to load ROM: %v", path)
	}

	C.bridge_retro_get_system_av_info(retroGetSystemAVInfo, &nano.sysAvInfo)
	libretroLogger.Debug().Msgf("System A/V >>> %vx%v (%vx%v), [%vfps], AR [%v], audio [%vHz]",
		nano.sysAvInfo.geometry.base_width, nano.sysAvInfo.geometry.base_height,
		nano.sysAvInfo.geometry.max_width, nano.sysAvInfo.geometry.max_height,
		nano.sysAvInfo.timing.fps, nano.sysAvInfo.geometry.aspect_ratio, nano.sysAvInfo.timing.sample_rate,
	)

	if nano.v.isGl {
		bufS := int(nano.sysAvInfo.geometry.max_width*nano.sysAvInfo.geometry.max_height) * nano.v.bpp
		graphics.SetBuffer(bufS)
		libretroLogger.Info().Msgf("Set buffer: %v", byteCountBinary(int64(bufS)))
		if usesLibCo {
			C.bridge_execute(C.initVideo_cgo)
		} else {
			runtime.LockOSThread()
			initVideo()
			runtime.UnlockOSThread()
		}
	}

	// set default controller types on all ports
	for i := 0; i < maxPort; i++ {
		C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(i), C.RETRO_DEVICE_JOYPAD)
	}

	return nil
}

func toggleMultitap() {
	if nano.multitap.supported && nano.multitap.value != 0 {
		// Official SNES games only support a single multitap device
		// Most require it to be plugged in player 2 port
		// And Snes9X requires it to be "plugged" after the game is loaded
		// Control this from the browser since player 2 will stop working in some games if multitap is "plugged" in
		if nano.multitap.enabled {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, C.RETRO_DEVICE_JOYPAD)
		} else {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, nano.multitap.value)
		}
		nano.multitap.enabled = !nano.multitap.enabled
	}
}

func nanoarchShutdown() {
	if usesLibCo {
		thread.Main(func() {
			C.bridge_execute(retroUnloadGame)
			C.bridge_execute(retroDeinit)
			if nano.v.isGl {
				C.bridge_execute(C.deinitVideo_cgo)
			}
		})
	} else {
		if nano.v.isGl {
			thread.Main(func() {
				// running inside a go routine, lock the thread to make sure the OpenGL context stays current
				runtime.LockOSThread()
				if err := sdlCtx.BindContext(); err != nil {
					libretroLogger.Error().Err(err).Msg("ctx switch fail")
				}
			})
		}
		C.bridge_retro_unload_game(retroUnloadGame)
		C.bridge_retro_deinit(retroDeinit)
		if nano.v.isGl {
			thread.Main(func() {
				deinitVideo()
				runtime.UnlockOSThread()
			})
		}
	}

	setRotation(0)
	if err := closeLib(coreLib); err != nil {
		libretroLogger.Error().Err(err).Msg("lib close failed")
	}
	coreConfig.Free()
	image.Clear()
}

func run() {
	if usesLibCo {
		C.bridge_execute(retroRun)
	} else {
		if nano.v.isGl {
			// running inside a go routine, lock the thread to make sure the OpenGL context stays current
			runtime.LockOSThread()
			if err := sdlCtx.BindContext(); err != nil {
				libretroLogger.Error().Err(err).Msg("ctx bind fail")
			}
		}
		C.bridge_retro_run(retroRun)
		if nano.v.isGl {
			runtime.UnlockOSThread()
		}
	}
}

func videoSetPixelFormat(format uint32) (C.bool, error) {
	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		nano.v.pixFmt = image.BitFormatShort5551
		if err := graphics.SetPixelFormat(graphics.UnsignedShort5551); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", nano.v.pixFmt)
		}
		nano.v.bpp = 2
		// format is not implemented
		return false, fmt.Errorf("unsupported pixel type %v converter", format)
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		nano.v.pixFmt = image.BitFormatInt8888Rev
		if err := graphics.SetPixelFormat(graphics.UnsignedInt8888Rev); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", nano.v.pixFmt)
		}
		nano.v.bpp = 4
	case C.RETRO_PIXEL_FORMAT_RGB565:
		nano.v.pixFmt = image.BitFormatShort565
		if err := graphics.SetPixelFormat(graphics.UnsignedShort565); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", nano.v.pixFmt)
		}
		nano.v.bpp = 2
	default:
		return false, fmt.Errorf("unknown pixel type %v", format)
	}
	return true, nil
}

func setRotation(rotation uint) {
	if nano.rot != nil && rotation == uint(nano.rot.Angle) {
		return
	}
	if rotation > 0 {
		r := image.GetRotation(image.Angle(rotation))
		nano.rot = &r
	} else {
		nano.rot = nil
	}
	libretroLogger.Debug().Msgf("Image rotated %v°", map[uint]uint{0: 0, 1: 90, 2: 180, 3: 270}[rotation])
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
	libretroLogger.Debug().Msg(openGLInfo.String())
}

// saveStateSize returns the amount of data the implementation requires
// to serialize internal state (save states).
func saveStateSize() uint { return uint(C.bridge_retro_serialize_size(retroSerializeSize)) }

// getSaveState returns emulator internal state.
func getSaveState() (state, error) {
	size := saveStateSize()
	data := C.malloc(C.size_t(size))
	defer C.free(data)
	if !bool(C.bridge_retro_serialize(retroSerialize, data, C.size_t(size))) {
		return nil, errors.New("retro_serialize failed")
	}
	return C.GoBytes(data, C.int(size)), nil
}

// restoreSaveState restores emulator internal state.
func restoreSaveState(st state) error {
	if len(st) == 0 {
		return nil
	}
	size := saveStateSize()
	if !bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&st[0]), C.size_t(size))) {
		return errors.New("retro_unserialize failed")
	}
	return nil
}

// getSaveRAM returns the game save RAM (cartridge) data or a nil slice.
func getSaveRAM() state {
	memory := ptSaveRAM()
	if memory == nil {
		return nil
	}
	return C.GoBytes(memory.ptr, C.int(memory.size))
}

// restoreSaveRAM restores game save RAM.
func restoreSaveRAM(st state) {
	if len(st) == 0 {
		return
	}
	if memory := ptSaveRAM(); memory != nil {
		sram := (*[1 << 30]byte)(memory.ptr)[:memory.size:memory.size]
		copy(sram, st)
	}
}

// getMemorySize returns memory region size.
func getMemorySize(id uint) uint {
	return uint(C.bridge_retro_get_memory_size(retroGetMemorySize, C.uint(id)))
}

// getMemoryData returns a pointer to memory data.
func getMemoryData(id uint) unsafe.Pointer {
	return C.bridge_retro_get_memory_data(retroGetMemoryData, C.uint(id))
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
