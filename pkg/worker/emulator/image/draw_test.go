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
		flipV     bool
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
		b.Run(fmt.Sprintf("%v", bn.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				DrawRgbaImage(bn.args.encoding, bn.args.rot, bn.args.scaleType, bn.args.w, bn.args.h, bn.args.packedW, bn.args.bpp, bn.args.data, bn.args.dw, bn.args.dh, bn.args.th)
			}
			b.ReportAllocs()
		})
	}
}
