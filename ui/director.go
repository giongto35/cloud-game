package ui

import (
	"image"
	"log"
	"time"

	"github.com/giongto35/cloud-game/nes"
	// "github.com/gordonklaus/portaudio"
)

type Director struct {
	// audio        *Audio
	view         *GameView
	timestamp    float64
	imageChannel chan *image.RGBA
	audioChannel chan float32
	inputChannel chan int
	Done         chan struct{}

	roomID string
	hash   string
}

const FPS = 60

func NewDirector(roomID string, imageChannel chan *image.RGBA, audioChannel chan float32, inputChannel chan int) *Director {
	director := Director{}
	director.Done = make(chan struct{})
	director.audioChannel = audioChannel
	director.imageChannel = imageChannel
	director.inputChannel = inputChannel
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

//func (d *Director) UpdateInput(input int) {
//d.view.UpdateInput(input)
//}

func (d *Director) Step() {
	timestamp := float64(time.Now().Nanosecond()) / float64(time.Second)
	dt := timestamp - d.timestamp
	d.timestamp = timestamp
	if d.view != nil {
		d.view.Update(timestamp, dt)
	}
}

func (d *Director) Start(paths []string) {
	// portaudio.Initialize()
	// defer portaudio.Terminate()

	// audio := NewAudio()
	// audio.Start()
	// d.audio = audio

	if len(paths) == 1 {
		d.PlayGame(paths[0])
	}
	d.Run()
}

func (d *Director) Run() {
	c := time.Tick(time.Second / FPS)
L:
	for range c {
	// for {
		// quit game
		// TODO: Anyway not using select because it will slow down
		select {
		// if there is event from close channel => the game is ended
		//case input := <-d.inputChannel:
		//d.UpdateInput(input)
		case <-d.Done:
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
	d.SetView(NewGameView(console, path, hash, d.imageChannel, d.audioChannel, d.inputChannel))
}

func (d *Director) SaveGame() error {
	if d.hash != "" {
		d.view.Save(d.hash)
		return nil
	} else {
		return nil
	}
}

func (d *Director) LoadGame() error {
	if d.hash != "" {
		d.view.Load(d.hash)
		return nil
	} else {
		return nil
	}
}
