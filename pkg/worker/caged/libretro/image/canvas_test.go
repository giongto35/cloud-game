package image

import (
	"fmt"
	"testing"
)

func BenchmarkDraw(b *testing.B) {
	type args struct {
		encoding  uint32
		rot       *Rotate
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
			name: "0th",
			args: args{
				encoding:  BitFormatInt8888Rev,
				rot:       nil,
				scaleType: ScaleNearestNeighbour,
				w:         256,
				h:         240,
				packedW:   256,
				bpp:       4,
				data:      make([]uint8, 256*240*4),
				dw:        256,
				dh:        240,
				th:        0,
			},
		},
		{
			name: "4th",
			args: args{
				encoding:  BitFormatInt8888Rev,
				rot:       nil,
				scaleType: ScaleNearestNeighbour,
				w:         256,
				h:         240,
				packedW:   256,
				bpp:       4,
				data:      make([]uint8, 256*240*4),
				dw:        256,
				dh:        240,
				th:        4,
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
		b.Run(fmt.Sprintf("%v", bn.name), func(b *testing.B) {
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
			ix8888(tt.args.dst, tt.args.px)
			if *tt.args.dst != tt.args.expect {
				t.Errorf("nope, %x %x", *tt.args.dst, tt.args.expect)
			}
		})
	}
}
