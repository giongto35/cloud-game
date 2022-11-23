package nanoarch

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/graphics"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/image"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
)

/*
#include "libretro.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

void bridge_retro_init(void *f);
void bridge_retro_deinit(void *f);
unsigned bridge_retro_api_version(void *f);
void bridge_retro_get_system_info(void *f, struct retro_system_info *si);
void bridge_retro_get_system_av_info(void *f, struct retro_system_av_info *si);
bool bridge_retro_set_environment(void *f, void *callback);
void bridge_retro_set_video_refresh(void *f, void *callback);
void bridge_retro_set_input_poll(void *f, void *callback);
void bridge_retro_set_input_state(void *f, void *callback);
void bridge_retro_set_audio_sample(void *f, void *callback);
void bridge_retro_set_audio_sample_batch(void *f, void *callback);
bool bridge_retro_load_game(void *f, struct retro_game_info *gi);
void bridge_retro_unload_game(void *f);
void bridge_retro_run(void *f);
void bridge_retro_set_controller_port_device(void *f, unsigned port, unsigned device);

bool coreEnvironment_cgo(unsigned cmd, void *data);
void coreVideoRefresh_cgo(void *data, unsigned width, unsigned height, size_t pitch);
void coreInputPoll_cgo();
void coreAudioSample_cgo(int16_t left, int16_t right);
size_t coreAudioSampleBatch_cgo(const int16_t *data, size_t frames);
int16_t coreInputState_cgo(unsigned port, unsigned device, unsigned index, unsigned id);
void coreLog_cgo(enum retro_log_level level, const char *msg);
uintptr_t coreGetCurrentFramebuffer_cgo();
retro_proc_address_t coreGetProcAddress_cgo(const char *sym);

void bridge_context_reset(retro_hw_context_reset_t f);

void initVideo_cgo();
void deinitVideo_cgo();
void bridge_execute(void *f);
*/
import "C"

var mu sync.Mutex
var lastFrameTime int64

var libretroLogger = logger.Default()
var sdlCtx *graphics.SDL

var video struct {
	pitch    uint32
	pixFmt   uint32
	bpp      uint32
	rotation image.Angle

	baseWidth  int32
	baseHeight int32
	maxWidth   int32
	maxHeight  int32

	hw            *C.struct_retro_hw_render_callback
	isGl          bool
	autoGlContext bool
}

// default core pix format converter
var pixelFormatConverterFn = image.Rgb565
var rotationFn = image.GetRotation(image.Angle(0))

//const joypadNumKeys = int(C.RETRO_DEVICE_ID_JOYPAD_R3 + 1)
//var joy [joypadNumKeys]bool

var isGlAllowed bool
var usesLibCo bool
var coreConfig ConfigProperties

var multitap struct {
	supported bool
	enabled   bool
	value     C.unsigned
}

var systemDirectory = C.CString("./pkg/emulator/libretro/system")
var saveDirectory = C.CString(".")
var currentUser *C.char

//var seed = rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()

var bindKeysMap = map[int]int{
	C.RETRO_DEVICE_ID_JOYPAD_A:      0,
	C.RETRO_DEVICE_ID_JOYPAD_B:      1,
	C.RETRO_DEVICE_ID_JOYPAD_X:      2,
	C.RETRO_DEVICE_ID_JOYPAD_Y:      3,
	C.RETRO_DEVICE_ID_JOYPAD_L:      4,
	C.RETRO_DEVICE_ID_JOYPAD_R:      5,
	C.RETRO_DEVICE_ID_JOYPAD_SELECT: 6,
	C.RETRO_DEVICE_ID_JOYPAD_START:  7,
	C.RETRO_DEVICE_ID_JOYPAD_UP:     8,
	C.RETRO_DEVICE_ID_JOYPAD_DOWN:   9,
	C.RETRO_DEVICE_ID_JOYPAD_LEFT:   10,
	C.RETRO_DEVICE_ID_JOYPAD_RIGHT:  11,
	C.RETRO_DEVICE_ID_JOYPAD_R2:     12,
	C.RETRO_DEVICE_ID_JOYPAD_L2:     13,
	C.RETRO_DEVICE_ID_JOYPAD_R3:     14,
	C.RETRO_DEVICE_ID_JOYPAD_L3:     15,
}

//export coreVideoRefresh
func coreVideoRefresh(data unsafe.Pointer, width C.unsigned, height C.unsigned, pitch C.size_t) {
	// some cores can return nothing
	// !to add duplicate if can dup
	if data == nil {
		return
	}

	// if Libretro renders frame with OpenGL context
	isOpenGLRender := data == C.RETRO_HW_FRAME_BUFFER_VALID

	// calculate real frame width in pixels from packed data (realWidth >= width)
	packedWidth := int(uint32(pitch) / video.bpp)
	if packedWidth < 1 {
		packedWidth = int(width)
	}
	// calculate space for the video frame
	bytes := int(height) * packedWidth * int(video.bpp)

	var data_ []byte
	if isOpenGLRender {
		data_ = graphics.ReadFramebuffer(bytes, int(width), int(height))
	} else {
		data_ = (*[1 << 30]byte)(data)[:bytes:bytes]
	}

	// the image is being resized and de-rotated
	frame := image.DrawRgbaImage(
		pixelFormatConverterFn,
		rotationFn,
		image.ScaleNearestNeighbour,
		isOpenGLRender,
		int(width), int(height), packedWidth, int(video.bpp),
		data_,
		NAEmulator.vw,
		NAEmulator.vh,
	)

	t := time.Now().UnixNano()
	dt := time.Duration(t - lastFrameTime)
	lastFrameTime = t

	select {
	case NAEmulator.imageChannel <- GameFrame{Data: frame, Duration: dt}:
	default:
	}
}

//export coreInputPoll
func coreInputPoll() {}

//export coreInputState
func coreInputState(port C.unsigned, device C.unsigned, index C.unsigned, id C.unsigned) C.int16_t {
	if device == C.RETRO_DEVICE_ANALOG {
		if index > C.RETRO_DEVICE_INDEX_ANALOG_RIGHT || id > C.RETRO_DEVICE_ID_ANALOG_Y {
			return 0
		}
		axis := index*2 + id
		value := NAEmulator.players.isDpadTouched(uint(port), uint(axis))
		if value != 0 {
			return (C.int16_t)(value)
		}
	}

	if id >= 255 || index > 0 || device != C.RETRO_DEVICE_JOYPAD {
		return 0
	}

	// map from id to control key
	key, ok := bindKeysMap[int(id)]
	if !ok {
		return 0
	}

	if NAEmulator.players.isKeyPressed(uint(port), key) {
		return 1
	}

	return 0
}

func audioWrite(buf unsafe.Pointer, frames C.size_t) C.size_t {
	// !to make it mono/stereo independent
	samples := int(frames) * 2
	pcm := (*[(1 << 30) - 1]int16)(buf)[:samples:samples]

	p := make([]int16, samples)
	// copy because pcm slice refer to buf underlying pointer,
	// and buf pointer is the same in continuous frames
	copy(p, pcm)

	// 1600 = x / 1000 * 48000 * 2
	estimate := float64(len(pcm)) / float64(NAEmulator.meta.AudioSampleRate*2) * 1000000000

	select {
	case NAEmulator.audioChannel <- GameAudio{Data: p, Duration: time.Duration(estimate)}:
	default:
	}
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

//export coreLog
func coreLog(_ C.enum_retro_log_level, msg *C.char) {
	message := strings.TrimRight(C.GoString(msg), "\n")
	libretroLogger.Debug().Msg(message)
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
		username := (**C.char)(data)
		if currentUser == nil {
			currentUserGo, err := user.Current()
			if err != nil {
				currentUser = C.CString("")
			} else {
				currentUser = C.CString(currentUserGo.Username)
			}
		}
		*username = currentUser
	case C.RETRO_ENVIRONMENT_GET_LOG_INTERFACE:
		cb := (*C.struct_retro_log_callback)(data)
		cb.log = (C.retro_log_printf_t)(C.coreLog_cgo)
	case C.RETRO_ENVIRONMENT_GET_CAN_DUPE:
		bval := (*C.bool)(data)
		*bval = C.bool(true)
	case C.RETRO_ENVIRONMENT_SET_PIXEL_FORMAT:
		res, err := videoSetPixelFormat(*(*C.enum_retro_pixel_format)(data))
		if err != nil {
			libretroLogger.Fatal().Err(err).Msg("pix format failed")
		}
		return res
	case C.RETRO_ENVIRONMENT_GET_SYSTEM_DIRECTORY:
		path := (**C.char)(data)
		*path = systemDirectory
		return true
	case C.RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
		path := (**C.char)(data)
		*path = saveDirectory
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
		if val, ok := coreConfig[key]; ok {
			variable.value = val
			libretroLogger.Debug().Msgf("Set %s=%v", key, C.GoString(val))
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_HW_RENDER:
		video.isGl = isGlAllowed
		if isGlAllowed {
			video.hw = (*C.struct_retro_hw_render_callback)(data)
			video.hw.get_current_framebuffer = (C.retro_hw_get_current_framebuffer_t)(C.coreGetCurrentFramebuffer_cgo)
			video.hw.get_proc_address = (C.retro_hw_get_proc_address_t)(C.coreGetProcAddress_cgo)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_CONTROLLER_INFO:
		if multitap.supported {
			info := (*[100]C.struct_retro_controller_info)(data)
			var i C.unsigned
			for i = 0; unsafe.Pointer(info[i].types) != nil; i++ {
				var j C.unsigned
				types := (*[100]C.struct_retro_controller_description)(unsafe.Pointer(info[i].types))
				for j = 0; j < info[i].num_types; j++ {
					if C.GoString(types[j].desc) == "Multitap" {
						multitap.value = types[j].id
						return true
					}
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
	switch video.hw.context_type {
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
		W:              int(video.maxWidth),
		H:              int(video.maxHeight),
		GLAutoContext:  video.autoGlContext,
		GLVersionMajor: uint(video.hw.version_major),
		GLVersionMinor: uint(video.hw.version_minor),
		GLHasDepth:     bool(video.hw.depth),
		GLHasStencil:   bool(video.hw.stencil),
	}, libretroLogger)
	if err != nil {
		panic(err)
	}
	sdlCtx = sdl

	C.bridge_context_reset(video.hw.context_reset)
	if libretroLogger.GetLevel() < logger.InfoLevel {
		printOpenGLDriverInfo()
	}
}

//export deinitVideo
func deinitVideo() {
	C.bridge_context_reset(video.hw.context_destroy)
	if err := sdlCtx.Deinit(); err != nil {
		libretroLogger.Error().Err(err).Msg("deinit fail")
	}
	video.isGl = false
	video.autoGlContext = false
}

var (
	//retroAPIVersion              unsafe.Pointer
	retroDeinit                  unsafe.Pointer
	retroGetSystemAVInfo         unsafe.Pointer
	retroGetSystemInfo           unsafe.Pointer
	retroHandle                  unsafe.Pointer
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
)

func SetLibretroLogger(log *logger.Logger) { libretroLogger = log }

func coreLoad(meta emulator.Metadata) {
	var err error
	isGlAllowed = meta.IsGlAllowed
	usesLibCo = meta.UsesLibCo
	video.autoGlContext = meta.AutoGlContext
	coreConfig, err = ScanConfigFile(meta.ConfigPath)
	if err != nil {
		libretroLogger.Warn().Err(err).Msg("config scan has been failed")
	}

	multitap.supported = meta.HasMultitap
	multitap.enabled = false
	multitap.value = 0

	filePath := meta.LibPath
	if arch, err := core.GetCoreExt(); err == nil {
		filePath = filePath + arch.LibExt
	} else {
		libretroLogger.Warn().Err(err).Msg("system arch guesser failed")
	}

	mu.Lock()
	retroHandle, err = loadLib(filePath)
	// fallback to sequential lib loader (first successfully loaded)
	if err != nil {
		retroHandle, err = loadLibRollingRollingRolling(filePath)
		if err != nil {
			libretroLogger.Fatal().Err(err).Msgf("core load: %s, %v", filePath, err)
		}
	}

	retroInit = loadFunction(retroHandle, "retro_init")
	retroDeinit = loadFunction(retroHandle, "retro_deinit")
	//retroAPIVersion = loadFunction(retroHandle, "retro_api_version")
	retroGetSystemInfo = loadFunction(retroHandle, "retro_get_system_info")
	retroGetSystemAVInfo = loadFunction(retroHandle, "retro_get_system_av_info")
	retroSetEnvironment = loadFunction(retroHandle, "retro_set_environment")
	retroSetVideoRefresh = loadFunction(retroHandle, "retro_set_video_refresh")
	retroSetInputPoll = loadFunction(retroHandle, "retro_set_input_poll")
	retroSetInputState = loadFunction(retroHandle, "retro_set_input_state")
	retroSetAudioSample = loadFunction(retroHandle, "retro_set_audio_sample")
	retroSetAudioSampleBatch = loadFunction(retroHandle, "retro_set_audio_sample_batch")
	retroRun = loadFunction(retroHandle, "retro_run")
	retroLoadGame = loadFunction(retroHandle, "retro_load_game")
	retroUnloadGame = loadFunction(retroHandle, "retro_unload_game")
	retroSerializeSize = loadFunction(retroHandle, "retro_serialize_size")
	retroSerialize = loadFunction(retroHandle, "retro_serialize")
	retroUnserialize = loadFunction(retroHandle, "retro_unserialize")
	retroSetControllerPortDevice = loadFunction(retroHandle, "retro_set_controller_port_device")
	retroGetMemorySize = loadFunction(retroHandle, "retro_get_memory_size")
	retroGetMemoryData = loadFunction(retroHandle, "retro_get_memory_data")

	mu.Unlock()

	C.bridge_retro_set_environment(retroSetEnvironment, C.coreEnvironment_cgo)
	C.bridge_retro_set_video_refresh(retroSetVideoRefresh, C.coreVideoRefresh_cgo)
	C.bridge_retro_set_input_poll(retroSetInputPoll, C.coreInputPoll_cgo)
	C.bridge_retro_set_input_state(retroSetInputState, C.coreInputState_cgo)
	C.bridge_retro_set_audio_sample(retroSetAudioSample, C.coreAudioSample_cgo)
	C.bridge_retro_set_audio_sample_batch(retroSetAudioSampleBatch, C.coreAudioSampleBatch_cgo)

	C.bridge_retro_init(retroInit)
}

func slurp(path string, size int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	bytes := make([]byte, size)
	buffer := bufio.NewReader(f)
	_, err = buffer.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func coreLoadGame(filename string) {
	lastFrameTime = 0

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}
	_ = file.Close()

	size := fi.Size()
	libretroLogger.Debug().Msgf("ROM size: %v", size)

	csFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(csFilename))
	gi := C.struct_retro_game_info{
		path: csFilename,
		size: C.size_t(size),
	}

	si := C.struct_retro_system_info{}
	C.bridge_retro_get_system_info(retroGetSystemInfo, &si)
	if libretroLogger.GetLevel() < logger.InfoLevel {
		libretroLogger.Debug().Msgf("Core: %s %s (%s)",
			C.GoString(si.library_name),
			C.GoString(si.library_version),
			C.GoString(si.valid_extensions),
		)
	}

	if !si.need_fullpath {
		bytes, err := slurp(filename, size)
		if err != nil {
			panic(err)
		}
		cstr := C.CString(string(bytes))
		defer C.free(unsafe.Pointer(cstr))
		gi.data = unsafe.Pointer(cstr)
	}

	if ok := C.bridge_retro_load_game(retroLoadGame, &gi); !ok {
		libretroLogger.Fatal().Msg("The core failed to load the content.")
	}

	avi := C.struct_retro_system_av_info{}
	C.bridge_retro_get_system_av_info(retroGetSystemAVInfo, &avi)

	// Append the library name to the window title.
	NAEmulator.meta.AudioSampleRate = int(avi.timing.sample_rate)
	NAEmulator.meta.Fps = float64(avi.timing.fps)
	NAEmulator.meta.BaseWidth = int(avi.geometry.base_width)
	NAEmulator.meta.BaseHeight = int(avi.geometry.base_height)
	// set aspect ratio
	/* Nominal aspect ratio of game. If aspect_ratio is <= 0.0,
	an aspect ratio of base_width / base_height is assumed.
	* A frontend could override this setting, if desired. */
	ratio := float64(avi.geometry.aspect_ratio)
	if ratio <= 0.0 {
		ratio = float64(avi.geometry.base_width) / float64(avi.geometry.base_height)
	}
	NAEmulator.meta.Ratio = ratio

	if libretroLogger.GetLevel() < logger.InfoLevel {
		libretroLogger.Debug().Msgf("Core media info: %vx%v (%vx%v), [%vfps], AR [%v], audio [%vHz]",
			avi.geometry.base_width, avi.geometry.base_height,
			avi.geometry.max_width, avi.geometry.max_height,
			avi.timing.fps, ratio, avi.timing.sample_rate,
		)
	}

	video.maxWidth = int32(avi.geometry.max_width)
	video.maxHeight = int32(avi.geometry.max_height)
	video.baseWidth = int32(avi.geometry.base_width)
	video.baseHeight = int32(avi.geometry.base_height)
	if video.isGl {
		bufS := int(video.maxWidth * video.maxHeight * int32(video.bpp))
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
	maxPort := 4 // controllersNum
	for i := 0; i < maxPort; i++ {
		C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, C.uint(i), C.RETRO_DEVICE_JOYPAD)
	}
}

func toggleMultitap() {
	if multitap.supported && multitap.value != 0 {
		// Official SNES games only support a single multitap device
		// Most require it to be plugged in player 2 port
		// And Snes9X requires it to be "plugged" after the game is loaded
		// Control this from the browser since player 2 will stop working in some games if multitap is "plugged" in
		if multitap.enabled {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, C.RETRO_DEVICE_JOYPAD)
		} else {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, multitap.value)
		}
		multitap.enabled = !multitap.enabled
	}
}

func nanoarchShutdown() {
	if usesLibCo {
		thread.Main(func() {
			C.bridge_execute(retroUnloadGame)
			C.bridge_execute(retroDeinit)
			if video.isGl {
				C.bridge_execute(C.deinitVideo_cgo)
			}
		})
	} else {
		if video.isGl {
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
		if video.isGl {
			thread.Main(func() {
				deinitVideo()
				runtime.UnlockOSThread()
			})
		}
	}

	setRotation(0)
	if err := closeLib(retroHandle); err != nil {
		libretroLogger.Error().Err(err).Msg("lib close failed")
	}
	for _, element := range coreConfig {
		C.free(unsafe.Pointer(element))
	}
	image.Clear()
}

func nanoarchRun() {
	if usesLibCo {
		C.bridge_execute(retroRun)
	} else {
		if video.isGl {
			// running inside a go routine, lock the thread to make sure the OpenGL context stays current
			runtime.LockOSThread()
			if err := sdlCtx.BindContext(); err != nil {
				libretroLogger.Error().Err(err).Msg("ctx bind fail")
			}
		}
		C.bridge_retro_run(retroRun)
		if video.isGl {
			runtime.UnlockOSThread()
		}
	}
}

func videoSetPixelFormat(format uint32) (C.bool, error) {
	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		video.pixFmt = image.BitFormatShort5551
		if err := graphics.SetPixelFormat(graphics.UnsignedShort5551); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", video.pixFmt)
		}
		video.bpp = 2
		// format is not implemented
		pixelFormatConverterFn = nil
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		video.pixFmt = image.BitFormatInt8888Rev
		if err := graphics.SetPixelFormat(graphics.UnsignedInt8888Rev); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", video.pixFmt)
		}
		video.bpp = 4
		pixelFormatConverterFn = image.Rgba8888
	case C.RETRO_PIXEL_FORMAT_RGB565:
		video.pixFmt = image.BitFormatShort565
		if err := graphics.SetPixelFormat(graphics.UnsignedShort565); err != nil {
			return false, fmt.Errorf("unknown pixel format %v", video.pixFmt)
		}
		video.bpp = 2
		pixelFormatConverterFn = image.Rgb565
	default:
		return false, fmt.Errorf("unknown pixel type %v", format)
	}
	return true, nil
}

func setRotation(rotation uint) {
	if rotation == uint(video.rotation) {
		return
	}
	video.rotation = image.Angle(rotation)
	rotationFn = image.GetRotation(video.rotation)
	NAEmulator.meta.Rotation = rotationFn
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
