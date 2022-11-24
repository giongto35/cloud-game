package nanoarch

// Save writes the current state to the filesystem.
func (f *Frontend) Save() error {
	if f.roomID == "" {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := getSaveState()
	if err != nil {
		return err
	}
	if err := f.storage.Save(f.GetHashPath(), ss); err != nil {
		return err
	}

	if sram := getSaveRAM(); sram != nil {
		if err := f.storage.Save(f.GetSRAMPath(), sram); err != nil {
			return err
		}
	}
	return nil
}

// Load restores the state from the filesystem.
func (f *Frontend) Load() error {
	if f.roomID == "" {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	ss, err := f.storage.Load(f.GetHashPath())
	if err != nil {
		return err
	}
	if err := restoreSaveState(ss); err != nil {
		return err
	}

	sram, err := f.storage.Load(f.GetSRAMPath())
	if err != nil {
		return err
	}
	if sram != nil {
		restoreSaveRAM(sram)
	}
	return nil
}
