// credit to https://github.com/fogleman/nes
package ui

import (
	"image"

	"github.com/giongto35/cloud-game/nes"
)

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

	a2
	b2
	select2
	start2
	up2
	down2
	left2
	right2
)
const NumKeys = 8

type GameView struct {
	console *nes.Console
	title   string
	hash    string

	// equivalent to the list key pressed const above
	keyPressed [NumKeys * 2]bool

	savingPath  string
	loadingPath string

	imageChannel chan *image.RGBA
	inputChannel chan int
}

func NewGameView(console *nes.Console, title, hash string, imageChannel chan *image.RGBA, inputChannel chan int) *GameView {
	gameview := &GameView{
		console:      console,
		title:        title,
		hash:         hash,
		keyPressed:   [NumKeys * 2]bool{false},
		imageChannel: imageChannel,
		inputChannel: inputChannel,
	}

	go gameview.ListenToInputChannel()
	return gameview
}

// ListenToInputChannel listen from input channel streamm, which is exposed to WebRTC session
//func (view *GameView) UpdateInput(keysInBinary int) {
//for i := 0; i < NumKeys*2; i++ {
//b := ((keysInBinary & 1) == 1)
//view.keyPressed[i] = (view.keyPressed[i] && b) || b
//keysInBinary = keysInBinary >> 1
//}
//}

// ListenToInputChannel listen from input channel streamm, which is exposed to WebRTC session
func (view *GameView) ListenToInputChannel() {
	for {
		keysInBinary := <-view.inputChannel
		for i := 0; i < NumKeys*2; i++ {
			b := ((keysInBinary & 1) == 1)
			view.keyPressed[i] = (view.keyPressed[i] && b) || b
			keysInBinary = keysInBinary >> 1
		}
	}
}

// Enter enter the game view.
func (view *GameView) Enter() {
	// Always reset game
	// Legacy Audio code. TODO: Add it back to support audio
	// view.console.SetAudioChannel(view.director.audio.channel)
	// view.console.SetAudioSampleRate(view.director.audio.sampleRate)

	// load state
	if err := view.console.LoadState(savePath(view.hash)); err == nil {
		return
	} else {
		view.console.Reset()
	}
	//view.console.Reset()

	// load sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		if sram, err := readSRAM(sramPath(view.hash)); err == nil {
			cartridge.SRAM = sram
		}
	}
}

// Exit ...
func (view *GameView) Exit() {
	// view.console.SetAudioChannel(nil)
	// view.console.SetAudioSampleRate(0)
	// save sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		writeSRAM(sramPath(view.hash), cartridge.SRAM)
	}
}

// Update is called for every period of time, dt is the elapsed time from the last frame
func (view *GameView) Update(t, dt float64) {
	if dt > 1 {
		dt = 0
	}
	console := view.console
	view.updateControllers()
	view.UpdateEvents()
	console.StepSeconds(dt)

	// fps to set frame
	view.imageChannel <- console.Buffer()
}

func (view *GameView) Save(hash string) {
	// put saving event to queue, process in updateEvent
	view.savingPath = savePath(view.hash)
}

func (view *GameView) Load(path string) {
	// put saving event to queue, process in updateEvent
	view.loadingPath = savePath(view.hash)
}

func (view *GameView) UpdateEvents() {
	// If there is saving event, save and discard the save event
	if view.savingPath != "" {
		view.console.SaveState(view.savingPath)
		view.savingPath = ""
	}
	// If there is loading event, save and discard the load event
	if view.loadingPath != "" {
		view.console.LoadState(view.loadingPath)
		view.loadingPath = ""
	}
}

func (view *GameView) updateControllers() {
	// Divide keyPressed to player 1 and player 2
	// First 8 keys are player 1
	var player1Keys [8]bool
	copy(player1Keys[:], view.keyPressed[:8])
	var player2Keys [8]bool
	copy(player2Keys[:], view.keyPressed[8:])

	view.console.Controller1.SetButtons(player1Keys)
	view.console.Controller2.SetButtons(player2Keys)

}
