package encoder

import "image"

type Encoder interface {
	GetInputChan() chan *image.RGBA
	GetOutputChan() chan []byte
	Stop()
}
