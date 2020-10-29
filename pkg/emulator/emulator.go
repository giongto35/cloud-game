package emulator

import "github.com/giongto35/cloud-game/v2/pkg/config"

// CloudEmulator is the interface of cloud emulator. Currently NES emulator and RetroArch implements this in codebase
type CloudEmulator interface {
	// LoadMeta returns meta data of emulator. Refer below
	LoadMeta(path string) config.EmulatorMeta
	// Start is called after LoadGame
	Start()
	// SetViewport sets viewport size
	SetViewport(width int, height int)
	// GetViewport debug encoder image
	GetViewport() interface{}
	// SaveGame save game state, saveExtraFunc is callback to do extra step. Ex: save to google cloud
	SaveGame(saveExtraFunc func() error) error
	// LoadGame load game state
	LoadGame() error
	// GetHashPath returns the path emulator will save state to
	GetHashPath() string
	// Close will be called when the game is done
	Close()

	ToggleMultitap() error
}
