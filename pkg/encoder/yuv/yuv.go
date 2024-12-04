package yuv

import (
	"image"

	"github.com/giongto35/cloud-game/v3/pkg/encoder/yuv/libyuv"
)

type Conv struct {
	w, h    int
	sw, sh  int
	scale   float64
	frame   []byte
	frameSc []byte
}

type RawFrame struct {
	Data   []byte
	Stride int
	W, H   int
}

type PixFmt uint32

const FourccRgbp = libyuv.FourccRgbp
const FourccArgb = libyuv.FourccArgb
const FourccAbgr = libyuv.FourccAbgr
const FourccRgb0 = libyuv.FourccRgb0

func NewYuvConv(w, h int, scale float64) Conv {
	if scale < 1 {
		scale = 1
	}

	sw, sh := round(w, scale), round(h, scale)
	conv := Conv{w: w, h: h, sw: sw, sh: sh, scale: scale}
	bufSize := int(float64(w) * float64(h) * 1.5)

	if scale == 1 {
		conv.frame = make([]byte, bufSize)
	} else {
		bufSizeSc := int(float64(sw) * float64(sh) * 1.5)
		// [original frame][scaled frame          ]
		frames := make([]byte, bufSize+bufSizeSc)
		conv.frame = frames[:bufSize]
		conv.frameSc = frames[bufSize:]
	}

	return conv
}

// Process converts an image to YUV I420 format inside the internal buffer.
func (c *Conv) Process(frame RawFrame, rot uint, pf PixFmt) []byte {
	cx, cy := c.w, c.h // crop
	if rot == 90 || rot == 270 {
		cx, cy = cy, cx
	}

	var stride int
	switch pf {
	case PixFmt(libyuv.FourccRgbp), PixFmt(libyuv.FourccRgb0):
		stride = frame.Stride >> 1
	default:
		stride = frame.Stride >> 2
	}

	libyuv.Y420(frame.Data, c.frame, frame.W, frame.H, stride, c.w, c.h, rot, uint32(pf), cx, cy)

	if c.scale > 1 {
		libyuv.Y420Scale(c.frame, c.frameSc, c.w, c.h, c.sw, c.sh)
		return c.frameSc
	}

	return c.frame
}

func (c *Conv) Version() string      { return libyuv.Version() }
func round(x int, scale float64) int { return (int(float64(x)*scale) + 1) & ^1 }

func ToYCbCr(bytes []byte, w, h int) *image.YCbCr {
	cw, ch := (w+1)/2, (h+1)/2

	i0 := w*h + 0*cw*ch
	i1 := w*h + 1*cw*ch
	i2 := w*h + 2*cw*ch

	yuv := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	yuv.Y = bytes[:i0:i0]
	yuv.Cb = bytes[i0:i1:i1]
	yuv.Cr = bytes[i1:i2:i2]
	return yuv
}
