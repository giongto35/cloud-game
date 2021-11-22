package img

import "image"

// ORGBA enforces image.RGBA to remove alpha channel when encoding PNGs.
type ORGBA struct {
	*image.RGBA
}

func (*ORGBA) Opaque() bool { return true }
