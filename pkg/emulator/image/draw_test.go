package image

import (
	"math/rand"
	"testing"
	"unsafe"
)

func Benchmark8881(b *testing.B) {
	rand.Seed(101)
	token := make([]byte, 128)
	rez := make([][]byte, 128/4)
	for i := range rez {
		rez[i] = make([]byte, 4)
	}
	rand.Read(token)

	for i := 0; i < b.N; i++ {
		for o := 0; o < len(token); o += 4 {
			idx := o >> 4
			rez[idx][0] = token[o+2]
			rez[idx][1] = token[o+1]
			rez[idx][2] = token[o]
		}
	}
}

func Benchmark8882(b *testing.B) {
	rand.Seed(101)
	token := make([]byte, 128)
	rez := make([][]byte, 128/4)
	for i := range rez {
		rez[i] = make([]byte, 4)
	}
	rand.Read(token)

	for i := 0; i < b.N; i++ {
		for o := 0; o < len(token); o += 4 {
			px := *(*uint32)(unsafe.Pointer(&token[o]))
			dst := (*uint32)(unsafe.Pointer(&rez[o/4]))

			*dst = ((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000) // | 0xff000000
		}
	}
}
