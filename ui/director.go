package ui

import (
	"image"
	"log"
	"time"

	"github.com/giongto35/cloud-game/nes"
)

type Director struct {
	// audio        *Audio
	view          *GameView
	timestamp     float64
	imageChannel  chan *image.RGBA
	inputChannel  chan int
	closedChannel chan bool
	roomID        string
	hash		  string
}

const FPS = 60

func NewDirector(roomID string, imageChannel chan *image.RGBA, inputChannel chan int, closedChannel chan bool) *Director {
	director := Director{}
	// director.audio = audio
	director.imageChannel = imageChannel
	director.inputChannel = inputChannel
	director.closedChannel = closedChannel
	director.roomID = roomID
	director.hash = ""
	return &director
}

func (d *Director) SetView(view *GameView) {
	if d.view != nil {
		d.view.Exit()
	}
	d.view = view
	if d.view != nil {
		d.view.Enter()
	}
	d.timestamp = float64(time.Now().Nanosecond()) / float64(time.Second)
}

func (d *Director) Step() {
	timestamp := float64(time.Now().Nanosecond()) / float64(time.Second)
	dt := timestamp - d.timestamp
	d.timestamp = timestamp
	if d.view != nil {
		d.view.Update(timestamp, dt)
	}
}

func (d *Director) Start(paths []string) {
	if len(paths) == 1 {
		d.PlayGame(paths[0])
	}
	d.Run()
}

func (d *Director) Run() {
	c := time.Tick(time.Second / FPS)
L:
	for range c {
		// quit game

		select {
		// if there is event from close channel => the game is ended
		case <-d.closedChannel:
			break L
		default:
		}

		d.Step()
	}
	d.SetView(nil)
}

func (d *Director) PlayGame(path string) {
	// Generate hash that is indentifier of a room (game path + ropomID)
	hash, err := hashFile(path, d.roomID)
	if err != nil {
		log.Fatalln(err)
	}
	d.hash = hash
	console, err := nes.NewConsole(path)
	if err != nil {
		log.Fatalln(err)
	}
	// Set GameView as current view
	d.SetView(NewGameView(console, path, hash, d.imageChannel, d.inputChannel))
}

func (d *Director) SaveGame() error {
	if d.hash != "" {
		return d.view.console.SaveState(savePath(d.hash))
	} else {
		return nil
	}
}

func (d *Director) LoadGame() error {
	if d.hash != "" {
		return d.view.console.LoadState(savePath(d.hash))
	} else {
		return nil
	}
}