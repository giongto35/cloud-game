package emulator

import (
	"image"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/emulator"
	"github.com/giongto35/cloud-game/libretro/core"
)

type CloudEmulator interface {
	SetView(view *emulator.GameView)
	Step()
	Start(path string)
	Run()
	PlayGame(path string)
	SaveGame(saveExtraFunc func() error) error
	LoadGame() error
	GetHashPath() string
	Close()
}

type ludoEmulator struct {
	timestamp    float64
	imageChannel chan<- *image.RGBA
	audioChannel chan<- float32
	inputChannel <-chan int
	Done         chan struct{}
}

func NewLudoEmulator() CloudEmulator {
	vid = v
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			state.Global.Lock()
			running := state.Global.CoreRunning
			state.Global.Unlock()
			if running && !state.Global.MenuActive {
				savefiles.SaveSRAM()
			}
		}
	}()

	core.Load("emulator/libretro/cores/pcsx_rearmed_libretro.so", refreshFunc)
	return &ludoEmulator{}
}

func (e *ludoEmulator) Refresh(data unsafe.Pointer, width int32, height int32, pitch int32) {
	// convert data to image.RGBA
	buffer := *image.RGBA{}
	pixels = *(*[]byte)(data)
	// Access RGBA
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			buffer.set(x, y, color.NewColor())
		}
	}
	view.imageChannel <- buffer
}

func (e *ludoEmulator) SetView(view *emulator.GameView) {
	return
}

func (e *ludoEmulator) PlayGame(path string) {
	core.LoadGame(path)
}

func (e *ludoEmulator) SaveGame(path string) {

}

func (e *ludoEmulator) LoadGame(path string) {

}
