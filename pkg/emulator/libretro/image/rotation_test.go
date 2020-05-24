package image

import (
	"bytes"
	"testing"
)

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
		rotateHow []Angle
		expected  [][]byte
	}{
		{
			// a cross
			[]byte{
				0, 1, 0,
				1, 1, 1,
				0, 1, 0,
			},
			3, 3, []Angle{Angle0, Angle90, Angle180, Angle270},
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
			2, 4, []Angle{Angle0, Angle90, Angle180, Angle270},
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
			8, 6, []Angle{Angle0, Angle90, Angle180, Angle270},
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
			if output := ExampleRotate(test.input, test.w, test.h, rot); bytes.Compare(output, test.expected[i]) != 0 {
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
		rotateHow []Angle
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
			[]Angle{Angle0, Angle90, Angle180, Angle270},
		},
	}

	for _, test := range tests {
		for _, rot := range test.rotateHow {
			rotationFn := Angles[rot]
			for _, dim := range test.dim {

				for y := 0; y < dim.h; y++ {
					for x := 0; x < dim.w; x++ {

						xx, yy := rotationFn.Call(x, y, dim.w, dim.h)

						if rotationFn.IsEven {
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

func BenchmarkDirect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p1, p2 := Rotate90(1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

func BenchmarkLiteral(b *testing.B) {
	fn := Rotate90
	for i := 0; i < b.N; i++ {
		p1, p2 := fn(1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

func BenchmarkAssign(b *testing.B) {
	fn := Angles[Angle90].Call
	for i := 0; i < b.N; i++ {
		p1, p2 := fn(1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

func BenchmarkMapReassign(b *testing.B) {
	fn := Angles[Angle90].Call
	for i := 0; i < b.N; i++ {
		fn2 := fn
		p1, p2 := fn2(1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

func BenchmarkMapDirect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p1, p2 := Angles[Angle90].Call(1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

func BenchmarkNewMapDirect(b *testing.B) {
	fns := map[Angle]func(x, y, w, h int) (int, int){
		Angle90: Rotate90,
	}

	for i := 0; i < b.N; i++ {
		p1, p2 := fns[Angle90](1, 1, 2, 2)
		p1, p2 = p2, p1
	}
}

