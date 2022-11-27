package emulator

import (
	img "image"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/emulator/image"
)

type CloudEmulator interface {
	// LoadMeta returns metadata of emulator
	LoadMeta(path string) Metadata
	// Start is called after LoadGame
	Start()
	// SetViewport sets viewport size
	SetViewport(width int, height int)
	// SaveGame save game state
	SaveGame() error
	// LoadGame load game state
	LoadGame() error
	// GetHashPath returns the path emulator will save state to
	GetHashPath() string
	// Close will be called when the game is done
	Close()
	// ToggleMultitap toggles multitap controller.
	ToggleMultitap()
	// Input passes input to the emulator
	Input(uid string, player int, data []byte)
}

type Metadata struct {
	// the full path to some emulator lib
	LibPath string
	// the full path to the emulator config
	ConfigPath string

	AudioSampleRate int
	Fps             float64
	BaseWidth       int
	BaseHeight      int
	Ratio           float64
	Rotation        image.Rotate
	IsGlAllowed     bool
	UsesLibCo       bool
	AutoGlContext   bool
	HasMultitap     bool
}

type (
	GameFrame struct {
		Data     *img.RGBA
		Duration time.Duration
	}
	GameAudio struct {
		Data     []int16
		Duration time.Duration
	}
	InputEvent struct {
		RawState []byte
	}
)
