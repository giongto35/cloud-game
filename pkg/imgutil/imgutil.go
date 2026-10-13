// Package imgutil validates user uploaded pictures (avatars, game covers):
// the file is decoded, optionally cropped, scaled down and encoded again as PNG,
// so nothing but pixels ever reaches the disk or other users.
package imgutil

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	// decoders accepted for uploads
	_ "image/gif"
	_ "image/jpeg"
)

var (
	ErrInvalid  = errors.New("invalid image")
	ErrTooLarge = errors.New("image too large")
)

const (
	maxSide   = 12000
	maxPixels = 40_000_000
)

// Options controls how an upload is normalized.
type Options struct {
	// Square crops the picture to a centered square (avatars).
	Square bool
	// MaxSide is the biggest width/height of the result; larger pictures are scaled down.
	MaxSide int
}

// Process decodes data (PNG, JPEG or GIF) and returns it as a PNG.
func Process(data []byte, o Options) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalid
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxSide || cfg.Height > maxSide || cfg.Width*cfg.Height > maxPixels {
		return nil, ErrTooLarge
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalid
	}
	b := src.Bounds()
	if o.Square {
		s := min(b.Dx(), b.Dy())
		x0 := b.Min.X + (b.Dx()-s)/2
		y0 := b.Min.Y + (b.Dy()-s)/2
		b = image.Rect(x0, y0, x0+s, y0+s)
	}
	w, h := b.Dx(), b.Dy()
	if o.MaxSide > 0 && max(w, h) > o.MaxSide {
		if w >= h {
			h, w = max(1, h*o.MaxSide/w), o.MaxSide
		} else {
			w, h = max(1, w*o.MaxSide/h), o.MaxSide
		}
	}
	out := scale(src, b, w, h)

	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// scale resamples the area r of src to w x h pixels (area average when shrinking,
// bilinear when enlarging), over an opaque black background so transparency never leaks.
func scale(src image.Image, r image.Rectangle, w, h int) *image.RGBA {
	// flatten the source area first
	flat := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(flat, flat.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), src, r.Min, draw.Over)
	if w == r.Dx() && h == r.Dy() {
		return flat
	}

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := float64(r.Dx()), float64(r.Dy())
	xr, yr := sw/float64(w), sh/float64(h)
	for y := 0; y < h; y++ {
		y0, y1 := float64(y)*yr, float64(y+1)*yr
		for x := 0; x < w; x++ {
			x0, x1 := float64(x)*xr, float64(x+1)*xr
			var rs, gs, bs, ws float64
			// box covering [x0,x1) x [y0,y1) with fractional edge weights
			for sy := int(y0); sy < int(ceil(y1)) && sy < r.Dy(); sy++ {
				wy := overlap(float64(sy), float64(sy+1), y0, y1)
				for sx := int(x0); sx < int(ceil(x1)) && sx < r.Dx(); sx++ {
					wx := overlap(float64(sx), float64(sx+1), x0, x1)
					c := flat.RGBAAt(sx, sy)
					wgt := wx * wy
					rs += float64(c.R) * wgt
					gs += float64(c.G) * wgt
					bs += float64(c.B) * wgt
					ws += wgt
				}
			}
			if ws > 0 {
				out.SetRGBA(x, y, color.RGBA{uint8(rs/ws + 0.5), uint8(gs/ws + 0.5), uint8(bs/ws + 0.5), 255})
			}
		}
	}
	return out
}

func ceil(f float64) float64 {
	i := float64(int(f))
	if i < f {
		return i + 1
	}
	return i
}

func overlap(a0, a1, b0, b1 float64) float64 {
	lo, hi := max(a0, b0), min(a1, b1)
	if hi <= lo {
		return 0
	}
	return hi - lo
}
