package nanoarch

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"os/user"
	"reflect"
	"sync"
	"unsafe"

	"github.com/giongto35/cloud-game/emulator"
	"github.com/go-gl/gl/v2.1/gl"
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

bool coreEnvironment_cgo(unsigned cmd, void *data);
void coreVideoRefresh_cgo(void *data, unsigned width, unsigned height, size_t pitch);
void coreInputPoll_cgo();
void coreAudioSample_cgo(int16_t left, int16_t right);
size_t coreAudioSampleBatch_cgo(const int16_t *data, size_t frames);
int16_t coreInputState_cgo(unsigned port, unsigned device, unsigned index, unsigned id);
void coreLog_cgo(enum retro_log_level level, const char *msg);
*/
import "C"

var mu sync.Mutex

var video struct {
	program uint32
	vao     uint32
	texID   uint32
	pitch   uint32
	pixFmt  uint32
	pixType uint32
	bpp     uint32
}

var scale = 3.0

const bufSize = 1024 * 4

const joypadNumKeys = C.RETRO_DEVICE_ID_JOYPAD_R3 + 1

var joy [joypadNumKeys]bool
var ewidth, eheight int

var bindRetroKeys = map[int]int{
	0: C.RETRO_DEVICE_ID_JOYPAD_A,
	1: C.RETRO_DEVICE_ID_JOYPAD_B,
	2: C.RETRO_DEVICE_ID_JOYPAD_X,
	3: C.RETRO_DEVICE_ID_JOYPAD_Y,
	4: C.RETRO_DEVICE_ID_JOYPAD_SELECT,
	5: C.RETRO_DEVICE_ID_JOYPAD_START,
	6: C.RETRO_DEVICE_ID_JOYPAD_UP,
	7: C.RETRO_DEVICE_ID_JOYPAD_DOWN,
	8: C.RETRO_DEVICE_ID_JOYPAD_LEFT,
	9: C.RETRO_DEVICE_ID_JOYPAD_RIGHT,
}

type CloudEmulator interface {
	SetView(view *emulator.GameView)
	Start(path string)
	SaveGame(saveExtraFunc func() error) error
	LoadGame() error
	GetHashPath() string
	Close()
}

func resizeToAspect(ratio float64, sw float64, sh float64) (dw float64, dh float64) {
	if ratio <= 0 {
		ratio = sw / sh
	}

	if sw/sh < 1.0 {
		dw = dh * ratio
		dh = sh
	} else {
		dw = sw
		dh = dw / ratio
	}
	return
}

func videoConfigure(geom *C.struct_retro_game_geometry) (int, int) {

	nwidth, nheight := resizeToAspect(float64(geom.aspect_ratio), float64(geom.base_width), float64(geom.base_height))

	fmt.Println("media config", nwidth, nheight, geom.base_width, geom.base_height, geom.aspect_ratio, video.bpp, scale)

	if video.texID == 0 {
		fmt.Println("Failed to create the video texture")
	}

	return int(math.Round(nwidth)), int(math.Round(nheight))
}

//export coreVideoRefresh
func coreVideoRefresh(data unsafe.Pointer, width C.unsigned, height C.unsigned, pitch C.size_t) {
	bytesPerRow := int(uint32(pitch) / video.bpp)

	if data != nil {
		NAEmulator.imageChannel <- toImageRGBA(data, bytesPerRow)
	}
}

// toImageRGBA convert nanoarch 2d array to image.RGBA
func toImageRGBA(data unsafe.Pointer, bytesPerRow int) *image.RGBA {
	// Convert unsafe Pointer to bytes array
	var bytes []byte

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	sh.Data = uintptr(data)
	sh.Len = bytesPerRow * eheight * 4
	sh.Cap = bytesPerRow * eheight * 4

	if video.pixFmt == gl.UNSIGNED_SHORT_5_6_5 {
		return to565Image(data, bytes, bytesPerRow)
	} else if video.pixFmt == gl.UNSIGNED_INT_8_8_8_8_REV {
		return to8888Image(data, bytes, bytesPerRow)
	}
	return nil
}

func to8888Image(data unsafe.Pointer, bytes []byte, bytesPerRow int) *image.RGBA {
	seek := 0

	// Convert bytes array to image
	// TODO: Reduce overhead of copying to bytes array by accessing unsafe.Pointer directly
	image := image.NewRGBA(image.Rect(0, 0, ewidth, eheight))
	for y := 0; y < eheight; y++ {
		for x := 0; x < bytesPerRow; x++ {
			if x < ewidth {
				b8 := bytes[seek]
				g8 := bytes[seek+1]
				r8 := bytes[seek+2]
				a8 := bytes[seek+3]

				image.Set(x, y, color.RGBA{byte(r8), byte(g8), byte(b8), byte(a8)})
			}
			seek += 4
		}
	}

	// TODO: Resize Image
	return image
}

func to565Image(data unsafe.Pointer, bytes []byte, bytesPerRow int) *image.RGBA {
	seek := 0

	// Convert bytes array to image
	// TODO: Reduce overhead of copying to bytes array by accessing unsafe.Pointer directly
	image := image.NewRGBA(image.Rect(0, 0, ewidth, eheight))
	for y := 0; y < eheight; y++ {
		for x := 0; x < bytesPerRow; x++ {
			if x < ewidth {
				var bi int
				bi = (int)(bytes[seek]) + ((int)(bytes[seek+1]) << 8)
				b5 := bi & 0x1F
				g6 := (bi >> 5) & 0x3F
				r5 := (bi >> 11)

				b8 := (b5*255 + 15) / 31
				g8 := (g6*255 + 31) / 63
				r8 := (r5*255 + 15) / 31

				image.Set(x, y, color.RGBA{byte(r8), byte(g8), byte(b8), 255})
			}
			seek += 2
		}
	}

	// TODO: Resize Image
	return image
}

//export coreInputPoll
func coreInputPoll() {
	//for i := range NAEmulator.keys {
	//joy[i] = NAEmulator.keys[i]
	//}
}

//export coreInputState
func coreInputState(port C.unsigned, device C.unsigned, index C.unsigned, id C.unsigned) C.int16_t {
	if port > 0 || index > 0 || device != C.RETRO_DEVICE_JOYPAD {
		return 0
	}

	if id < 255 && NAEmulator.keys[id] {
		return 1
	}
	return 0
}

func min(a, b C.size_t) C.size_t {
	if a < b {
		return a
	}
	return b
}

func audioWrite2(buf unsafe.Pointer, frames C.size_t) C.size_t {
	numFrames := int(frames) * 2
	pcm := (*[1 << 30]int16)(unsafe.Pointer(buf))[:numFrames:numFrames]

	for i := 0; i < numFrames; i += 1 {
		s := float32(pcm[i])
		NAEmulator.audioChannel <- s
	}

	return 2 * frames
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

//export coreEnvironment
func coreEnvironment(cmd C.unsigned, data unsafe.Pointer) C.bool {
	switch cmd {
	case C.RETRO_ENVIRONMENT_GET_USERNAME:
		username := (**C.char)(data)
		currentUser, err := user.Current()
		if err != nil {
			*username = C.CString("")
		} else {
			*username = C.CString(currentUser.Username)
		}
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
		*path = C.CString("./libretro/system")
		return true
	case C.RETRO_ENVIRONMENT_GET_SAVE_DIRECTORY:
		path := (**C.char)(data)
		*path = C.CString(".")
		return true
	case C.RETRO_ENVIRONMENT_SHUTDOWN:
		//window.SetShouldClose(true)
		return true
	case C.RETRO_ENVIRONMENT_GET_VARIABLE:
		variable := (*C.struct_retro_variable)(data)
		fmt.Println("[Env]: get variable:", C.GoString(variable.key))
		return false
	default:
		//fmt.Println("[Env]: command not implemented", cmd)
		return false
	}
	return true
}

func init() {
}

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

func coreLoad(sofile string) {

	mu.Lock()
	h := C.dlopen(C.CString(sofile), C.RTLD_NOW)
	if h == nil {
		log.Fatalf("error loading %s\n", sofile)
	}

	retroInit = C.dlsym(h, C.CString("retro_init"))
	retroDeinit = C.dlsym(h, C.CString("retro_deinit"))
	retroAPIVersion = C.dlsym(h, C.CString("retro_api_version"))
	retroGetSystemInfo = C.dlsym(h, C.CString("retro_get_system_info"))
	retroGetSystemAVInfo = C.dlsym(h, C.CString("retro_get_system_av_info"))
	retroSetEnvironment = C.dlsym(h, C.CString("retro_set_environment"))
	retroSetVideoRefresh = C.dlsym(h, C.CString("retro_set_video_refresh"))
	retroSetInputPoll = C.dlsym(h, C.CString("retro_set_input_poll"))
	retroSetInputState = C.dlsym(h, C.CString("retro_set_input_state"))
	retroSetAudioSample = C.dlsym(h, C.CString("retro_set_audio_sample"))
	retroSetAudioSampleBatch = C.dlsym(h, C.CString("retro_set_audio_sample_batch"))
	retroRun = C.dlsym(h, C.CString("retro_run"))
	retroLoadGame = C.dlsym(h, C.CString("retro_load_game"))
	retroUnloadGame = C.dlsym(h, C.CString("retro_unload_game"))
	retroSerializeSize = C.dlsym(h, C.CString("retro_serialize_size"))
	retroSerialize = C.dlsym(h, C.CString("retro_serialize"))
	retroUnserialize = C.dlsym(h, C.CString("retro_unserialize"))

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

	gi := C.struct_retro_game_info{
		path: C.CString(filename),
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
		gi.data = unsafe.Pointer(cstr)

	}

	ok := C.bridge_retro_load_game(retroLoadGame, &gi)
	if !ok {
		log.Fatal("The core failed to load the content.")
	}

	avi := C.struct_retro_system_av_info{}

	C.bridge_retro_get_system_av_info(retroGetSystemAVInfo, &avi)

	ewidth, eheight = videoConfigure(&avi.geometry)
	// Append the library name to the window title.
	NAEmulator.meta.AudioSampleRate = int(avi.timing.sample_rate)
	NAEmulator.meta.Fps = int(avi.timing.fps)
	NAEmulator.meta.Width = ewidth
	NAEmulator.meta.Height = eheight
}

// serializeSize returns the amount of data the implementation requires to serialize
// internal state (save states).
// Between calls to retro_load_game() and retro_unload_game(), the
// returned size is never allowed to be larger than a previous returned
// value, to ensure that the frontend can allocate a save state buffer once.
func serializeSize() uint {
	return uint(C.bridge_retro_serialize_size(retroSerializeSize))
}

// serialize serializes internal state and returns the state as a byte slice.
func serialize(size uint) ([]byte, error) {
	data := C.malloc(C.size_t(size))
	ok := bool(C.bridge_retro_serialize(retroSerialize, data, C.size_t(size)))
	if !ok {
		return nil, errors.New("retro_serialize failed")
	}
	bytes := C.GoBytes(data, C.int(size))
	return bytes, nil
}

// unserialize unserializes internal state from a byte slice.
func unserialize(bytes []byte, size uint) error {
	ok := bool(C.bridge_retro_unserialize(retroUnserialize, unsafe.Pointer(&bytes[0]), C.size_t(size)))
	if !ok {
		return errors.New("retro_unserialize failed")
	}
	return nil
}

func nanoarchShutdown() {
	C.bridge_retro_unload_game(retroUnloadGame)
	C.bridge_retro_deinit(retroDeinit)
}

func nanoarchRun() {
	C.bridge_retro_run(retroRun)
}

func videoSetPixelFormat(format uint32) C.bool {
	if video.texID != 0 {
		log.Fatal("Tried to change pixel format after initialization.")
	}

	switch format {
	case C.RETRO_PIXEL_FORMAT_0RGB1555:
		video.pixFmt = gl.UNSIGNED_SHORT_5_5_5_1
		video.pixType = gl.BGRA
		video.bpp = 2
		break
	case C.RETRO_PIXEL_FORMAT_XRGB8888:
		video.pixFmt = gl.UNSIGNED_INT_8_8_8_8_REV
		video.pixType = gl.BGRA
		video.bpp = 4
		break
	case C.RETRO_PIXEL_FORMAT_RGB565:
		video.pixFmt = gl.UNSIGNED_SHORT_5_6_5
		video.pixType = gl.RGB
		video.bpp = 2
		break
	default:
		log.Fatalf("Unknown pixel type %v", format)
	}

	fmt.Printf("Video pixel: %v %v %v %v %v", video, format, C.RETRO_PIXEL_FORMAT_0RGB1555, C.RETRO_PIXEL_FORMAT_XRGB8888, C.RETRO_PIXEL_FORMAT_RGB565)
	return true
}
