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
	"time"

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
	name   string
	room   string
	system string
	rom    string
	frames int
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
	l2 := l.Extend(l.Level(logger.WarnLevel).With())

	if err := manager.CheckCores(conf.Emulator, l2); err != nil {
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
	emu.ViewportRecalculate()
}

// Shutdown closes the emulator and cleans its resources.
func (emu *TestFrontend) Shutdown() {
	_ = os.Remove(emu.HashPath())
	_ = os.Remove(emu.SRAMPath())
	emu.Frontend.Close()
	emu.Frontend.Shutdown()
}

// dumpState returns both current and previous emulator save state as MD5 hash string.
func (emu *TestFrontend) dumpState() (cur string, prev string) {
	emu.mu.Lock()
	b, _ := os.ReadFile(emu.HashPath())
	prev = hash(b)
	emu.mu.Unlock()

	emu.mu.Lock()
	b, _ = nanoarch.SaveState()
	emu.mu.Unlock()
	cur = hash(b)

	return
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

func TestSavePersistence(t *testing.T) {
	tests := []testRun{
		{system: sushi.system, rom: sushi.rom, frames: 100},
		{system: angua.system, rom: angua.rom, frames: 100},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("If saves persistent on %v - %v", test.system, test.rom), func(t *testing.T) {
			front := DefaultFrontend(test.room, test.system, test.rom)

			for test.frames > 0 {
				front.Tick()
				test.frames--
			}

			_, _ = front.dumpState()
			if err := front.Save(); err != nil {
				t.Error(err)
			}

			hash1, hash2 := front.dumpState()
			if hash1 != hash2 {
				t.Errorf("%v != %v", hash1, hash2)
			}

			front.Shutdown()
		})
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
		{room: "test_load_00", system: alwa.system, rom: alwa.rom, frames: 100},
		//{room: "test_load_01", system: sushi.system, rom: sushi.rom, frames: 1000},
		//{room: "test_load_02", system: angua.system, rom: angua.rom, frames: 100},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] load with [%v]\n", test.system, test.rom)

		mock := DefaultFrontend(test.room, test.system, test.rom)

		mock.dumpState()

		for ticks := test.frames; ticks > 0; ticks-- {
			mock.Tick()
		}
		mock.dumpState()

		if err := mock.Save(); err != nil {
			t.Errorf("Save fail %v", err)
		}
		snapshot1, _ := mock.dumpState()

		for ticks := test.frames; ticks > 0; ticks-- {
			mock.Tick()
		}
		mock.dumpState()

		if err := mock.Load(); err != nil {
			t.Errorf("Load fail %v", err)
		}
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
			run:  testRun{room: "test_concurrency_00", system: alwa.system, rom: alwa.rom, frames: 120},
			seed: 42,
		},
		{
			run:  testRun{room: "test_concurrency_01", system: alwa.system, rom: alwa.rom, frames: 300},
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

		for i := 0; i < test.run.frames; i++ {
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

func TestStartStop(t *testing.T) {
	f1 := DefaultFrontend("sushi", sushi.system, sushi.rom)
	go f1.Start()
	time.Sleep(1 * time.Second)
	f1.Close()

	f2 := DefaultFrontend("sushi", sushi.system, sushi.rom)
	go f2.Start()
	time.Sleep(100 * time.Millisecond)
	f2.Close()
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
