package image

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkDraw(b *testing.B) {
	w1, h1 := 256, 240
	w2, h2 := 640, 480

	type args struct {
		encoding  uint32
		rot       Rotation
		scaleType int
		w         int
		h         int
		packedW   int
		bpp       int
		data      []byte
		dw        int
		dh        int
		th        int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "565_0th",
			args: args{
				encoding: BitFormatShort565, scaleType: ScaleNearestNeighbour,
				w: w1, h: h1, packedW: w1, bpp: 2, data: make([]uint8, w1*h1*2), dw: w1, dh: h1, th: 0,
			},
		},
		{
			name: "565_0th_90",
			args: args{
				encoding: BitFormatShort565, rot: A90, scaleType: ScaleNearestNeighbour,
				w: h1, h: w1, packedW: h1, bpp: 2, data: make([]uint8, w1*h1*2), dw: w1, dh: h1, th: 0,
			},
		},
		{
			name: "565_0th",
			args: args{
				encoding: BitFormatShort565, scaleType: ScaleNearestNeighbour,
				w: w2, h: h2, packedW: w1, bpp: 2, data: make([]uint8, w2*h2*2), dw: w2, dh: h2, th: 0,
			},
		},
		{
			name: "565_4th",
			args: args{
				encoding: BitFormatShort565, scaleType: ScaleNearestNeighbour,
				w: w1, h: h1, packedW: w1, bpp: 2, data: make([]uint8, w1*h1*2), dw: w1, dh: h1, th: 4,
			},
		},
		{
			name: "565_4th",
			args: args{
				encoding: BitFormatShort565, scaleType: ScaleNearestNeighbour,
				w: w2, h: h2, packedW: w2, bpp: 2, data: make([]uint8, w2*h2*2), dw: w2, dh: h2, th: 4,
			},
		},
		{
			name: "8888 - 0th",
			args: args{
				encoding: BitFormatInt8888Rev, scaleType: ScaleNearestNeighbour,
				w: w1, h: h1, packedW: w1, bpp: 4, data: make([]uint8, w1*h1*4), dw: w1, dh: h1, th: 0,
			},
		},
		{
			name: "8888 - 4th",
			args: args{
				encoding: BitFormatInt8888Rev, scaleType: ScaleNearestNeighbour,
				w: w1, h: h1, packedW: w1, bpp: 4, data: make([]uint8, w1*h1*4), dw: w1, dh: h1, th: 4,
			},
		},
	}

	for _, bn := range tests {
		c := NewCanvas(bn.args.dw, bn.args.dh, bn.args.dw*bn.args.dh)
		img := c.Get(bn.args.dw, bn.args.dh)
		c.Put(img)
		img2 := c.Get(bn.args.dw, bn.args.dh)
		c.Put(img2)
		b.ResetTimer()
		b.Run(fmt.Sprintf("%vx%v_%v", bn.args.w, bn.args.h, bn.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := c.Draw(bn.args.encoding, bn.args.rot, bn.args.w, bn.args.h, bn.args.packedW, bn.args.bpp, bn.args.data, bn.args.th)
				c.Put(p)
			}
			b.ReportAllocs()
		})
	}
}

func Test_ix8888(t *testing.T) {
	type args struct {
		dst    *uint32
		px     uint32
		expect uint32
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				dst:    new(uint32),
				px:     0x11223344,
				expect: 0x00443322,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*tt.args.dst = _8888rev(tt.args.px)
			if *tt.args.dst != tt.args.expect {
				t.Errorf("nope, %x %x", *tt.args.dst, tt.args.expect)
			}
		})
	}
}

type dimensions struct {
	w int
	h int
}

func TestRotate(t *testing.T) {
	tests := []struct {
		// packed bytes from a 2D matrix
		input []byte
		// original matrix's width
		w int
		// original matrix's height
		h int
		// rotation algorithm
		rotateHow []Rotation
		expected  [][]byte
	}{
		{
			// a cross
			[]byte{
				0, 1, 0,
				1, 1, 1,
				0, 1, 0,
			},
			3, 3, []Rotation{0, A90, A180, A270},
			[][]byte{
				{
					0, 1, 0,
					1, 1, 1,
					0, 1, 0,
				},
				{
					0, 1, 0,
					1, 1, 1,
					0, 1, 0,
				},
				{
					0, 1, 0,
					1, 1, 1,
					0, 1, 0,
				},
				{
					0, 1, 0,
					1, 1, 1,
					0, 1, 0,
				},
			},
		},
		{
			[]byte{
				1, 2,
				3, 4,
				5, 6,
				7, 8,
			},
			2, 4, []Rotation{0, A90, A180, A270},
			[][]byte{
				{
					1, 2,
					3, 4,
					5, 6,
					7, 8,
				},
				{
					2, 4, 6, 8,
					1, 3, 5, 7,
				},
				{
					8, 7,
					6, 5,
					4, 3,
					2, 1,
				},
				{
					7, 5, 3, 1,
					8, 6, 4, 2,
				},
			},
		},
		{
			// a square
			[]byte{
				1, 0, 0, 0, 0, 0, 0, 0,
				0, 1, 1, 1, 1, 1, 1, 0,
				0, 1, 1, 1, 1, 1, 1, 0,
				0, 1, 0, 0, 0, 0, 1, 0,
				0, 1, 1, 1, 1, 1, 1, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
			},
			8, 6, []Rotation{0, A90, A180, A270},
			[][]byte{
				{
					// L              // R
					1, 0, 0, 0, 0, 0, 0, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 1, 0, 0, 0, 0, 1, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 0, 0, 0, 0, 0, 0, 1,
				},
				{
					0, 0, 0, 0, 0, 1,
					0, 1, 1, 1, 1, 0,
					0, 1, 1, 0, 1, 0,
					0, 1, 1, 0, 1, 0,
					0, 1, 1, 0, 1, 0,
					0, 1, 1, 0, 1, 0,
					0, 1, 1, 1, 1, 0,
					1, 0, 0, 0, 0, 0,
				},

				{
					1, 0, 0, 0, 0, 0, 0, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 1, 0, 0, 0, 0, 1, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 1, 1, 1, 1, 1, 1, 0,
					0, 0, 0, 0, 0, 0, 0, 1,
				},
				{
					0, 0, 0, 0, 0, 1,
					0, 1, 1, 1, 1, 0,
					0, 1, 0, 1, 1, 0,
					0, 1, 0, 1, 1, 0,
					0, 1, 0, 1, 1, 0,
					0, 1, 0, 1, 1, 0,
					0, 1, 1, 1, 1, 0,
					1, 0, 0, 0, 0, 0,
				},
			},
		},
	}

	for _, test := range tests {
		for i, rot := range test.rotateHow {
			if output := exampleRotate(test.input, test.w, test.h, rot); !bytes.Equal(output, test.expected[i]) {
				t.Errorf(
					"Test fail for angle %v with %v that should be \n%v but it's \n%v",
					rot, test.input, test.expected[i], output)
			}
		}
	}
}

func TestBoundsAfterRotation(t *testing.T) {
	tests := []struct {
		dim       []dimensions
		rotateHow []Rotation
	}{
		{
			// a combinatorics lib would be nice instead
			[]dimensions{
				// square
				{w: 100, h: 100},
				// even w/h
				{w: 100, h: 50},
				// even h/w
				{w: 50, h: 100},
				// odd even w/h
				{w: 77, h: 32},
				// even odd h/w
				{w: 32, h: 77},
				// just odd
				{w: 13, h: 19},
			},
			[]Rotation{0, A90, A180, A270},
		},
	}

	for _, test := range tests {
		for _, rot := range test.rotateHow {
			for _, dim := range test.dim {

				for y := 0; y < dim.h; y++ {
					for x := 0; x < dim.w; x++ {

						xx, yy := rotate(int(rot), x, y, dim.w, dim.h)

						if rot == A90 || rot == A270 { // is even
							yy, xx = xx, yy
						}

						if xx < 0 || xx > dim.w {
							t.Errorf("Rot %v, coordinate x should be in range [0; %v]: %v", rot, dim.w-1, xx)
						}

						if yy < 0 || yy > dim.h {
							t.Errorf("Rot %v, coordinate y should be in range [0; %v]: %v", rot, dim.h-1, yy)
						}
					}
				}
			}
		}
	}
}

// exampleRotate is an example of rotation usage.
//
//	[1 2 3 4 5 6 7 8 9]
//	[7 4 1 8 5 2 9 6 3]
func exampleRotate(data []uint8, w int, h int, rot Rotation) []uint8 {
	dest := make([]uint8, len(data))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx, ny := rotate(int(rot), x, y, w, h)
			stride := w
			if rot == A90 || rot == A270 { // is even
				stride = h
			}
			dest[nx+ny*stride] = data[x+y*w]
		}
	}
	return dest
}
