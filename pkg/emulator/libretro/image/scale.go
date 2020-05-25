package image

import (
	"golang.org/x/image/draw"
	"image"
)

const (
	// skips image interpolation
	ScaleNot = iota
	// nearest neighbour interpolation
	ScaleNearestNeighbour
	// bilinear interpolation
	ScaleBilinear
)

func Resize(scaleType int, src *image.RGBA, out *image.RGBA) {
	// !to do set it once instead switching on each iteration
	// !to do skip resize if w=vw h=vh
	switch scaleType {
	case ScaleBilinear:
		draw.ApproxBiLinear.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
	case ScaleNot:
		fallthrough
	case ScaleNearestNeighbour:
		fallthrough
	default:
		draw.NearestNeighbor.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
	}
}
