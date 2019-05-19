package emulator

import (
	"image"
	"log"
	"time"

	"github.com/giongto35/cloud-game/emulator/nes"
	// "github.com/gordonklaus/portaudio"
)

// Director is the nes emulator
type Director struct {
	// audio        *Audio
	view         *GameView
	timestamp    float64
	imageChannel chan<- *image.RGBA
	audioChannel chan<- float32
	inputChannel <-chan int
	Done         chan struct{}

	roomID string
}

const fps = 60

// NewDirector returns a new director
func NewDirector(roomID string, imageChannel chan<- *image.RGBA, audioChannel chan<- float32, inputChannel <-chan int) *Director {
	// TODO: return image channel from where it write
	director := Director{}
	director.Done = make(chan struct{})
	director.audioChannel = audioChannel
	director.imageChannel = imageChannel
	director.inputChannel = inputChannel
	director.roomID = roomID
	return &director
}

// SetView ...
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

// Step ...
func (d *Director) Step() {
	timestamp := float64(time.Now().Nanosecond()) / float64(time.Second)
	dt := timestamp - d.timestamp
	d.timestamp = timestamp
	if d.view != nil {
		d.view.Update(timestamp, dt)
	}
}

// Start ...
func (d *Director) Start(paths []string) {
	// portaudio.Initialize()
	// defer portaudio.Terminate()

	// audio := NewAudio()
	// audio.Start()
	// d.audio = audio
	log.Println("Start game: ", paths)

	if len(paths) == 1 {
		d.PlayGame(paths[0])
	}
	d.Run()
}

// Run ...
func (d *Director) Run() {
	c := time.Tick(time.Second / fps)
L:
	for range c {
		// for {
		// quit game
		// TODO: How to not using select because it will slow down
		select {
		// if there is event from close channel => the game is ended
		//case input := <-d.inputChannel:
		//d.UpdateInput(input)
		case <-d.Done:
			log.Println("Closing Director")
			break L
		default:
		}

		d.Step()
	}
	d.SetView(nil)
	log.Println("Closed Director")
}

// PalyGame starts a game given a rom path
func (d *Director) PlayGame(path string) {
	console, err := nes.NewConsole(path)
	if err != nil {
		log.Println("Err: Cannot load path, Got:", err)
	}
	// Set GameView as current view
	d.SetView(NewGameView(console, path, d.roomID, d.imageChannel, d.audioChannel, d.inputChannel))
}

// SaveGame creates save events and doing extra step for load
func (d *Director) SaveGame(saveExtraFunc func() error) error {
	if d.roomID != "" {
		d.view.Save(d.roomID, saveExtraFunc)
		return nil
	} else {
		return nil
	}
}

// LoadGame creates load events and doing extra step for load
func (d *Director) LoadGame() error {
	if d.roomID != "" {
		d.view.Load(d.roomID)
		return nil
	} else {
		return nil
	}
}

// GetHashPath return the full path to hash file
func (d *Director) GetHashPath() string {
	return savePath(d.roomID)
}
