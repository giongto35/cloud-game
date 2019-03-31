package ui

import (
	"runtime"
)

const (
	width  = 256
	height = 240
	scale  = 3
	title  = "NES"
)

func init() {
	// we need a parallel OS thread to avoid audio stuttering
	runtime.GOMAXPROCS(2)

	// we need to keep OpenGL calls on a single thread
	runtime.LockOSThread()
}

func Run(path string) {
	// initialize audio
	//portaudio.Initialize()
	//defer portaudio.Terminate()

	//audio := NewAudio()
	//if err := audio.Start(); err != nil {
	//log.Fatalln(err)
	//}
	//defer audio.Stop()

	// run director
}
