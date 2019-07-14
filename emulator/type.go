package emulator

type CloudEmulator interface {
	Start(path string)
	SaveGame(saveExtraFunc func() error) error
	LoadGame() error
	GetHashPath() string
	GetSampleRate() uint
	Close()
}
