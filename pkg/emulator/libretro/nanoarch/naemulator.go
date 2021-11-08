package nanoarch

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image"
	"net"
	"sync"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
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

type naEmulator struct {
	sync.Mutex

	imageChannel  chan<- GameFrame
	audioChannel  chan<- []int16
	inputChannel  <-chan InputEvent
	videoExporter *VideoExporter

	meta            emulator.Metadata
	gamePath        string
	roomID          string
	gameName        string
	isSavingLoading bool
	storage         Storage

	players Players

	done chan struct{}
	log  *logger.Logger
}

// VideoExporter produces image frame to unix socket
type VideoExporter struct {
	sock         net.Conn
	imageChannel chan<- GameFrame
}

// GameFrame contains image and timeframe
type GameFrame struct {
	Image     *image.RGBA
	Timestamp uint32
}

var NAEmulator *naEmulator
var outputImg *image.RGBA

// NewNAEmulator implements CloudEmulator interface for a Libretro frontend.
func NewNAEmulator(roomID string, inputChannel <-chan InputEvent, storage Storage, conf config.LibretroCoreConfig, log *logger.Logger) (*naEmulator, chan GameFrame, chan []int16) {
	imageChannel := make(chan GameFrame, 30)
	audioChannel := make(chan []int16, 30)

	SetLibretroLogger(log)

	return &naEmulator{
		meta: emulator.Metadata{
			LibPath:       conf.Lib,
			ConfigPath:    conf.Config,
			Ratio:         conf.Ratio,
			IsGlAllowed:   conf.IsGlAllowed,
			UsesLibCo:     conf.UsesLibCo,
			HasMultitap:   conf.HasMultitap,
			AutoGlContext: conf.AutoGlContext,
		},
		storage:      storage,
		imageChannel: imageChannel,
		audioChannel: audioChannel,
		inputChannel: inputChannel,
		players:      NewPlayerSessionInput(),
		roomID:       roomID,
		done:         make(chan struct{}, 1),
		log:          log,
	}, imageChannel, audioChannel
}

// NewVideoExporter creates new video Exporter that produces to unix socket
func NewVideoExporter(roomID string, imgChannel chan GameFrame, log *logger.Logger) *VideoExporter {
	sockAddr := fmt.Sprintf("/tmp/cloudretro-retro-%s.sock", roomID)
	go func(sockAddr string) {
		log.Info().Msgf("Dialing to %v", sockAddr)
		conn, err := net.Dial("unix", sockAddr)
		if err != nil {
			log.Panic().Err(err)
		}

		defer conn.Close()

		for img := range imgChannel {
			reqBodyBytes := new(bytes.Buffer)
			_ = gob.NewEncoder(reqBodyBytes).Encode(img)
			b := reqBodyBytes.Bytes()
			_, _ = conn.Write(b)
		}
	}(sockAddr)
	return &VideoExporter{imageChannel: imgChannel}
}

// Init initialize new RetroArch cloud emulator
// withImageChan returns an image stream as Channel for output else it will write to unix socket
func Init(roomID string, withImageChannel bool, inputChannel <-chan InputEvent, storage Storage, config config.LibretroCoreConfig, log *logger.Logger) (*naEmulator, chan GameFrame, chan []int16) {
	emu, imageChannel, audioChannel := NewNAEmulator(roomID, inputChannel, storage, config, log)
	// Set to global NAEmulator
	NAEmulator = emu
	if !withImageChannel {
		NAEmulator.videoExporter = NewVideoExporter(roomID, imageChannel, log)
	}

	go NAEmulator.listenInput()

	return emu, imageChannel, audioChannel
}

// listenInput handles user input.
// The user input is encoded as bitmap that we decode
// and send into the game emulator.
func (na *naEmulator) listenInput() {
	for in := range NAEmulator.inputChannel {
		bitmap := in.bitmap()
		if bitmap == InputTerminate {
			na.players.session.close(in.ConnID)
			continue
		}
		na.players.session.setInput(in.ConnID, in.PlayerIdx, bitmap, in.RawState)
	}
}

func (na *naEmulator) LoadMeta(path string) emulator.Metadata {
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
	if err := na.LoadGame(); err != nil {
		na.log.Error().Err(err).Msg("couldn't load a save file")
	}

	ticker := time.NewTicker(time.Second / time.Duration(na.meta.Fps))

	for range ticker.C {
		select {
		// Slow response here
		case <-na.done:
			nanoarchShutdown()
			close(na.imageChannel)
			close(na.audioChannel)
			na.log.Debug().Msg("Closed Director")
			return
		default:
		}

		na.Lock()
		nanoarchRun()
		na.Unlock()
	}
}

func (na *naEmulator) SaveGame() error {
	// !to fix
	if usesLibCo {
		return nil
	}
	if na.roomID != "" {
		return na.Save()
	}
	return nil
}

func (na *naEmulator) LoadGame() error {
	// !to fix
	if usesLibCo {
		return nil
	}
	if na.roomID != "" {
		return na.Load()
	}
	return nil
}

func (na *naEmulator) ToggleMultitap() error {
	if na.roomID != "" {
		toggleMultitap()
	}
	return nil
}

func (na *naEmulator) GetHashPath() string { return na.storage.GetSavePath() }

func (na *naEmulator) GetSRAMPath() string { return na.storage.GetSRAMPath() }

func (*naEmulator) GetViewport() interface{} { return outputImg }

func (na *naEmulator) Close() { close(na.done) }
