package emulator

import (
	"image"
	"log"
	"os"
	"time"

	"github.com/giongto35/cloud-game/emulator/nes"
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
	// Hash represents a game state (roomID, gamePath).
	// It is used for save file name
	hash string
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
	// Generate hash that is indentifier of a room (game path + roomID)
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

func (d *Director) IsGameOnLocal(path string, roomID string) bool {
	hash, _ := hashFile(path, roomID)
	_, err := os.Open(savePath(hash))
	return err != nil
}

// SaveGame creates save events and doing extra step for load
func (d *Director) SaveGame(saveExtraFunc func() error) error {
	if d.hash != "" {
		d.view.Save(d.hash, saveExtraFunc)
		return nil
	} else {
		return nil
	}
}

// LoadGame creates load events and doing extra step for load
func (d *Director) LoadGame() error {
	if d.hash != "" {
		d.view.Load(d.hash)
		return nil
	} else {
		return nil
	}
}

// GetHash return hash
func (d *Director) GetHash() string {
	return d.hash
}

// GetHashPath return the full path to hash file
func (d *Director) GetHashPath() string {
	return savePath(d.hash)
}
