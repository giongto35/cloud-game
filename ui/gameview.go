package ui

import (
	"image"

	"fmt"
	"time"

	"github.com/giongto35/cloud-game/nes"
)

const padding = 0

type GameView struct {
	director *Director
	console  *nes.Console
	title    string
	hash     string
	record   bool
	frames   []image.Image

	keyPressed [8]bool

	imageChannel chan *image.RGBA
	inputChannel chan int

	nanotime int64
}

func NewGameView(director *Director, console *nes.Console, title, hash string, imageChannel chan *image.RGBA, inputChannel chan int) View {
	gameview := &GameView{director, console, title, hash, false, nil, [8]bool{false}, imageChannel, inputChannel, time.Now().UnixNano()}
	go gameview.ListenToInputChannel()
	return gameview
}

func (view *GameView) ListenToInputChannel() {
	for {
		key := <-view.inputChannel
		s := fmt.Sprintf("%.8b", key)
		for i := 0; i < len(s); i++ {
			if s[i] == '1' {
				view.keyPressed[i] = true
			} else {
				view.keyPressed[i] = false
			}
		}
	}
}

func (view *GameView) Enter() {
	// view.console.SetAudioChannel(view.director.audio.channel)
	// view.console.SetAudioSampleRate(view.director.audio.sampleRate)
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
	// view.console.SetAudioChannel(nil)
	// view.console.SetAudioSampleRate(0)
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
	//updateControllers(window, console)
	view.updateControllers()
	//fmt.Println(console.Buffer())
	console.StepSeconds(dt)

	// fps to set frame
	n := time.Now().UnixNano()
	if n - view.nanotime > 1000000000 / 100000 {
		view.nanotime = n
		view.imageChannel <- console.Buffer()
	}
	


	if view.record {
		view.frames = append(view.frames, copyImage(console.Buffer()))
	}
}

func (view *GameView) updateControllers() {
	// TODO: switch case
	// var buttons [8]bool
	// buttons[nes.ButtonLeft] = view.keyPressed[37]
	// buttons[nes.ButtonUp] = view.keyPressed[38]
	// buttons[nes.ButtonRight] = view.keyPressed[39]
	// buttons[nes.ButtonDown] = view.keyPressed[40]
	// buttons[nes.ButtonA] = view.keyPressed[32]
	// buttons[nes.ButtonB] = view.keyPressed[17]
	// buttons[nes.ButtonStart] = view.keyPressed[13]
	// buttons[nes.ButtonSelect] = view.keyPressed[16]
	// view.console.Controller1.SetButtons(buttons)
	view.console.Controller1.SetButtons(view.keyPressed)
}
