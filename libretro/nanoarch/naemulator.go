package nanoarch

import (
	"image"
	"log"

	"github.com/go-gl/glfw/v3.2/glfw"
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

bool coreEnvironment_cgo(unsigned cmd, void *data);
void coreVideoRefresh_cgo(void *data, unsigned width, unsigned height, size_t pitch);
void coreInputPoll_cgo();
void coreAudioSample_cgo(int16_t left, int16_t right);
size_t coreAudioSampleBatch_cgo(const int16_t *data, size_t frames);
int16_t coreInputState_cgo(unsigned port, unsigned device, unsigned index, unsigned id);
void coreLog_cgo(enum retro_log_level level, const char *msg);
*/
import "C"

// naEmulator implements CloudEmulator
type naEmulator struct {
	imageChannel chan<- *image.RGBA
	audioChannel chan<- float32
	inputChannel <-chan int
	corePath     string
	gamePath     string
	roomID       string
}

var NAEmulator *naEmulator

func NewNAEmulator(imageChannel chan<- *image.RGBA, inputChannel <-chan int) *naEmulator {
	return &naEmulator{
		//corePath:     "libretro/cores/pcsx_rearmed_libretro.so",
		corePath:     "libretro/cores/mgba_libretro.so",
		imageChannel: imageChannel,
		inputChannel: inputChannel,
	}
}

func Init(imageChannel chan<- *image.RGBA, inputChannel <-chan int) {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	NAEmulator = NewNAEmulator(imageChannel, inputChannel)
}

func (na *naEmulator) Start(path string) {
	coreLoad(na.corePath)
	na.playGame(path)

	for !window.ShouldClose() {
		glfw.PollEvents()

		C.bridge_retro_run(retroRun)

		//gl.Clear(gl.COLOR_BUFFER_BIT)

		videoRender()

		window.SwapBuffers()
	}
}

func (na *naEmulator) playGame(path string) {
	coreLoadGame(path)
}

func (na *naEmulator) SaveGame(saveExtraFunc func() error) error {
	return nil
}

func (na *naEmulator) LoadGame() error {
	return nil
}

func (na *naEmulator) GetHashPath() string {
	return savePath(na.roomID)
}

func savePath(hash string) string {
	//return homeDir + "/.nes/save/" + hash + ".dat"
	return ""
}

func (na *naEmulator) Close() {
	// Unload and deinit in the core.
	C.bridge_retro_unload_game(retroUnloadGame)
	C.bridge_retro_deinit(retroDeinit)
	glfw.Terminate()
}
