package media

import "sync"

const bufSize = 3000

type ch struct{ l, r []int16 }

var (
	bufHalfPool     sync.Pool
	BufOutAudioPool = sync.Pool{New: func() any { b := make([]int16, bufSize); return &b }}
)

// ResampleStretch does a simple stretching of audio samples.
func ResampleStretch(pcm []int16, size int) []int16 {
	hs := size >> 1
	lr, _ := bufHalfPool.Get().(*ch)
	if lr == nil {
		lr = &ch{l: make([]int16, bufSize/2), r: make([]int16, bufSize/2)}
	}
	ratio := float32(size) / float32(len(pcm))
	for i, o, n := 0, 0, len(pcm)-1; i < n; i += 2 {
		o = int(float32(i>>1) * ratio)
		lr.r[o], lr.l[o] = pcm[i], pcm[i+1]
	}
	audio := (*BufOutAudioPool.Get().(*[]int16))[:size]
	audio[0], audio[1] = lr.r[0], lr.l[0]
	for i, x := 1, 0; i < hs; i++ {
		if lr.r[i] == 0 {
			lr.r[i] = lr.r[i-1]
		}
		if lr.l[i] == 0 {
			lr.l[i] = lr.l[i-1]
		}
		x = i << 1
		audio[x], audio[x+1] = lr.r[i], lr.l[i]
		lr.r[i-1] = 0
		lr.l[i-1] = 0
	}
	lr.r[hs-1] = 0
	lr.l[hs-1] = 0
	lr.r[0] = 0
	lr.l[0] = 0
	bufHalfPool.Put(lr)
	return audio
}
