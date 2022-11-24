package nanoarch

import (
	"image"
	"sync"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type Frontend struct {
	imageChannel chan<- GameFrame
	audioChannel chan<- GameAudio
	inputChannel <-chan InputEvent

	meta            emulator.Metadata
	gamePath        string
	roomID          string
	gameName        string
	isSavingLoading bool
	storage         Storage

	// out frame size
	vw, vh int

	// draw threads
	th int

	players Players

	done chan struct{}
	log  *logger.Logger

	mu sync.Mutex
}

type (
	GameFrame struct {
		Data     *image.RGBA
		Duration time.Duration
	}
	GameAudio struct {
		Data     []int16
		Duration time.Duration
	}
)

// NewFrontend implements CloudEmulator interface for a Libretro frontend.
func NewFrontend(roomID string, inputChannel <-chan InputEvent, storage Storage, conf config.LibretroCoreConfig, threads int, log *logger.Logger) (*Frontend, chan GameFrame, chan GameAudio) {
	imageChannel := make(chan GameFrame, 30)
	audioChannel := make(chan GameAudio, 30)

	log = log.Extend(log.With().Str("[m]", "Libretro"))
	SetLibretroLogger(log)

	f := Frontend{
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
		th:           threads,
		log:          log,
	}

	// set global link to the Libretro
	frontend = &f

	go f.listenInput()

	return &f, imageChannel, audioChannel
}

// listenInput handles user input.
// The user input is encoded as bitmap that we decode
// and send into the game emulator.
func (f *Frontend) listenInput() {
	for {
		select {
		case <-f.done:
			return
		case in, ok := <-f.inputChannel:
			if !ok {
				return
			}
			bitmap := in.bitmap()
			if bitmap == InputTerminate {
				f.players.session.close(in.ConnID)
				continue
			}
			f.players.session.setInput(in.ConnID, in.PlayerIdx, bitmap, in.RawState)
		}
	}
}

func (f *Frontend) LoadMeta(path string) emulator.Metadata {
	coreLoad(f.meta)
	coreLoadGame(path)
	f.gamePath = path
	return f.meta
}

func (f *Frontend) SetViewport(width int, height int) { f.vw, f.vh = width, height }

func (f *Frontend) Start() {
	if err := f.LoadGame(); err != nil {
		f.log.Error().Err(err).Msg("couldn't load a save file")
	}

	framerate := 1 / f.meta.Fps
	f.log.Info().Msgf("framerate: %vms", framerate)
	ticker := time.NewTicker(time.Second / time.Duration(f.meta.Fps))
	defer ticker.Stop()

	lastFrameTime = time.Now().UnixNano()

	for {
		f.mu.Lock()
		nanoarchRun()
		f.mu.Unlock()

		select {
		case <-ticker.C:
			continue
		case <-f.done:
			nanoarchShutdown()
			close(f.imageChannel)
			close(f.audioChannel)
			f.log.Debug().Msg("Closed Director")
			return
		}
	}
}

func (f *Frontend) SaveGame() error { return f.Save() }

func (f *Frontend) LoadGame() error { return f.Load() }

func (f *Frontend) ToggleMultitap() { toggleMultitap() }

func (f *Frontend) GetHashPath() string { return f.storage.GetSavePath() }

func (f *Frontend) GetSRAMPath() string { return f.storage.GetSRAMPath() }

func (f *Frontend) Close() {
	f.mu.Lock()
	f.SetViewport(0, 0)
	f.mu.Unlock()
	close(f.done)
}
