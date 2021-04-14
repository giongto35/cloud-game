package nanoarch

import "io/ioutil"

// Save writes the current state to the filesystem.
// Deadlock warning: locks the emulator.
func (na *naEmulator) Save() (err error) {
	na.Lock()
	defer na.Unlock()

	if sramState := getSaveRAM(); sramState != nil {
		err = toFile(na.GetSRAMPath(), sramState)
	}
	if saveState, err := getSaveState(); err == nil {
		return toFile(na.GetHashPath(), saveState)
	}
	return
}

// Load restores the state from the filesystem.
// Deadlock warning: locks the emulator.
func (na *naEmulator) Load() (err error) {
	na.Lock()
	defer na.Unlock()

	if sramState, err := fromFile(na.GetSRAMPath()); err == nil {
		restoreSaveRAM(sramState)
	}
	if saveState, err := fromFile(na.GetHashPath()); err == nil {
		return restoreSaveState(saveState)
	}
	return
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
