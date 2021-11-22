package nanoarch

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

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

		mock := GetDefaultEmulatorMock(test.room, test.system, test.rom)

		for test.emulationTicks > 0 {
			mock.emulateOneFrame()
			test.emulationTicks--
		}

		fmt.Printf("[%-14v] ", "before save")
		_, _ = mock.dumpState()
		if err := mock.Save(); err != nil {
			t.Errorf("Save fail %v", err)
		}
		fmt.Printf("[%-14v] ", "after  save")
		snapshot1, snapshot2 := mock.dumpState()

		if snapshot1 != snapshot2 {
			t.Errorf("It seems rom state save has failed: %v != %v", snapshot1, snapshot2)
		}

		mock.shutdownEmulator()
	}
}

// Tests save and restore function:
//
// Emulate n ticks.
// Call save (a).
// Emulate n ticks again.
// Call load from the save (b).
// Compare states (a) and (b), should be =.
//
func TestLoad(t *testing.T) {
	tests := []testRun{
		{
			room:           "test_load_00",
			system:         "nes",
			rom:            "Super Mario Bros.nes",
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

		mock := GetDefaultEmulatorMock(test.room, test.system, test.rom)

		fmt.Printf("[%-14v] ", "initial")
		mock.dumpState()

		for ticks := test.emulationTicks; ticks > 0; ticks-- {
			mock.emulateOneFrame()
		}
		fmt.Printf("[%-14v] ", fmt.Sprintf("emulated %d", test.emulationTicks))
		mock.dumpState()

		if err := mock.Save(); err != nil {
			t.Errorf("Save fail %v", err)
		}
		fmt.Printf("[%-14v] ", "saved")
		snapshot1, _ := mock.dumpState()

		for ticks := test.emulationTicks; ticks > 0; ticks-- {
			mock.emulateOneFrame()
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

		mock.shutdownEmulator()
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
		op := 0

		mock.loadRom(test.run.rom)
		go mock.handleVideo(func(frame GameFrame) {
			if len(frame.Data.Pix) == 0 {
				t.Errorf("It seems that rom video frame was empty, which is strange!")
			}
		})
		go mock.handleAudio(func(_ []int16) {})
		go mock.handleInput(func(_ InputEvent) {})

		rand.Seed(int64(test.seed))
		t.Logf("Random seed is [%v]\n", test.seed)
		t.Logf("Save path is [%v]\n", mock.paths.save)

		_ = mock.Save()

		// emulation fps ROM cap
		ticker := time.NewTicker(time.Second / time.Duration(mock.meta.Fps))
		t.Logf("FPS limit is [%v]\n", mock.meta.Fps)

		for range ticker.C {
			select {
			case <-mock.done:
				mock.shutdownEmulator()
				return
			default:
			}

			op++
			if op > test.run.emulationTicks {
				mock.Close()
			} else {
				qLock.Lock()
				mock.emulateOneFrame()
				qLock.Unlock()

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
							t.Errorf("States are inconsistent %v != %v on tick %v\n", snapshot1, snapshot2, op)
						}
						ops.Done()
					}()
				}
			}
		}

		ops.Wait()
		ticker.Stop()
	}
}

// lucky returns random boolean.
func lucky() bool {
	return rand.Intn(2) == 1
}
