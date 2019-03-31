package ui

import (
	"image"

	"github.com/fogleman/nes/nes"
)

const padding = 0

type GameView struct {
	director     *Director
	console      *nes.Console
	title        string
	hash         string
	record       bool
	frames       []image.Image
	imageChannel chan *image.RGBA
}

func NewGameView(director *Director, console *nes.Console, title, hash string) View {
	imageChannel := make(chan *image.RGBA, 2)
	return &GameView{director, console, title, hash, false, nil, imageChannel}
}

func (view *GameView) GetImageChannel() chan *image.RGBA {
	return view.imageChannel
}

func (view *GameView) Enter() {
	// load state
	if err := view.console.LoadState(savePath(view.hash)); err == nil {
		return
	} else {
		view.console.Reset()
	}
	// load sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		if sram, err := readSRAM(sramPath(view.hash)); err == nil {
			cartridge.SRAM = sram
		}
	}
}

func (view *GameView) Exit() {
	view.console.SetAudioChannel(nil)
	view.console.SetAudioSampleRate(0)
	// save sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		writeSRAM(sramPath(view.hash), cartridge.SRAM)
	}
	// save state
	view.console.SaveState(savePath(view.hash))
}

func (view *GameView) Update(t, dt float64) {
	if dt > 1 {
		dt = 0
	}
	console := view.console
	//if readKey(window, glfw.KeyEscape) {
	//view.director.ShowMenu()
	//}
	//updateControllers(window, console)
	view.imageChannel <- console.Buffer()
	console.StepSeconds(dt)
	if view.record {
		view.frames = append(view.frames, copyImage(console.Buffer()))
	}
}

//func (view *GameView) onKey(window *glfw.Window,
//key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
//if action == glfw.Press {
//switch key {
//case glfw.KeyR:
//view.console.Reset()
//case glfw.KeyTab:
//if view.record {
//view.record = false
//view.frames = nil
//} else {
//view.record = true
//}
//}
//}
//}

//func updateControllers(window *glfw.Window, console *nes.Console) {
//turbo := console.PPU.Frame%6 < 3
//k1 := readKeys(window, turbo)
//j1 := readJoystick(glfw.Joystick1, turbo)
//j2 := readJoystick(glfw.Joystick2, turbo)
//console.SetButtons1(combineButtons(k1, j1))
//console.SetButtons2(j2)
//}
