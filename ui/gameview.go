// credit to https://github.com/fogleman/nes
package ui

import (
	"image"

	// "strconv"

	"github.com/giongto35/cloud-game/nes"
)

const padding = 0

// List key pressed
const (
	a1 = iota
	b1
	select1
	start1
	up1
	down1
	left1
	right1
	save1
	load1
	a2
	b2
	select2
	start2
	up2
	down2
	left2
	right2
	save2
	load2
)
const NumKeys = 20

type GameView struct {
	director *Director
	console  *nes.Console
	title    string
	hash     string
	record   bool
	frames   []image.Image

	// equivalent to the list key pressed const above
	keyPressed [20]bool

	imageChannel chan *image.RGBA
	inputChannel chan int
}

func NewGameView(director *Director, console *nes.Console, title, hash string, imageChannel chan *image.RGBA, inputChannel chan int) View {
	gameview := &GameView{director, console, title, hash, false, nil, [NumKeys]bool{false}, imageChannel, inputChannel}
	go gameview.ListenToInputChannel()
	return gameview
}

func (view *GameView) ListenToInputChannel() {
	for {
		keysInBinary := <-view.inputChannel
		for i := 0; i < NumKeys; i++ {
			view.keyPressed[i] = ((keysInBinary & 1) == 1)
			keysInBinary = keysInBinary >> 1
		}
	}
}

func (view *GameView) Enter() {
	// Always reset game
	// view.console.SetAudioChannel(view.director.audio.channel)
	// view.console.SetAudioSampleRate(view.director.audio.sampleRate)
	// load state
	//if err := view.console.LoadState(savePath(view.hash)); err == nil {
	//return
	//} else {
	view.console.Reset()
	//}
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
	//view.console.SaveState(savePath(view.hash))
}

func (view *GameView) Update(t, dt float64) {
	if dt > 1 {
		dt = 0
	}
	console := view.console
	//updateControllers(window, console)
	view.updateControllers()
	console.StepSeconds(dt)

	// fps to set frame
	view.imageChannel <- console.Buffer()

	if view.record {
		view.frames = append(view.frames, copyImage(console.Buffer()))
	}
}

func (view *GameView) updateControllers() {
	// TODO: switch case
	// Divide keyPressed to player 1 and player 2
	// First 10 keys are player 1
	var player1Keys [8]bool
	copy(player1Keys[:], view.keyPressed[:8])
	var player2Keys [8]bool
	copy(player2Keys[:], view.keyPressed[10:18])

	view.console.Controller1.SetButtons(player1Keys)
	view.console.Controller2.SetButtons(player2Keys)

	if view.keyPressed[save1] || view.keyPressed[save2] {
		view.console.SaveState(savePath(view.hash))
	}
	if view.keyPressed[load1] || view.keyPressed[load2] {
		view.console.LoadState(savePath(view.hash))
	}
}
