package nanoarch

import (
	"bufio"
	"errors"
	"fmt"
	stdimage "image"
	"log"
	"math/rand"
	"os"
	"os/user"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/disintegration/imaging"
	"github.com/faiface/mainthread"
	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/emulator/libretro/image"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

/*
#include "libretro.h"
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <dlfcn.h>
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
size_t bridge_retro_get_memory_size(void *f, unsigned id);
void* bridge_retro_get_memory_data(void *f, unsigned id);
bool bridge_retro_serialize(void *f, void *data, size_t size);
bool bridge_retro_unserialize(void *f, void *data, size_t size);
size_t bridge_retro_serialize_size(void *f);
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

var video struct {
	pitch       uint32
	pixFmt      uint32
	bpp         uint32
	rotation    image.Angle
	fbo         uint32
	rbo         uint32
	tex         uint32
	hw          *C.struct_retro_hw_render_callback
	window      *sdl.Window
	context     sdl.GLContext
	isGl        bool
	max_width   int32
	max_height  int32
	base_width  int32
	base_height int32
}

// default core pix format converter
var pixelFormatConverterFn = image.Rgb565
var rotationFn = image.GetRotation(image.Angle(0))

const bufSize = 1024 * 4
const joypadNumKeys = int(C.RETRO_DEVICE_ID_JOYPAD_R3 + 1)

var joy [joypadNumKeys]bool
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

var seed = rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()

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

type CloudEmulator interface {
	Start(path string)
	SaveGame(saveExtraFunc func() error) error
	LoadGame() error
	GetHashPath() string
	Close()
	ToggleMultitap() error
}

//export coreVideoRefresh
func coreVideoRefresh(data unsafe.Pointer, width C.unsigned, height C.unsigned, pitch C.size_t) {
	// some cores can return nothing
	if data == nil {
		return
	}
	// divide by 8333 to give us the equivalent of a 120fps resolution
	timestamp := uint32(time.Now().UnixNano() / 8333) + seed

	if (data == C.RETRO_HW_FRAME_BUFFER_VALID) {
		im := stdimage.NewNRGBA(stdimage.Rect(0, 0, int(width), int(height)))
		gl.BindFramebuffer(gl.FRAMEBUFFER, video.fbo)
		gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(im.Pix))
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		im = imaging.FlipV(im)
		rgba := &stdimage.RGBA{
			Pix:    im.Pix,
			Stride: im.Stride,
			Rect:   im.Rect,
		}
		NAEmulator.imageChannel <- GameFrame{ Image: rgba, Timestamp: timestamp }
		return
	}

	// calculate real frame width in pixels from packed data (realWidth >= width)
	packedWidth := int(uint32(pitch) / video.bpp)

	// convert data from C
	bytes := int(height) * packedWidth * int(video.bpp)
	data_ := (*[1 << 30]byte)(data)[:bytes:bytes]

	// the image is being resized and de-rotated
	image.DrawRgbaImage(
		pixelFormatConverterFn,
		rotationFn,
		image.ScaleNearestNeighbour,
		int(width), int(height), packedWidth, int(video.bpp),
		data_,
		outputImg,
	)

	// the image is pushed into a channel
	// where it will be distributed with fan-out
	NAEmulator.imageChannel <- GameFrame{ Image: outputImg, Timestamp: timestamp }
}

//export coreInputPoll
func coreInputPoll() {
}

//export coreInputState
func coreInputState(port C.unsigned, device C.unsigned, index C.unsigned, id C.unsigned) C.int16_t {
	if device == C.RETRO_DEVICE_ANALOG {
		if index > C.RETRO_DEVICE_INDEX_ANALOG_RIGHT || id > C.RETRO_DEVICE_ID_ANALOG_Y {
			return 0
		}
		axis := index * 2 + id
		for k := range NAEmulator.controllersMap {
			value := NAEmulator.controllersMap[k][port].axes[axis]
			if value != 0 {
				return (C.int16_t)(value)
			}
		}
	}

	if id >= 255 || index > 0 || device != C.RETRO_DEVICE_JOYPAD {
		return 0
	}

	// map from id to controll key
	key, ok := bindKeysMap[int(id)]
	if !ok {
		return 0
	}

	// check if any player is pressing that key
	for k := range NAEmulator.controllersMap {
		if ((NAEmulator.controllersMap[k][port].keyState >> uint(key)) & 1) == 1 {
			return 1
		}
	}
	return 0
}

func audioWrite2(buf unsafe.Pointer, frames C.size_t) C.size_t {
	// !to make it mono/stereo independent
	samples := int(frames) * 2
	pcm := (*[(1 << 30) - 1]int16)(buf)[:samples:samples]

	p := make([]int16, samples)
	// copy because pcm slice refer to buf underlying pointer, and buf pointer is the same in continuos frames
	copy(p, pcm)

	select {
	case NAEmulator.audioChannel <- p:
	default:
	}

	return frames
}

//export coreAudioSample
func coreAudioSample(left C.int16_t, right C.int16_t) {
	buf := []C.int16_t{left, right}
	audioWrite2(unsafe.Pointer(&buf), 1)
}

//export coreAudioSampleBatch
func coreAudioSampleBatch(data unsafe.Pointer, frames C.size_t) C.size_t {
	return audioWrite2(data, frames)
}

//export coreLog
func coreLog(level C.enum_retro_log_level, msg *C.char) {
	fmt.Print("[Log]: ", C.GoString(msg))
}

//export coreGetCurrentFramebuffer
func coreGetCurrentFramebuffer() C.uintptr_t {
	return (C.uintptr_t)(video.fbo)
}

//export coreGetProcAddress
func coreGetProcAddress(sym *C.char) C.retro_proc_address_t {
	return (C.retro_proc_address_t) (sdl.GLGetProcAddress(C.GoString(sym)))
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
		break
	case C.RETRO_ENVIRONMENT_GET_LOG_INTERFACE:
		cb := (*C.struct_retro_log_callback)(data)
		cb.log = (C.retro_log_printf_t)(C.coreLog_cgo)
		break
	case C.RETRO_ENVIRONMENT_GET_CAN_DUPE:
		bval := (*C.bool)(data)
		*bval = C.bool(true)
		break
	case C.RETRO_ENVIRONMENT_SET_PIXEL_FORMAT:
		format := (*C.enum_retro_pixel_format)(data)
		if *format > C.RETRO_PIXEL_FORMAT_RGB565 {
			return false
		}
		return videoSetPixelFormat(*format)
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
		setRotation(*(*int)(data) % 4)
		return true
	case C.RETRO_ENVIRONMENT_GET_VARIABLE:
		variable := (*C.struct_retro_variable)(data)
		key := C.GoString(variable.key)
		if val, ok := coreConfig[key]; ok {
			fmt.Printf("[Env]: get variable: key:%v value:%v\n", key, C.GoString(val))
			variable.value = val
			return true
		}
		// fmt.Printf("[Env]: get variable: key:%v not found\n", key)
		return false
	case C.RETRO_ENVIRONMENT_SET_HW_RENDER:
		if (isGlAllowed) {
			video.isGl = true
			// runtime.LockOSThread()
			video.hw = (*C.struct_retro_hw_render_callback)(data)
			video.hw.get_current_framebuffer = (C.retro_hw_get_current_framebuffer_t)(C.coreGetCurrentFramebuffer_cgo)
			video.hw.get_proc_address = (C.retro_hw_get_proc_address_t)(C.coreGetProcAddress_cgo)
			return true
		}
		return false
	case C.RETRO_ENVIRONMENT_SET_CONTROLLER_INFO:
		if (multitap.supported) {
			info := (*[100]C.struct_retro_controller_info)(data)
			var i C.unsigned
			for i = 0; unsafe.Pointer(info[i].types) != nil; i++ {
				var j C.unsigned
				types := (*[100]C.struct_retro_controller_description)(unsafe.Pointer(info[i].types))
				for j = 0; j < info[i].num_types; j++ {
					if (C.GoString(types[j].desc) == "Multitap") {
						multitap.value = types[j].id
						return true
					}
				}
			}
		}
		return false
	default:
		//fmt.Println("[Env]: command not implemented", cmd)
		return false
	}
	return true
}

func init() {
}

var sdlInitialized = false
//export initVideo
func initVideo() {
	// create_window()
	var winTitle string = "CloudRetro"
	var winWidth, winHeight int32 = 1, 1
	var err error

	if !sdlInitialized {
		sdlInitialized = true
		if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
			panic(err)
		}
	}

	switch video.hw.context_type {
	case C.RETRO_HW_CONTEXT_OPENGL_CORE:
		fmt.Println("RETRO_HW_CONTEXT_OPENGL_CORE")
		sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
		break
	case C.RETRO_HW_CONTEXT_OPENGLES2:
		fmt.Println("RETRO_HW_CONTEXT_OPENGLES2")
		sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
		sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 0)
		break
	case C.RETRO_HW_CONTEXT_OPENGL:
		fmt.Println("RETRO_HW_CONTEXT_OPENGL")
		if video.hw.version_major >= 3 {
			sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_COMPATIBILITY)
		}
		break
	default:
		fmt.Println("Unsupported hw context:", video.hw.context_type)
	}

	// In OSX 10.14+ window creation and context creation must happen in the main thread
	mainthread.Call(func() {
		video.window, err = sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, winWidth, winHeight, sdl.WINDOW_OPENGL)
		if err != nil {
			panic(err)
		}

		video.context, err = video.window.GLCreateContext()
		if err != nil {
			panic(err)
		}
	})
	// Bind context to current thread
	video.window.GLMakeCurrent(video.context)

	if err = gl.InitWithProcAddrFunc(sdl.GLGetProcAddress); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version: ", version)

	// init_texture()
	gl.GenTextures(1, &video.tex)
	if video.tex < 0 {
		panic(fmt.Sprintf("GenTextures: 0x%X", video.tex))
	}

	gl.BindTexture(gl.TEXTURE_2D, video.tex)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, video.max_width, video.max_height, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	gl.BindTexture(gl.TEXTURE_2D, 0)

	//init_framebuffer()
	gl.GenFramebuffers(1, &video.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, video.fbo)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, video.tex, 0)

	if video.hw.depth {
		gl.GenRenderbuffers(1, &video.rbo);
		gl.BindRenderbuffer(gl.RENDERBUFFER, video.rbo)
		if video.hw.stencil {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, video.base_width, video.base_height);
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, video.rbo);
		} else {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, video.base_width, video.base_height);
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, video.rbo);
		}
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	}

	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if status != gl.FRAMEBUFFER_COMPLETE {
		if e := gl.GetError(); e != gl.NO_ERROR {
			panic(fmt.Sprintf("GL error: 0x%X, Frame status: 0x%X", e, status))
		}
		panic(fmt.Sprintf("Frame status: 0x%X", status))
	}

	C.bridge_context_reset(video.hw.context_reset)
}

//export deinitVideo
func deinitVideo() {
	C.bridge_context_reset(video.hw.context_destroy)
	if video.hw.depth {
		gl.DeleteRenderbuffers(1, &video.rbo);
	}
	gl.DeleteFramebuffers(1, &video.fbo)
	gl.DeleteTextures(1, &video.tex)
	// In OSX 10.14+ window deletion must happen in the main thread
	mainthread.Call(func() {
		video.window.GLMakeCurrent(video.context)
		sdl.GLDeleteContext(video.context)
		video.window.Destroy()
	})
	video.isGl = false
}

var retroHandle unsafe.Pointer
var retroInit unsafe.Pointer
var retroDeinit unsafe.Pointer
var retroAPIVersion unsafe.Pointer
var retroGetSystemInfo unsafe.Pointer
var retroGetSystemAVInfo unsafe.Pointer
var retroSetEnvironment unsafe.Pointer
var retroSetVideoRefresh unsafe.Pointer
var retroSetInputPoll unsafe.Pointer
var retroSetInputState unsafe.Pointer
var retroSetAudioSample unsafe.Pointer
var retroSetAudioSampleBatch unsafe.Pointer
var retroRun unsafe.Pointer
var retroLoadGame unsafe.Pointer
var retroUnloadGame unsafe.Pointer
var retroGetMemorySize unsafe.Pointer
var retroGetMemoryData unsafe.Pointer
var retroSerializeSize unsafe.Pointer
var retroSerialize unsafe.Pointer
var retroUnserialize unsafe.Pointer
var retroSetControllerPortDevice unsafe.Pointer

func loadFunction(handle unsafe.Pointer, name string) unsafe.Pointer {
	cs := C.CString(name)
	pointer := C.dlsym(handle, cs)
	C.free(unsafe.Pointer(cs))
	return pointer
}

func coreLoad(meta config.EmulatorMeta) {
	isGlAllowed = meta.IsGlAllowed
	usesLibCo = meta.UsesLibCo
	coreConfig = ScanConfigFile(meta.Config)

	multitap.supported = meta.HasMultitap
	multitap.enabled = false
	multitap.value = 0

	mu.Lock()
	// Different OS requires different library, bruteforce till it finish
	for _, ext := range config.EmulatorExtension {
		pathWithExt := meta.Path + ext
		cs := C.CString(pathWithExt)
		retroHandle = C.dlopen(cs, C.RTLD_LAZY)
		C.free(unsafe.Pointer(cs))
		if retroHandle != nil {
			break
		}
	}

	if retroHandle == nil {
		err := C.dlerror()
		log.Fatalf("error loading %s, err %+v", meta.Path, *err)
	}

	retroInit = loadFunction(retroHandle, "retro_init")
	retroDeinit = loadFunction(retroHandle, "retro_deinit")
	retroAPIVersion = loadFunction(retroHandle, "retro_api_version")
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

	mu.Unlock()

	C.bridge_retro_set_environment(retroSetEnvironment, C.coreEnvironment_cgo)
	C.bridge_retro_set_video_refresh(retroSetVideoRefresh, C.coreVideoRefresh_cgo)
	C.bridge_retro_set_input_poll(retroSetInputPoll, C.coreInputPoll_cgo)
	C.bridge_retro_set_input_state(retroSetInputState, C.coreInputState_cgo)
	C.bridge_retro_set_audio_sample(retroSetAudioSample, C.coreAudioSample_cgo)
	C.bridge_retro_set_audio_sample_batch(retroSetAudioSampleBatch, C.coreAudioSampleBatch_cgo)

	C.bridge_retro_init(retroInit)

	v := C.bridge_retro_api_version(retroAPIVersion)
	fmt.Println("Libretro API version:", v)
}

func slurp(path string, size int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes := make([]byte, size)
	buffer := bufio.NewReader(f)
	_, err = buffer.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func coreLoadGame(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	size := fi.Size()

	fmt.Println("ROM size:", size)

	csFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(csFilename))
	gi := C.struct_retro_game_info{
		path: csFilename,
		size: C.size_t(size),
	}

	si := C.struct_retro_system_info{}

	C.bridge_retro_get_system_info(retroGetSystemInfo, &si)

	var libName = C.GoString(si.library_name)
	fmt.Println("  library_name:", libName)
	fmt.Println("  library_version:", C.GoString(si.library_version))
	fmt.Println("  valid_extensions:", C.GoString(si.valid_extensions))
	fmt.Println("  need_fullpath:", si.need_fullpath)
	fmt.Println("  block_extract:", si.block_extract)

	if !si.need_fullpath {
		bytes, err := slurp(filename, size)
		if err != nil {
			panic(err)
		}
		cstr := C.CString(string(bytes))
		defer C.free(unsafe.Pointer(cstr))
		gi.data = unsafe.Pointer(cstr)
	}

	ok := C.bridge_retro_load_game(retroLoadGame, &gi)
	if !ok {
		log.Fatal("The core failed to load the content.")
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

	fmt.Println("-----------------------------------")
	fmt.Println("--- System audio and video info ---")
	fmt.Println("-----------------------------------")
	fmt.Println("  Aspect ratio: ", ratio)
	fmt.Println("  Base width: ", avi.geometry.base_width)   /* Nominal video width of game. */
	fmt.Println("  Base height: ", avi.geometry.base_height) /* Nominal video height of game. */
	fmt.Println("  Max width: ", avi.geometry.max_width)     /* Maximum possible width of game. */
	fmt.Println("  Max height: ", avi.geometry.max_height)   /* Maximum possible height of game. */
	fmt.Println("  Sample rate: ", avi.timing.sample_rate)   /* Sampling rate of audio. */
	fmt.Println("  FPS: ", avi.timing.fps)                   /* FPS of video content. */
	fmt.Println("-----------------------------------")

	video.max_width   = int32(avi.geometry.max_width)
	video.max_height  = int32(avi.geometry.max_height)
	video.base_width  = int32(avi.geometry.base_width)
	video.base_height = int32(avi.geometry.base_height)
	if video.isGl {
		if usesLibCo {
			C.bridge_execute(C.initVideo_cgo)
		} else {
			runtime.LockOSThread()
			initVideo()
			runtime.UnlockOSThread()
		}
	}
}

func toggleMultitap() {
	if (multitap.supported && multitap.value != 0) {
		// Official SNES games only support a single multitap device
		// Most require it to be plugged in player 2 port
		// And Snes9X requires it to be "plugged" after the game is loaded
		// Control this from the browser since player 2 will stop working in some games if multitap is "plugged" in
		if (multitap.enabled) {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, C.RETRO_DEVICE_JOYPAD)
		} else {
			C.bridge_retro_set_controller_port_device(retroSetControllerPortDevice, 1, multitap.value)
		}
		multitap.enabled = !multitap.enabled
	}
}

// serializeSize returns the amount of data the implementation requires to serialize
// internal state (save states).
// Between calls to retro_load_game() and retro_unload_game(), the
// returned size is never allowed to be larger than a previous returned
// value, to ensure that the frontend can allocate a save state buffer once.
func serializeSize() uint {
	return uint(C.bridge_retro_serialize_size(retroSerializeSize))
}

// Serializes internal state and returns the state as a byte slice.
func serialize(size uint) ([]byte, error) {
	data := C.malloc(C.size_t(size))
	defer C.free(data)

	ok := bool(C.bridge_retro_serialize(retroSerialize, data, C.size_t(size)))
	if !ok {
		return nil, errors.New("retro_serialize failed")
	}

	bytes := C.GoBytes(data, C.int(size))
	return bytes, nil
}

// unserialize unserializes internal state from a byte slice.
func unserialize(bytes []byte, size uint) error {
	if len(bytes) == 0 {
		return nil
	}
	ok := bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&bytes[0]), C.size_t(size)))
	if !ok {
		return errors.New("retro_unserialize failed")
	}
	return nil
}

func nanoarchShutdown() {
	if usesLibCo {
		C.bridge_execute(retroUnloadGame)
		C.bridge_execute(retroDeinit)
		if video.isGl {
			C.bridge_execute(C.deinitVideo_cgo)
		}
	} else {
		if video.isGl {
			// running inside a go routine, lock the thread to make sure the OpenGL context stays current
			runtime.LockOSThread()
			video.window.GLMakeCurrent(video.context)
		}
		C.bridge_retro_unload_game(retroUnloadGame)
		C.bridge_retro_deinit(retroDeinit)
		if video.isGl {
			deinitVideo()
			runtime.UnlockOSThread()
		}
	}

	setRotation(0)
	if r := C.dlclose(retroHandle); r != 0 {
		fmt.Println("error closing core")
	}
	for _, element := range coreConfig {
		C.free(unsafe.Pointer(element))
	}
}

func nanoarchRun() {
	if usesLibCo {
		C.bridge_execute(retroRun)
	} else {
		if video.isGl {
			// running inside a go routine, lock the thread to make sure the OpenGL context stays current
			runtime.LockOSThread()
			video.window.GLMakeCurrent(video.context)
		}
		C.bridge_retro_run(retroRun)
		if video.isGl {
			runtime.UnlockOSThread()
		}
	}
}

func videoSetPixelFormat(format uint32) C.bool {
	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		video.pixFmt = image.BIT_FORMAT_SHORT_5_5_5_1
		video.bpp = 2
		// format is not implemented
		pixelFormatConverterFn = nil
		break
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		video.pixFmt = image.BIT_FORMAT_INT_8_8_8_8_REV
		video.bpp = 4
		pixelFormatConverterFn = image.Rgba8888
		break
	case C.RETRO_PIXEL_FORMAT_RGB565:
		video.pixFmt = image.BIT_FORMAT_SHORT_5_6_5
		video.bpp = 2
		pixelFormatConverterFn = image.Rgb565
		break
	default:
		log.Fatalf("Unknown pixel type %v", format)
	}

	fmt.Printf("Video pixel: %v %v %v %v %v\n", video, format, C.RETRO_PIXEL_FORMAT_0RGB1555, C.RETRO_PIXEL_FORMAT_XRGB8888, C.RETRO_PIXEL_FORMAT_RGB565)
	return true
}

func setRotation(rotation int) {
	video.rotation = image.Angle(rotation)
	rotationFn = image.GetRotation(video.rotation)
	NAEmulator.meta.Rotation = rotationFn
		log.Printf("[Env]: the game video is rotated %v°", map[int]int{0: 0, 1: 90, 2: 180, 3: 270}[rotation])
}
