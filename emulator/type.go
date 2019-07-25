package emulator

type CloudEmulator interface {
	LoadMeta(path string) Meta
	Start()
	SaveGame(saveExtraFunc func() error) error
	LoadGame() error
	GetHashPath() string
	//GetSampleRate() uint
	Close()
}

type Meta struct {
	AudioSampleRate int
	Fps             int
	Width           int
	Height          int
}
