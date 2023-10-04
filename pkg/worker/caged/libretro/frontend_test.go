package libretro

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/manager"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"

	_ "github.com/giongto35/cloud-game/v3/test"
)

type TestFrontend struct {
	*Frontend

	corePath string
	gamePath string
}

type testRun struct {
	room           string
	system         string
	rom            string
	emulationTicks int
}

type game struct {
	rom    string
	system string
}

var (
	alwa  = game{system: "nes", rom: "Alwa's Awakening (Demo).nes"}
	sushi = game{system: "gba", rom: "Sushi The Cat.gba"}
	angua = game{system: "gba", rom: "anguna.gba"}
)

// TestMain runs all tests in the main thread in macOS.
func TestMain(m *testing.M) {
	thread.Wrap(func() { os.Exit(m.Run()) })
}

// EmulatorMock returns a properly stubbed emulator instance.
// Due to extensive use of globals -- one mock instance is allowed per a test run.
// Don't forget to init one image channel consumer, it will lock-out otherwise.
// Make sure you call Shutdown().
func EmulatorMock(room string, system string) *TestFrontend {
	var conf config.WorkerConfig
	if _, err := config.LoadConfig(&conf, ""); err != nil {
		panic(err)
	}

	conf.Emulator.Libretro.Cores.Repo.ExtLock = expand("tests", ".cr", "cloud-game.lock")
	conf.Emulator.LocalPath = expand("tests", conf.Emulator.LocalPath)
	conf.Emulator.Storage = expand("tests", "storage")

	l := logger.Default()
	l2 := l.Extend(l.Level(logger.ErrorLevel).With())

	if err := manager.CheckCores(conf.Emulator, l); err != nil {
		panic(err)
	}

	nano := nanoarch.NewNano(conf.Emulator.LocalPath)
	nano.SetLogger(l2)

	// an emu
	emu := &TestFrontend{
		Frontend: &Frontend{
			conf: conf.Emulator,
			storage: &StateStorage{
				Path:     os.TempDir(),
				MainSave: room,
			},
			input:       NewGameSessionInput(),
			done:        make(chan struct{}),
			th:          conf.Emulator.Threads,
			log:         l2,
			SaveOnClose: false,
		},
		corePath: expand(conf.Emulator.GetLibretroCoreConfig(system).Lib),
		gamePath: expand(conf.Worker.Library.BasePath),
	}
	emu.linkNano(nano)

	return emu
}

// DefaultFrontend returns initialized emulator mock with default params.
// Spawns audio/image channels consumers.
// Don't forget to close emulator mock with Shutdown().
func DefaultFrontend(room string, system string, rom string) *TestFrontend {
	mock := EmulatorMock(room, system)
	mock.loadRom(rom)
	mock.SetVideoCb(func(app.Video) {})
	mock.SetAudioCb(func(app.Audio) {})
	return mock
}

// loadRom loads a ROM into the emulator.
// The rom will be loaded from emulators' games path.
func (emu *TestFrontend) loadRom(game string) {
	emu.nano.CoreLoad(nanoarch.Metadata{LibPath: emu.corePath})

	gamePath := expand(emu.gamePath, game)

	conf := emu.conf.GetLibretroCoreConfig(gamePath)
	scale := 1.0
	if conf.Scale > 1 {
		scale = conf.Scale
	}
	emu.scale = scale

	err := emu.nano.LoadGame(gamePath)
	if err != nil {
		log.Fatal(err)
	}
	w, h := emu.FrameSize()
	emu.SetViewport(w, h)
}

// Shutdown closes the emulator and cleans its resources.
func (emu *TestFrontend) Shutdown() {
	_ = os.Remove(emu.HashPath())
	_ = os.Remove(emu.SRAMPath())
	emu.Frontend.Close()
	emu.Frontend.Shutdown()
}

// dumpState returns the current emulator state and
// the latest saved state for its session.
// Locks the emulator.
func (emu *TestFrontend) dumpState() (string, string) {
	emu.mu.Lock()
	bytes, _ := os.ReadFile(emu.HashPath())
	lastStateHash := hash(bytes)
	emu.mu.Unlock()

	emu.mu.Lock()
	state, _ := nanoarch.SaveState()
	emu.mu.Unlock()
	stateHash := hash(state)

	fmt.Printf("mem: %v, dat: %v\n", stateHash, lastStateHash)
	return stateHash, lastStateHash
}

func BenchmarkEmulators(b *testing.B) {
	log.SetOutput(io.Discard)
	os.Stdout, _ = os.Open(os.DevNull)

	benchmarks := []struct {
		name   string
		system string
		rom    string
	}{
		{name: "GBA Sushi", system: sushi.system, rom: sushi.rom},
		{name: "NES Alwa", system: alwa.system, rom: alwa.rom},
	}

	for _, bench := range benchmarks {
		b.Run(bench.name, func(b *testing.B) {
			s := DefaultFrontend("bench_"+bench.system+"_performance", bench.system, bench.rom)
			for i := 0; i < b.N; i++ {
				s.nano.Run()
			}
			s.Shutdown()
		})
	}
}

// Tests a successful emulator state save.
func TestSave(t *testing.T) {
	tests := []testRun{
		{room: "test_save_ok_00", system: sushi.system, rom: sushi.rom, emulationTicks: 100},
		{room: "test_save_ok_01", system: angua.system, rom: angua.rom, emulationTicks: 10},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] save with [%v]\n", test.system, test.rom)

		front := DefaultFrontend(test.room, test.system, test.rom)

		for test.emulationTicks > 0 {
			front.Tick()
			test.emulationTicks--
		}

		fmt.Printf("[%-14v] ", "before save")
		_, _ = front.dumpState()
		if err := front.Save(); err != nil {
			t.Errorf("Save fail %v", err)
		}
		fmt.Printf("[%-14v] ", "after  save")
		snapshot1, snapshot2 := front.dumpState()

		if snapshot1 != snapshot2 {
			t.Errorf("It seems rom state save has failed: %v != %v", snapshot1, snapshot2)
		}

		front.Shutdown()
	}
}

// Tests save and restore function:
//
// Emulate n ticks.
// Call save (a).
// Emulate n ticks again.
// Call load from the save (b).
// Compare states (a) and (b), should be =.
func TestLoad(t *testing.T) {
	tests := []testRun{
		{room: "test_load_00", system: alwa.system, rom: alwa.rom, emulationTicks: 100},
		{room: "test_load_01", system: sushi.system, rom: sushi.rom, emulationTicks: 1000},
		{room: "test_load_02", system: angua.system, rom: angua.rom, emulationTicks: 100},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] load with [%v]\n", test.system, test.rom)

		mock := DefaultFrontend(test.room, test.system, test.rom)

		fmt.Printf("[%-14v] ", "initial")
		mock.dumpState()

		for ticks := test.emulationTicks; ticks > 0; ticks-- {
			mock.Tick()
		}
		fmt.Printf("[%-14v] ", fmt.Sprintf("emulated %d", test.emulationTicks))
		mock.dumpState()

		if err := mock.Save(); err != nil {
			t.Errorf("Save fail %v", err)
		}
		fmt.Printf("[%-14v] ", "saved")
		snapshot1, _ := mock.dumpState()

		for ticks := test.emulationTicks; ticks > 0; ticks-- {
			mock.Tick()
		}
		fmt.Printf("[%-14v] ", fmt.Sprintf("emulated %d", test.emulationTicks))
		mock.dumpState()

		if err := mock.Load(); err != nil {
			t.Errorf("Load fail %v", err)
		}
		fmt.Printf("[%-14v] ", "restored")
		snapshot2, _ := mock.dumpState()

		if snapshot1 != snapshot2 {
			t.Errorf("It seems rom state restore has failed: %v != %v", snapshot1, snapshot2)
		}

		mock.Shutdown()
	}
}

func TestStateConcurrency(t *testing.T) {
	tests := []struct {
		run  testRun
		seed int
	}{
		{
			run:  testRun{room: "test_concurrency_00", system: sushi.system, rom: sushi.rom, emulationTicks: 120},
			seed: 42,
		},
		{
			run:  testRun{room: "test_concurrency_01", system: angua.system, rom: angua.rom, emulationTicks: 300},
			seed: 42 + 42,
		},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] concurrency with [%v]\n", test.run.system, test.run.rom)

		mock := EmulatorMock(test.run.room, test.run.system)

		ops := &sync.WaitGroup{}
		// quantum lock
		qLock := &sync.Mutex{}

		mock.loadRom(test.run.rom)
		mock.SetVideoCb(func(v app.Video) {
			if len(v.Frame.Data) == 0 {
				t.Errorf("It seems that rom video frame was empty, which is strange!")
			}
		})
		mock.SetAudioCb(func(app.Audio) {})

		t.Logf("Random seed is [%v]\n", test.seed)
		t.Logf("Save path is [%v]\n", mock.HashPath())

		_ = mock.Save()

		for i := 0; i < test.run.emulationTicks; i++ {
			qLock.Lock()
			mock.Tick()
			qLock.Unlock()

			i := i
			if lucky() && !lucky() {
				ops.Add(1)
				go func() {
					qLock.Lock()
					defer qLock.Unlock()

					mock.dumpState()
					// remove save to reproduce the bug
					_ = mock.Save()
					_, snapshot1 := mock.dumpState()
					_ = mock.Load()
					snapshot2, _ := mock.dumpState()

					// Bug or feature?
					// When you load a state from the file
					// without immediate preceding save,
					// it won't be in the loaded state
					// even without calling retro_run.
					// But if you pause the threads with a debugger
					// and run the code step by step, then it will work as expected.
					// Possible background emulation?

					if snapshot1 != snapshot2 {
						t.Errorf("States are inconsistent %v != %v on tick %v\n", snapshot1, snapshot2, i+1)
					}
					ops.Done()
				}()
			}
		}

		ops.Wait()
		mock.Shutdown()
	}
}

func TestConcurrentInput(t *testing.T) {
	var wg sync.WaitGroup
	state := NewGameSessionInput()
	events := 1000
	wg.Add(2 * events)

	for i := 0; i < events; i++ {
		player := rand.Intn(maxPort)
		go func() { state.setInput(player, []byte{0, 1}); wg.Done() }()
		go func() { state.isKeyPressed(uint(player), 100); wg.Done() }()
	}
	wg.Wait()
}

// expand joins a list of file path elements.
func expand(p ...string) string {
	ph, _ := filepath.Abs(filepath.FromSlash(filepath.Join(p...)))
	return ph
}

// hash returns MD5 hash.
func hash(bytes []byte) string { return fmt.Sprintf("%x", md5.Sum(bytes)) }

// lucky returns random boolean.
func lucky() bool { return rand.Intn(2) == 1 }
