package image

import (
	"image"
	"sync"
)

type imageCache struct {
	image *image.RGBA
	w     int
	h     int
}

func (i *imageCache) get(w, h int) *image.RGBA {
	if i.w == w && i.h == h {
		return i.image
	}
	i.w, i.h = w, h
	i.image = image.NewRGBA(image.Rect(0, 0, w, h))
	return i.image
}

var (
	canvas1 = imageCache{image.NewRGBA(image.Rectangle{}), 0, 0}
	canvas2 = imageCache{image.NewRGBA(image.Rectangle{}), 0, 0}
	wg      sync.WaitGroup
)

func DrawRgbaImage(pixFormat Format, rot *Rotate, scaleType int, flipV bool, w, h, packedW, bpp int,
	data []byte, dw, dh, th int) *image.RGBA {
	// !to implement own image interfaces img.Pix = bytes[]
	ww, hh := w, h
	if rot != nil && rot.IsEven {
		ww, hh = hh, ww
	}
	src := canvas1.get(ww, hh)

	normY := !flipV
	hn := h / th
	pwb := packedW * bpp
	wg.Add(th)
	for i := 0; i < th; i++ {
		xx := hn * i
		go func() {
			for y, yy, l, lx, row := xx, 0, xx+hn, 0, 0; y < l; y++ {
				if normY {
					yy = y
				} else {
					yy = (h - 1) - y
				}
				row = yy * src.Stride
				lx = y * pwb
				for x, k := 0, 0; x < w; x++ {
					if rot == nil {
						k = x<<2 + row
					} else {
						dx, dy := rot.Call(x, yy, w, h)
						k = dx<<2 + dy*src.Stride
					}
					r := pixFormat(data, x*bpp+lx)
					src.Pix[k], src.Pix[k+1], src.Pix[k+2], src.Pix[k+3] = r.R, r.G, r.B, 255
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if ww == dw && hh == dh {
		return src
	} else {
		out := canvas2.get(dw, dh)
		Resize(scaleType, src, out)
		return out
	}
}

func Clear() {
	wg = sync.WaitGroup{}
	canvas1.get(0, 0)
	canvas2.get(0, 0)
}
