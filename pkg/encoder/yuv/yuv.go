package yuv

import (
	"image"
	"sync"

	"github.com/giongto35/cloud-game/v3/pkg/encoder/yuv/libyuv"
)

type Conv struct {
	w, h   int
	sw, sh int
	scale  float64
	pool   sync.Pool
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

func NewYuvConv(w, h int, scale float64) Conv {
	if scale < 1 {
		scale = 1
	}
	sw, sh := round(w, scale), round(h, scale)
	bufSize := int(float64(sw) * float64(sh) * 1.5)
	return Conv{
		w: w, h: h, sw: sw, sh: sh, scale: scale,
		pool: sync.Pool{New: func() any { b := make([]byte, bufSize); return &b }},
	}
}

// Process converts an image to YUV I420 format inside the internal buffer.
func (c *Conv) Process(frame RawFrame, rot uint, pf PixFmt) []byte {
	dx, dy := c.w, c.h // dest
	cx, cy := c.w, c.h // crop
	if rot == 90 || rot == 270 {
		cx, cy = cy, cx
	}

	stride := frame.Stride >> 2
	if pf == PixFmt(libyuv.FourccRgbp) {
		stride = frame.Stride >> 1
	}

	buf := *c.pool.Get().(*[]byte)
	libyuv.Y420(frame.Data, buf, frame.W, frame.H, stride, dx, dy, rot, uint32(pf), cx, cy)

	if c.scale > 1 {
		dstBuf := *c.pool.Get().(*[]byte)
		libyuv.Y420Scale(buf, dstBuf, dx, dy, c.sw, c.sh)
		c.pool.Put(&buf)
		return dstBuf
	}
	return buf
}

func (c *Conv) Put(x *[]byte)        { c.pool.Put(x) }
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
