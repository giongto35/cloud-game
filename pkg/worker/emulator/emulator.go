package emulator

import (
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/image"
)

type Emulator interface {
	// SetAudio sets the audio callback
	SetAudio(func(*GameAudio))
	// SetVideo sets the video callback
	SetVideo(func(*GameFrame))
	GetAudio() func(*GameAudio)
	GetVideo() func(*GameFrame)
	LoadMetadata(name string)
	LoadGame(path string) error
	GetFps() uint
	GetSampleRate() uint
	GetFrameSize() (w, h int)
	HasVerticalFrame() bool
	// Start is called after LoadGame
	Start()
	// SetViewport sets viewport size
	SetViewport(width int, height int, scale int)
	// SetMainSaveName sets distinct name for saves naming
	SetMainSaveName(name string)
	// SaveGameState save game state
	SaveGameState() error
	// LoadGameState load game state
	LoadGameState() error
	// GetHashPath returns the path emulator will save state to
	GetHashPath() string
	// Close will be called when the game is done
	Close()
	// ToggleMultitap toggles multitap controller.
	ToggleMultitap()
	// Input passes input to the emulator
	Input(player int, data []byte)
}

type Metadata struct {
	LibPath         string // the full path to some emulator lib
	AudioSampleRate int
	Fps             float64
	BaseWidth       int
	BaseHeight      int
	Rotation        image.Rotate
	IsGlAllowed     bool
	UsesLibCo       bool
	AutoGlContext   bool
	HasMultitap     bool
	HasVFR          bool
	Options         map[string]string
	Hacks           []string
}

func (m Metadata) HasHack(h string) bool {
	for _, n := range m.Hacks {
		if h == n {
			return true
		}
	}
	return false
}

type (
	GameFrame struct {
		Data     *image.Frame
		Duration time.Duration
	}
	GameAudio struct {
		Data     *[]int16
		Duration time.Duration
	}
	InputEvent struct {
		RawState []byte
	}
)
