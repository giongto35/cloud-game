package nanoarch

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/util"
)

/*
#include "libretro.h"
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <dlfcn.h>
#include <string.h>

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

// naEmulator implements CloudEmulator
type naEmulator struct {
	imageChannel chan<- *image.RGBA
	audioChannel chan<- []int16
	inputChannel <-chan InputEvent

	meta            config.EmulatorMeta
	gamePath        string
	roomID          string
	gameName        string
	isSavingLoading bool

	keys []bool
	done chan struct{}

	// lock to lock uninteruptable operation
	lock *sync.Mutex
}

type InputEvent struct {
	KeyState  int
	PlayerIdx int
}

var NAEmulator *naEmulator
var outputImg *image.RGBA

// NAEmulator implements CloudEmulator interface based on NanoArch(golang RetroArch)
func NewNAEmulator(etype string, roomID string, inputChannel <-chan InputEvent) (*naEmulator, chan *image.RGBA, chan []int16) {
	meta := config.EmulatorConfig[etype]
	imageChannel := make(chan *image.RGBA, 30)
	audioChannel := make(chan []int16, 30)

	return &naEmulator{
		meta:         meta,
		imageChannel: imageChannel,
		audioChannel: audioChannel,
		inputChannel: inputChannel,
		keys:         make([]bool, joypadNumKeys*4),
		roomID:       roomID,
		done:         make(chan struct{}, 1),
		lock:         &sync.Mutex{},
	}, imageChannel, audioChannel
}

// Init initialize new RetroArch cloud emulator
func Init(etype string, roomID string, inputChannel <-chan InputEvent) (*naEmulator, chan *image.RGBA, chan []int16) {
	emulator, imageChannel, audioChannel := NewNAEmulator(etype, roomID, inputChannel)
	// Set to global NAEmulator
	NAEmulator = emulator

	go NAEmulator.listenInput()
	return emulator, imageChannel, audioChannel
}

func (na *naEmulator) listenInput() {
	// input from javascript follows bitmap. Ex: 00110101
	// we decode the bitmap and send to channel
	for inpEvent := range NAEmulator.inputChannel {
		inpBitmap := inpEvent.KeyState

		for k := 0; k < len(bindRetroKeys); k++ {
			key, ok := bindRetroKeys[k]
			if ok == false {
				continue
			}

			if (inpBitmap & 1) == 1 {
				na.keys[key*4+inpEvent.PlayerIdx] = true
			} else {
				na.keys[key*4+inpEvent.PlayerIdx] = false
			}
			inpBitmap >>= 1
		}
	}
}

func (na *naEmulator) LoadMeta(path string) config.EmulatorMeta {
	coreLoad(na.meta.Path)
	coreLoadGame(path)
	na.gamePath = path

	return na.meta
}

func (na *naEmulator) SetViewport(width int, height int) {
	// outputImg is tmp img used for decoding and reuse in encoding flow
	outputImg = image.NewRGBA(image.Rect(0, 0, width, height))

	ewidth = width
	eheight = height
}

func (na *naEmulator) Start() {
	na.playGame(na.gamePath)
	ticker := time.NewTicker(time.Second / 60)

	for range ticker.C {
		select {
		// Slow response here
		case <-na.done:
			nanoarchShutdown()
			close(na.imageChannel)
			close(na.audioChannel)
			log.Println("Closed Director")
			return
		default:
		}

		na.GetLock()
		nanoarchRun()
		na.ReleaseLock()
	}
}

func (na *naEmulator) playGame(path string) {
	// When start game, we also try loading if there was a saved state
	na.LoadGame()
}

func (na *naEmulator) SaveGame(saveExtraFunc func() error) error {
	if na.roomID != "" {
		err := na.Save()
		if err != nil {
			return err
		}
		err = saveExtraFunc()
		if err != nil {
			return err
		}
	}

	return nil
}

func (na *naEmulator) LoadGame() error {
	if na.roomID != "" {
		err := na.Load()
		if err != nil {
			log.Println("Error: Cannot load", err)
			return err
		}
	}

	return nil
}

func (na *naEmulator) GetHashPath() string {
	return util.GetSavePath(na.roomID)
}

func (na *naEmulator) Close() {
	// Unload and deinit in the core.
	close(na.done)
}
