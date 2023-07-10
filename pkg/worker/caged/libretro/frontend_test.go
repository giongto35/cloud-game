package libretro

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
)

type testRun struct {
	room           string
	system         string
	rom            string
	emulationTicks int
}

// EmulatorMock contains Frontend mocking data.
type EmulatorMock struct {
	*Frontend

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
// Make sure you call Shutdown().
func GetEmulatorMock(room string, system string) *EmulatorMock {
	rootPath := getRootPath()

	var conf config.WorkerConfig
	if _, err := config.LoadConfig(&conf, ""); err != nil {
		panic(err)
	}

	meta := conf.Emulator.GetLibretroCoreConfig(system)

	nano := nanoarch.NewNano(cleanPath(conf.Emulator.LocalPath))

	l := logger.Default()
	l2 := l.Extend(l.Level(logger.ErrorLevel).With())
	nano.SetLogger(l2)

	// an emu
	emu := &EmulatorMock{
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

		core: path.Base(meta.Lib),

		paths: EmulatorPaths{
			assets: cleanPath(rootPath),
			cores:  cleanPath(rootPath + "assets/cores/"),
			games:  cleanPath(rootPath + "assets/games/"),
		},
	}
	emu.linkNano(nano)

	emu.paths.save = cleanPath(emu.HashPath())

	return emu
}

// GetDefaultFrontend returns initialized emulator mock with default params.
// Spawns audio/image channels consumers.
// Don't forget to close emulator mock with Shutdown().
func GetDefaultFrontend(room string, system string, rom string) *EmulatorMock {
	mock := GetEmulatorMock(room, system)
	mock.loadRom(rom)
	mock.SetVideoCb(func(app.Video) {})
	mock.SetAudioCb(func(app.Audio) {})
	return mock
}

// loadRom loads a ROM into the emulator.
// The rom will be loaded from emulators' games path.
func (emu *EmulatorMock) loadRom(game string) {
	fmt.Printf("%v %v\n", emu.paths.cores, emu.core)
	emu.nano.CoreLoad(nanoarch.Metadata{LibPath: emu.paths.cores + emu.core})
	err := emu.nano.LoadGame(emu.paths.games + game)
	if err != nil {
		log.Fatal(err)
	}
	w, h := emu.FrameSize()
	if emu.conf.Scale == 0 {
		emu.conf.Scale = 1
	}
	emu.SetViewport(w, h, emu.conf.Scale)
}

// Shutdown closes the emulator and cleans its resources.
func (emu *EmulatorMock) Shutdown() {
	_ = os.Remove(emu.HashPath())
	_ = os.Remove(emu.SRAMPath())

	emu.Frontend.Close()
	emu.Frontend.Shutdown()
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
	state, _ := nanoarch.SaveState()
	emu.mu.Unlock()

	return getHash(state)
}

// getRootPath returns absolute path to the root directory.
func getRootPath() string {
	p, _ := filepath.Abs("../../../../")
	return p + string(filepath.Separator)
}

// getHash returns MD5 hash.
func getHash(bytes []byte) string { return fmt.Sprintf("%x", md5.Sum(bytes)) }

// cleanPath returns a proper file path for current OS.
func cleanPath(path string) string { return filepath.FromSlash(path) }

// benchmarkEmulator is a generic function for
// measuring emulator performance for one emulation frame.
func benchmarkEmulator(system string, rom string, b *testing.B) {
	b.StopTimer()
	log.SetOutput(io.Discard)
	os.Stdout, _ = os.Open(os.DevNull)

	s := GetDefaultFrontend("bench_"+system+"_performance", system, rom)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.nano.Run()
	}
	s.Shutdown()
}

func BenchmarkEmulatorGba(b *testing.B) {
	benchmarkEmulator("gba", "Sushi The Cat.gba", b)
}

func BenchmarkEmulatorNes(b *testing.B) {
	benchmarkEmulator("nes", "Alwa's Awakening (Demo).nes", b)
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

// Tests a successful emulator state save.
func TestSave(t *testing.T) {
	tests := []testRun{
		{
			room:           "test_save_ok_00",
			system:         "gba",
			rom:            "Sushi The Cat.gba",
			emulationTicks: 100,
		},
		{
			room:           "test_save_ok_01",
			system:         "gba",
			rom:            "anguna.gba",
			emulationTicks: 10,
		},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] save with [%v]\n", test.system, test.rom)

		front := GetDefaultFrontend(test.room, test.system, test.rom)

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
		{
			room:           "test_load_00",
			system:         "nes",
			rom:            "Alwa's Awakening (Demo).nes",
			emulationTicks: 100,
		},
		{
			room:           "test_load_01",
			system:         "gba",
			rom:            "Sushi The Cat.gba",
			emulationTicks: 1000,
		},
		{
			room:           "test_load_02",
			system:         "gba",
			rom:            "anguna.gba",
			emulationTicks: 100,
		},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] load with [%v]\n", test.system, test.rom)

		mock := GetDefaultFrontend(test.room, test.system, test.rom)

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
		run testRun
		// determine random
		seed int
	}{
		{
			run: testRun{
				room:           "test_concurrency_00",
				system:         "gba",
				rom:            "Sushi The Cat.gba",
				emulationTicks: 120,
			},
			seed: 42,
		},
		{
			run: testRun{
				room:           "test_concurrency_01",
				system:         "gba",
				rom:            "anguna.gba",
				emulationTicks: 300,
			},
			seed: 42 + 42,
		},
	}

	for _, test := range tests {
		t.Logf("Testing [%v] concurrency with [%v]\n", test.run.system, test.run.rom)

		mock := GetEmulatorMock(test.run.room, test.run.system)

		ops := &sync.WaitGroup{}
		// quantum lock
		qLock := &sync.Mutex{}

		mock.loadRom(test.run.rom)
		mock.SetVideoCb(func(v app.Video) {
			if len(v.Frame.Pix) == 0 {
				t.Errorf("It seems that rom video frame was empty, which is strange!")
			}
		})
		mock.SetAudioCb(func(app.Audio) {})

		t.Logf("Random seed is [%v]\n", test.seed)
		t.Logf("Save path is [%v]\n", mock.paths.save)

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

// lucky returns random boolean.
func lucky() bool { return rand.Intn(2) == 1 }

func TestConcurrentInput(t *testing.T) {
	players := NewGameSessionInput()

	events := 1000
	var wg sync.WaitGroup

	wg.Add(events * 2)

	go func() {
		for i := 0; i < events; i++ {
			player := rand.Intn(maxPort)
			go func() {
				players.setInput(player, []byte{0, 1})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < events; i++ {
			player := rand.Intn(maxPort)
			go func() {
				players.isKeyPressed(uint(player), 100)
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}
