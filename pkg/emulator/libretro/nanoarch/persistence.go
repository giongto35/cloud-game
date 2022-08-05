package nanoarch

// Save writes the current state to the filesystem.
func (na *naEmulator) Save() error {
	na.Lock()
	defer na.Unlock()

	ss, err := getSaveState()
	if err != nil {
		return err
	}
	if err := na.storage.Save(na.GetHashPath(), ss); err != nil {
		return err
	}

	if sram := getSaveRAM(); sram != nil {
		if err := na.storage.Save(na.GetSRAMPath(), sram); err != nil {
			return err
		}
	}
	return nil
}

// Load restores the state from the filesystem.
func (na *naEmulator) Load() error {
	na.Lock()
	defer na.Unlock()

	ss, err := na.storage.Load(na.GetHashPath())
	if err != nil {
		return err
	}
	if err := restoreSaveState(ss); err != nil {
		return err
	}

	sram, err := na.storage.Load(na.GetSRAMPath())
	if err != nil {
		return err
	}
	if sram != nil {
		restoreSaveRAM(sram)
	}
	return nil
}
