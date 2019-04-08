package ui

import (
	"image"
	"log"
	"time"

	"github.com/giongto35/cloud-game/nes"
)

type View interface {
	Enter()
	Exit()
	Update(t, dt float64)
}

type Director struct {
	// audio        *Audio
	view         View
	timestamp    float64
	imageChannel chan *image.RGBA
	inputChannel chan int
	roomID       string
}

func NewDirector(roomID string, imageChannel chan *image.RGBA, inputChannel chan int) *Director {
	// func NewDirector(audio *Audio, imageChannel chan *image.RGBA, inputChannel chan int) *Director {
	director := Director{}
	// director.audio = audio
	director.imageChannel = imageChannel
	director.inputChannel = inputChannel
	director.roomID = roomID
	return &director
}

func (d *Director) SetView(view View) {
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
	for {
		// quit game
		// TODO: Check if noone using

		d.Step()
	}
	d.SetView(nil)
}

func (d *Director) PlayGame(path string) {
	hash, err := hashFile(path, d.roomID)
	if err != nil {
		log.Fatalln(err)
	}
	console, err := nes.NewConsole(path)
	if err != nil {
		log.Fatalln(err)
	}
	d.SetView(NewGameView(d, console, path, hash, d.imageChannel, d.inputChannel))
}
