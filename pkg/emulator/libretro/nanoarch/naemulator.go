package nanoarch

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image"
	"log"
	"net"
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

const numAxes = 4

type constrollerState struct {
	keyState uint16
	axes     [numAxes]int16
}

// naEmulator implements CloudEmulator
type naEmulator struct {
	imageChannel  chan<- GameFrame
	audioChannel  chan<- []int16
	inputChannel  <-chan InputEvent
	videoExporter *VideoExporter

	meta            config.EmulatorMeta
	gamePath        string
	roomID          string
	gameName        string
	isSavingLoading bool

	controllersMap map[string][]constrollerState
	done           chan struct{}

	// lock to lock uninteruptable operation
	lock *sync.Mutex
}

// VideoExporter produces image frame to unix socket
type VideoExporter struct {
	sock         net.Conn
	imageChannel chan<- GameFrame
}

type InputEvent struct {
	RawState  []byte
	PlayerIdx int
	ConnID    string
}

// GameFrame contains image and timeframe
type GameFrame struct {
	Image     *image.RGBA
	Timestamp uint32
}

var NAEmulator *naEmulator
var outputImg *image.RGBA

const maxPort = 8

const SocketAddrTmpl = "/tmp/cloudretro-retro-%s.sock"

// NAEmulator implements CloudEmulator interface based on NanoArch(golang RetroArch)
func NewNAEmulator(etype string, roomID string, inputChannel <-chan InputEvent) (*naEmulator, chan GameFrame, chan []int16) {
	meta := config.EmulatorConfig[etype]
	imageChannel := make(chan GameFrame, 30)
	audioChannel := make(chan []int16, 30)

	return &naEmulator{
		meta:           meta,
		imageChannel:   imageChannel,
		audioChannel:   audioChannel,
		inputChannel:   inputChannel,
		controllersMap: map[string][]constrollerState{},
		roomID:         roomID,
		done:           make(chan struct{}, 1),
		lock:           &sync.Mutex{},
	}, imageChannel, audioChannel
}

// NewVideoExporter creates new video Exporter that produces to unix socket
func NewVideoExporter(roomID string, imgChannel chan GameFrame) *VideoExporter {
	sockAddr := fmt.Sprintf(SocketAddrTmpl, roomID)

	go func(sockAddr string) {
		log.Println("Dialing to ", sockAddr)
		conn, err := net.Dial("unix", sockAddr)
		if err != nil {
			log.Fatal("accept error: ", err)
		}

		defer conn.Close()

		for img := range imgChannel {
			reqBodyBytes := new(bytes.Buffer)
			gob.NewEncoder(reqBodyBytes).Encode(img)
			//fmt.Printf("%+v %+v %+v \n", img.Image.Stride, img.Image.Rect.Max.X, len(img.Image.Pix))
			// conn.Write(img.Image.Pix)
			b := reqBodyBytes.Bytes()
			fmt.Printf("Bytes %d\n", len(b))
			conn.Write(b)
		}
	}(sockAddr)

	return &VideoExporter{
		imageChannel: imgChannel,
	}

}

// Init initialize new RetroArch cloud emulator
// withImageChan returns an image stream as Channel for output else it will write to unix socket
func Init(etype string, roomID string, withImageChannel bool, inputChannel <-chan InputEvent) (*naEmulator, chan GameFrame, chan []int16) {
	emulator, imageChannel, audioChannel := NewNAEmulator(etype, roomID, inputChannel)
	// Set to global NAEmulator
	NAEmulator = emulator
	if !withImageChannel {
		NAEmulator.videoExporter = NewVideoExporter(roomID, imageChannel)
	}

	go NAEmulator.listenInput()

	return emulator, imageChannel, audioChannel
}

func (na *naEmulator) listenInput() {
	// input from javascript follows bitmap. Ex: 00110101
	// we decode the bitmap and send to channel
	for inpEvent := range NAEmulator.inputChannel {
		inpBitmap := uint16(inpEvent.RawState[1])<<8 + uint16(inpEvent.RawState[0])

		if inpBitmap == 0xFFFF {
			// terminated
			delete(na.controllersMap, inpEvent.ConnID)
			continue
		}

		if _, ok := na.controllersMap[inpEvent.ConnID]; !ok {
			na.controllersMap[inpEvent.ConnID] = make([]constrollerState, maxPort)
		}

		na.controllersMap[inpEvent.ConnID][inpEvent.PlayerIdx].keyState = inpBitmap
		for i := 0; i < numAxes && (i+1)*2+1 < len(inpEvent.RawState); i++ {
			na.controllersMap[inpEvent.ConnID][inpEvent.PlayerIdx].axes[i] = int16(inpEvent.RawState[(i+1)*2+1])<<8 + int16(inpEvent.RawState[(i+1)*2])
		}
	}
}

func (na *naEmulator) LoadMeta(path string) config.EmulatorMeta {
	coreLoad(na.meta)
	coreLoadGame(path)
	na.gamePath = path

	return na.meta
}

func (na *naEmulator) SetViewport(width int, height int) {
	// outputImg is tmp img used for decoding and reuse in encoding flow
	outputImg = image.NewRGBA(image.Rect(0, 0, width, height))
}

func (na *naEmulator) Start() {
	na.playGame(na.gamePath)
	ticker := time.NewTicker(time.Second / time.Duration(na.meta.Fps))

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

func (na *naEmulator) ToggleMultitap() error {
	if na.roomID != "" {
		toggleMultitap()
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
