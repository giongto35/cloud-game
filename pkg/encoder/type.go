package encoder

import (
	"image"
	"time"
)

type InFrame struct {
	Image    *image.RGBA
	Duration time.Duration
}

type OutFrame struct {
	Data     []byte
	Duration time.Duration
}

type Encoder interface {
	Encode(input []byte) []byte
	Shutdown() error
}
