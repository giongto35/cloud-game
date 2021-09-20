package nanoarch

import (
	"crypto/md5"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
)

type testRun struct {
	room           string
	system         string
	rom            string
	emulationTicks int
}

// EmulatorMock contains naEmulator mocking data.
type EmulatorMock struct {
	naEmulator

	// Libretro compiled lib core name
	core string
	// draw canvas instance
	canvas *image.RGBA
	// shared core paths (can't be changed)
	paths EmulatorPaths

	// channels
	imageInCh  <-chan GameFrame
	audioInCh  <-chan []int16
	inputOutCh chan<- InputEvent
}

// EmulatorPaths defines various emulator file paths.
type EmulatorPaths struct {
	assets string
	cores  string
	games  string
	save   string
}

// GetEmulatorMock returns a properly stubbed emulator instance.
// Due to extensive use of globals -- one mock instance is allowed per a test run.
// Don't forget to init one image channel consumer, it will lock-out otherwise.
// Make sure you call shutdownEmulator().
func GetEmulatorMock(room string, system string) *EmulatorMock {
	rootPath := getRootPath()
	configPath := rootPath + "configs/"

	var conf worker.Config
	if err := config.LoadConfig(&conf, configPath); err != nil {
		panic(err)
	}

	meta := conf.Emulator.GetLibretroCoreConfig(system)

	images := make(chan GameFrame, 30)
	audio := make(chan []int16, 30)
	inputs := make(chan InputEvent, 100)

	store := Storage{
		Path:     os.TempDir(),
		MainSave: room,
	}

	// an emu
	emu := &EmulatorMock{
		naEmulator: naEmulator{
			imageChannel: images,
			audioChannel: audio,
			inputChannel: inputs,
			storage:      store,

			meta: emulator.Metadata{
				LibPath:     meta.Lib,
				ConfigPath:  meta.Config,
				Ratio:       meta.Ratio,
				IsGlAllowed: meta.IsGlAllowed,
				UsesLibCo:   meta.UsesLibCo,
				HasMultitap: meta.HasMultitap,
			},
			players: NewPlayerSessionInput(),
			roomID:  room,
			done:    make(chan struct{}, 1),
		},

		canvas: image.NewRGBA(image.Rect(0, 0, meta.Width, meta.Height)),
		core:   path.Base(meta.Lib),

		paths: EmulatorPaths{
			assets: cleanPath(rootPath),
			cores:  cleanPath(rootPath + "assets/cores/"),
			games:  cleanPath(rootPath + "assets/games/"),
		},

		imageInCh:  images,
		audioInCh:  audio,
		inputOutCh: inputs,
	}

	// stub globals
	NAEmulator = &emu.naEmulator
	outputImg = emu.canvas

	emu.paths.save = cleanPath(emu.GetHashPath())

	return emu
}

// GetDefaultEmulatorMock returns initialized emulator mock with default params.
// Spawns audio/image channels consumers.
// Don't forget to close emulator mock with shutdownEmulator().
func GetDefaultEmulatorMock(room string, system string, rom string) *EmulatorMock {
	mock := GetEmulatorMock(room, system)
	mock.loadRom(rom)
	go mock.handleVideo(func(_ GameFrame) {})
	go mock.handleAudio(func(_ []int16) {})

	return mock
}

// loadRom loads a ROM into the emulator.
// The rom will be loaded from emulators' games path.
func (emu *EmulatorMock) loadRom(game string) {
	fmt.Printf("%v %v\n", emu.paths.cores, emu.core)
	coreLoad(emulator.Metadata{LibPath: emu.paths.cores + emu.core})
	coreLoadGame(emu.paths.games + game)

	if emu.canvas.Rect.Dx() == 0 || emu.canvas.Rect.Dy() == 0 {
		emu.canvas = image.NewRGBA(image.Rect(0, 0, emu.meta.BaseWidth, emu.meta.BaseHeight))
		outputImg = emu.canvas
	}
}

// shutdownEmulator closes the emulator and cleans its resources.
func (emu *EmulatorMock) shutdownEmulator() {
	_ = os.Remove(emu.GetHashPath())
	_ = os.Remove(emu.GetSRAMPath())

	close(emu.imageChannel)
	close(emu.audioChannel)
	close(emu.inputOutCh)

	nanoarchShutdown()
}

// emulateOneFrame emulates one frame with exclusive lock.
func (emu *EmulatorMock) emulateOneFrame() {
	emu.Lock()
	nanoarchRun()
	emu.Unlock()
}

// Who needs generics anyway?
// handleVideo is a custom message handler for the video channel.
func (emu *EmulatorMock) handleVideo(handler func(image GameFrame)) {
	for frame := range emu.imageInCh {
		handler(frame)
	}
}

// handleAudio is a custom message handler for the audio channel.
func (emu *EmulatorMock) handleAudio(handler func(sample []int16)) {
	for frame := range emu.audioInCh {
		handler(frame)
	}
}

// handleInput is a custom message handler for the input channel.
func (emu *EmulatorMock) handleInput(handler func(event InputEvent)) {
	for event := range emu.inputChannel {
		handler(event)
	}
}

// dumpState returns the current emulator state and
// the latest saved state for its session.
// Locks the emulator.
func (emu *EmulatorMock) dumpState() (string, string) {
	emu.Lock()
	bytes, _ := ioutil.ReadFile(emu.paths.save)
	persistedStateHash := getHash(bytes)
	emu.Unlock()

	stateHash := emu.getStateHash()
	fmt.Printf("mem: %v, dat: %v\n", stateHash, persistedStateHash)
	return stateHash, persistedStateHash
}

// getStateHash returns the current emulator state hash.
// Locks the emulator.
func (emu *EmulatorMock) getStateHash() string {
	emu.Lock()
	state, _ := getSaveState()
	emu.Unlock()

	return getHash(state)
}

// getRootPath returns absolute path to the root directory.
func getRootPath() string {
	p, _ := filepath.Abs("../../../../")
	return p + string(filepath.Separator)
}

// getHash returns MD5 hash.
func getHash(bytes []byte) string {
	return fmt.Sprintf("%x", md5.Sum(bytes))
}

// cleanPath returns a proper file path for current OS.
func cleanPath(path string) string {
	return filepath.FromSlash(path)
}

// benchmarkEmulator is a generic function for
// measuring emulator performance for one emulation frame.
func benchmarkEmulator(system string, rom string, b *testing.B) {
	log.SetOutput(ioutil.Discard)
	os.Stdout, _ = os.Open(os.DevNull)

	s := GetDefaultEmulatorMock("bench_"+system+"_performance", system, rom)
	for i := 0; i < b.N; i++ {
		s.emulateOneFrame()
	}
	s.shutdownEmulator()
}

func BenchmarkEmulatorGba(b *testing.B) {
	benchmarkEmulator("gba", "Sushi The Cat.gba", b)
}

func BenchmarkEmulatorNes(b *testing.B) {
	benchmarkEmulator("nes", "Super Mario Bros.nes", b)
}
