package image

import (
	"image"
	"image/color"

	"golang.org/x/image/draw"
)

const (
	// skips image interpolation
	ScaleSkip = -1
	// initial image interpolation algorithm
	ScaleOld = 0
	// nearest neighbour interpolation
	ScaleNearestNeighbour = 1
	// bilinear interpolation
	ScaleBilinear = 2
)

func Resize(scaleType int, fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, out *image.RGBA) {

	// !to implement own image interfaces img.Pix = bytes[]
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	toRgba(fn, w, h, packedW, bpp, data, src)

	// !to do set it once instead switching on each iteration
	// !to do skip resize if w=vw h=vh
	switch scaleType {
	case ScaleSkip:
		skip(fn, w, h, packedW, vw, vh, bpp, data, src, out)
	case ScaleNearestNeighbour:
		draw.NearestNeighbor.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
		//nearest(fn, w, h, packedW, vw, vh, bpp, data, src, out)
	case ScaleBilinear:
		draw.ApproxBiLinear.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
		//bilinear(fn, w, h, packedW, vw, vh, bpp, data, src, out)
	case ScaleOld:
		fallthrough
	default:
		old(fn, w, h, packedW, vw, vh, bpp, data, src, out)
	}
}

func old(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, _ *image.RGBA, out *image.RGBA) {
	seek := 0

	scaleWidth := float64(vw) / float64(w)
	scaleHeight := float64(vh) / float64(h)

	for y := 0; y < h; y++ {
		y2 := int(float64(y) * scaleHeight)
		for x := 0; x < packedW; x++ {
			x2 := int(float64(x) * scaleWidth)
			if x2 < vw {
				out.Set(x2, y2, fn(data, seek))
			}

			seek += bpp
		}
	}
}

func skip(fn Format, w int, h int, packedW int, _ int, _ int, bpp int, data []byte, _ *image.RGBA, out *image.RGBA) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			index := (y * packedW) + x
			index *= bpp
			out.Set(x, y, fn(data, index))
		}
	}
}

func nearest(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, _ *image.RGBA, out *image.RGBA) {
	xRatio := ((w << 16) / vw) + 1
	yRatio := ((h << 16) / vh) + 1

	for y := 0; y < vh; y++ {
		y2 := (y * yRatio) >> 16
		for x := 0; x < vw; x++ {
			x2 := (x * xRatio) >> 16

			index := (y2 * packedW) + x2
			index *= bpp

			out.Set(x, y, fn(data, index))
		}
	}
}

// This implementation has some color bleeding issues
func bilinear(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, _ *image.RGBA, out *image.RGBA) {
	xRatio := float64(w-1) / float64(vw)
	yRatio := float64(h-1) / float64(vh)

	for y := 0; y < vh; y++ {
		y2 := int(yRatio * float64(y))
		for x := 0; x < vw; x++ {
			x2 := int(xRatio * float64(x))

			w := (xRatio * float64(x)) - float64(x2)
			h := (yRatio * float64(y)) - float64(y2)

			index := (y2 * packedW) + x2

			a := fn(data, index*bpp)
			b := fn(data, (index+1)*bpp)
			c := fn(data, (index+packedW)*bpp)
			d := fn(data, (index+packedW+1)*bpp)

			out.Set(x, y, color.RGBA{
				// don't sink the boat
				R: byte(float64(a.R)*(1-w)*(1-h) + float64(b.R)*w*(1-h) + float64(c.R)*h*(1-w) + float64(d.R)*w*h),
				G: byte(float64(a.G)*(1-w)*(1-h) + float64(b.G)*w*(1-h) + float64(c.G)*h*(1-w) + float64(d.G)*w*h),
				B: byte(float64(a.B)*(1-w)*(1-h) + float64(b.B)*w*(1-h) + float64(c.B)*h*(1-w) + float64(d.B)*w*h),
				//A: byte(float64(a.A)*(1-w)*(1-h) + float64(b.A)*w*(1-h) + float64(c.A)*h*(1-w) + float64(d.A)*w*h),
			})
		}
	}
}

func toRgba(fn Format, w int, h int, packedW int, bpp int, data []byte, image *image.RGBA) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			index := (y*packedW + x) * bpp
			c := fn(data, index)
			i := (y-image.Rect.Min.Y)*image.Stride + (x-image.Rect.Min.X)*4
			s := image.Pix[i : i+4 : i+4]
			s[0] = c.R
			s[1] = c.G
			s[2] = c.B
			s[3] = c.A
		}
	}
}
