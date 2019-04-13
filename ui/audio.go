package ui

import (
	"github.com/gordonklaus/portaudio"
	"log"

	"gopkg.in/hraban/opus.v2"
	// "github.com/xlab/opus-go/opus"
)


type Audio struct {
	stream         *portaudio.Stream
	sampleRate     float64
	outputChannels int
	channel        chan float32
}

func NewAudio() *Audio {
	a := Audio{}
	a.channel = make(chan float32, 16000)
	return &a
}

func (a *Audio) Start() error {
	parameters := portaudio.StreamParameters{}
	parameters.SampleRate = 44100

	// host, err := portaudio.DefaultHostApi()
	// if err != nil {
	// 	return err
	// }

	// parameters := portaudio.HighLatencyParameters(nil, host.DefaultOutputDevice)
	stream, err := portaudio.OpenDefaultStream(0, 1, 48000, 0, a.Callback)
	// stream, err := portaudio.OpenStream(parameters, a.Callback)
	if err != nil {
		return err
	}
	if err := stream.Start(); err != nil {
		return err
	}
	a.stream = stream
	a.sampleRate = parameters.SampleRate
	// a.outputChannels = parameters.Output.Channels
	a.outputChannels = 1

	log.Println(a.sampleRate, a.outputChannels, parameters.FramesPerBuffer)

	return nil
}

func (a *Audio) Stop() error {
	return a.stream.Close()
}

func (a *Audio) Callback(out []float32) {
	var output float32
	log.Println(len(out))
	for i := range out {
		if i%a.outputChannels == 1 {
			select {
			case sample := <-a.channel:
				output = sample
			default:
				output = 0
			}
		}
		out[i] = output
	}

	enc, err := opus.NewEncoder(48000, 1, opus.AppVoIP)
	if err != nil {
		log.Println("[!] Cannot create audio encoder", err)
		return
	}
	data := make([]byte, 1000)
	n, err := enc.EncodeFloat32(out, data)

	if err != nil {
		log.Println("[!] Failed to decode")
		return
	}
	data = data[:n]
}