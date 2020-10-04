package encoder

import "image"

type InFrame struct {
	Image     *image.RGBA
	Timestamp uint32
}

type OutFrame struct {
	Data      []byte
	Timestamp uint32
}

type Encoder interface {
	GetInputChan() chan InFrame
	GetOutputChan() chan OutFrame
	Stop()
}
