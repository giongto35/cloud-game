// credit to https://github.com/fogleman/nes
package ui

import (
	"image"
	// "log"
	"runtime"
	// "github.com/gordonklaus/portaudio"
)

const (
	width  = 256
	height = 240
	scale  = 3
	title  = "NES"
)

func init() {
	// we need to keep OpenGL calls on a single thread
	runtime.LockOSThread()
}

func Run(paths []string, roomID string, imageChannel chan *image.RGBA, inputChannel chan int) {
	// TODO: stream audio
	// initialize audio
	// portaudio.Initialize()
	// defer portaudio.Terminate()

	// audio := NewAudio()
	// if err := audio.Start(); err != nil {
	// 	log.Fatalln(err)
	// }
	// defer audio.Stop()

	// run director
	director := NewDirector(roomID, imageChannel, inputChannel)
	// director := NewDirector(audio, imageChannel, inputChannel)
	director.Start(paths)
}
