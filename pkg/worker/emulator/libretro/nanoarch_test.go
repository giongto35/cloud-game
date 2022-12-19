package libretro

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
)

type testRun struct {
	room           string
	system         string
	rom            string
	emulationTicks int
}

// EmulatorMock contains Frontend mocking data.
type EmulatorMock struct {
	Frontend

	// Libretro compiled lib core name
	core string
	// shared core paths (can't be changed)
	paths EmulatorPaths
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

	l := logger.Default()
	SetLibretroLogger(l.Extend(l.Level(logger.ErrorLevel).With()))

	// an emu
	emu := &EmulatorMock{
		Frontend: Frontend{
			conf: conf.Emulator,
			storage: &StateStorage{
				Path:     os.TempDir(),
				MainSave: room,
			},
			input: NewGameSessionInput(),
			done:  make(chan struct{}),
			th:    conf.Emulator.Threads,
		},

		core: path.Base(meta.Lib),

		paths: EmulatorPaths{
			assets: cleanPath(rootPath),
			cores:  cleanPath(rootPath + "assets/cores/"),
			games:  cleanPath(rootPath + "assets/games/"),
		},
	}

	emu.paths.save = cleanPath(emu.GetHashPath())
	frontend = &emu.Frontend

	Init(cleanPath(conf.Emulator.LocalPath))

	return emu
}

// GetDefaultEmulatorMock returns initialized emulator mock with default params.
// Spawns audio/image channels consumers.
// Don't forget to close emulator mock with shutdownEmulator().
func GetDefaultEmulatorMock(room string, system string, rom string) *EmulatorMock {
	mock := GetEmulatorMock(room, system)
	mock.loadRom(rom)
	mock.handleVideo(func(_ *emulator.GameFrame) {})
	mock.handleAudio(func(_ *emulator.GameAudio) {})

	return mock
}

// loadRom loads a ROM into the emulator.
// The rom will be loaded from emulators' games path.
func (emu *EmulatorMock) loadRom(game string) {
	fmt.Printf("%v %v\n", emu.paths.cores, emu.core)
	coreLoad(emulator.Metadata{LibPath: emu.paths.cores + emu.core})
	err := LoadGame(emu.paths.games + game)
	if err != nil {
		log.Fatal(err)
	}
	emu.vw, emu.vh = emu.GetFrameSize()
}

// shutdownEmulator closes the emulator and cleans its resources.
func (emu *EmulatorMock) shutdownEmulator() {
	_ = os.Remove(emu.GetHashPath())
	_ = os.Remove(emu.GetSRAMPath())

	nanoarchShutdown()
}

// emulateOneFrame emulates one frame with exclusive lock.
func (emu *EmulatorMock) emulateOneFrame() {
	emu.mu.Lock()
	run()
	emu.mu.Unlock()
}

// Who needs generics anyway?
// handleVideo is a custom message handler for the video channel.
func (emu *EmulatorMock) handleVideo(handler func(image *emulator.GameFrame)) {
	emu.Frontend.onVideo = handler
}

// handleAudio is a custom message handler for the audio channel.
func (emu *EmulatorMock) handleAudio(handler func(sample *emulator.GameAudio)) {
	emu.Frontend.onAudio = handler
}

// dumpState returns the current emulator state and
// the latest saved state for its session.
// Locks the emulator.
func (emu *EmulatorMock) dumpState() (string, string) {
	emu.mu.Lock()
	bytes, _ := os.ReadFile(emu.paths.save)
	persistedStateHash := getHash(bytes)
	emu.mu.Unlock()

	stateHash := emu.getStateHash()
	fmt.Printf("mem: %v, dat: %v\n", stateHash, persistedStateHash)
	return stateHash, persistedStateHash
}

// getStateHash returns the current emulator state hash.
// Locks the emulator.
func (emu *EmulatorMock) getStateHash() string {
	emu.mu.Lock()
	state, _ := getSaveState()
	emu.mu.Unlock()

	return getHash(state)
}

func (emu *EmulatorMock) Close() {
	emu.Frontend.Close()
	<-nano.reserved
}

// getRootPath returns absolute path to the root directory.
func getRootPath() string {
	p, _ := filepath.Abs("../../../../../")
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
	b.StopTimer()
	log.SetOutput(io.Discard)
	os.Stdout, _ = os.Open(os.DevNull)
	libretroLogger = logger.New(false)

	s := GetDefaultEmulatorMock("bench_"+system+"_performance", system, rom)

	b.StartTimer()
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

func TestSwap(t *testing.T) {
	data := []byte{1, 254, 255, 32}
	pixel := *(*uint32)(unsafe.Pointer(&data[0]))
	// 0 1 2 3
	// 2 1 0 3
	ll := ((pixel >> 16) & 0xff) | (pixel & 0xff00) | ((pixel << 16) & 0xff0000) | 0xff000000

	rez := []byte{0, 0, 0, 0}
	*(*uint32)(unsafe.Pointer(&rez[0])) = ll

	log.Printf("%v\n%v", data, rez)
}
