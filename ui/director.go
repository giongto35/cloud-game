package ui

import (
	"image"
	"log"
	"time"

	"github.com/fogleman/nes/nes"
)

type View interface {
	Enter()
	Exit()
	GetImageChannel() chan *image.RGBA
	Update(t, dt float64)
}

type Director struct {
	view      View
	timestamp float64
}

func NewDirector() *Director {
	director := Director{}
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
	d.timestamp = float64(time.Now().Unix())
}

func (d *Director) Step() {
	//timestamp := glfw.GetTime()
	timestamp := float64(time.Now().Unix())
	dt := timestamp - d.timestamp
	d.timestamp = timestamp
	if d.view != nil {
		d.view.Update(timestamp, dt)
	}
}

func (d *Director) Start(path string) {
	d.PlayGame(path)
	d.Run()
}

func (d *Director) Run() {
	d.SetView(nil)
}

func (d *Director) PlayGame(path string) {
	hash, err := hashFile(path)
	if err != nil {
		log.Fatalln(err)
	}
	console, err := nes.NewConsole(path)
	if err != nil {
		log.Fatalln(err)
	}
	d.SetView(NewGameView(d, console, path, hash))
}

func (d *Director) GetImageChannel() chan *image.RGBA {
	return d.view.GetImageChannel()
}
