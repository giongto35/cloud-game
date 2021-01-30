package nanoarch

import "C"
import (
	"io/ioutil"
	"unsafe"
)

type state []byte

type mem struct {
	ptr  unsafe.Pointer
	size uint
}

// Save writes the current state to the filesystem.
// Deadlock warning: locks the emulator.
func (na *naEmulator) Save() (err error) {
	na.Lock()
	defer na.Unlock()

	if sram := getSRAM(); sram != nil {
		err = toFile(na.GetSRAMPath(), sram)
	}
	if state, err := getState(); err == nil {
		err = toFile(na.GetHashPath(), state)
	} else {
		return err
	}

	return
}

// Load restores the state from the filesystem.
// Deadlock warning: locks the emulator.
func (na *naEmulator) Load() (err error) {
	na.Lock()
	defer na.Unlock()

	if data, err := fromFile(na.GetSRAMPath()); err == nil {
		restoreSRAM(data)
	}

	path := na.GetHashPath()
	if state, err := fromFile(path); err == nil {
		err = restoreState(state)
	} else {
		return err
	}
	return err
}

// getSRAM returns the game SRAM data or a nil slice.
func getSRAM() []byte {
	mem := getSRAMMemory()
	if mem == nil {
		return nil
	}
	return C.GoBytes(mem.ptr, C.int(mem.size))
}

// restoreSRAM restores game SRAM.
func restoreSRAM(data []byte) {
	if len(data) == 0 {
		return
	}
	if mem := getSRAMMemory(); mem != nil {
		sram := (*[1 << 30]byte)(mem.ptr)[:mem.size:mem.size]
		copy(sram, data)
	}
}

// getState returns the current emulator state.
func getState() (state, error) {
	if dat, err := serialize(serializeSize()); err == nil {
		return dat, nil
	} else {
		return state{}, err
	}
}

// restoreState restores an emulator state.
func restoreState(dat state) error {
	return unserialize(dat, serializeSize())
}

// toFile writes the state to a file with the path.
func toFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0644)
}

// fromFile reads the state from a file with the path.
func fromFile(path string) ([]byte, error) {
	if bytes, err := ioutil.ReadFile(path); err == nil {
		return bytes, nil
	} else {
		return []byte{}, err
	}
}
