package emulator

// CloudEmulator is the interface of cloud emulator. Currently NES emulator and RetroArch implements this in codebase
type CloudEmulator interface {
	// LoadMeta returns meta data of emulator. Refer below
	LoadMeta(path string) Meta
	// Start is called after LoadGame
	Start()
	// SaveGame save game state, saveExtraFunc is callback to do extra step. Ex: save to google cloud
	SaveGame(saveExtraFunc func() error) error
	// LoadGame load game state
	LoadGame() error
	// GetHashPath returns the path emulator will save state to
	GetHashPath() string
	// Close will be called when the game is done
	Close()
}

// Meta is metadata of game
type Meta struct {
	AudioSampleRate int
	Fps             int
	Width           int
	Height          int
}
