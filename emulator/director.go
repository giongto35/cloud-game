package emulator

import (
	"image"
	"log"
	"time"

	"github.com/giongto35/cloud-game/config"
	"github.com/giongto35/cloud-game/emulator/nes"
	"github.com/giongto35/cloud-game/util"
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

	gamePath string
	roomID   string
}

const fps = 300

// NewDirector returns a new director
func NewDirector(roomID string, imageChannel chan<- *image.RGBA, audioChannel chan<- float32, inputChannel <-chan int) CloudEmulator {
	// TODO: return image channel from where it write
	director := Director{}
	director.Done = make(chan struct{}, 1)
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

func (d *Director) LoadMeta(path string) config.EmulatorMeta {
	log.Println("Start game: ", path)

	d.gamePath = path
	return config.EmulatorMeta{
		AudioSampleRate: 48000,
		Fps:             300,
		Width:           256,
		Height:          240,
	}
}

// Start ...
func (d *Director) Start() {
	// portaudio.Initialize()
	// defer portaudio.Terminate()

	// audio := NewAudio()
	// audio.Start()
	// d.audio = audio
	log.Println("Start game: ", d.gamePath)

	d.playGame(d.gamePath)
	d.run()
}

// step ...
func (d *Director) step() {
	timestamp := float64(time.Now().Nanosecond()) / float64(time.Second)
	dt := timestamp - d.timestamp
	d.timestamp = timestamp
	if d.view != nil {
		d.view.Update(timestamp, dt)
	}
}

// run ...
func (d *Director) run() {
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

		d.step()
	}
	d.SetView(nil)
	log.Println("Closed Director")
}

// PalyGame starts a game given a rom path
func (d *Director) playGame(path string) {
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
		d.view.Save(saveExtraFunc)
		return nil
	}

	return nil
}

// LoadGame creates load events and doing extra step for load
func (d *Director) LoadGame() error {
	if d.roomID != "" {
		d.view.Load()
		return nil
	}

	return nil
}

// GetHashPath return the full path to hash file
func (d *Director) GetHashPath() string {
	return util.GetSavePath(d.roomID)
}

func (d *Director) GetSampleRate() uint {
	return SampleRate
}

// Close
func (d *Director) Close() {
	close(d.Done)
}
