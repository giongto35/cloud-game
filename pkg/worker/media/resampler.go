package media

import "sync"

const bufSize = 3000

var (
	bufHalfPool     = sync.Pool{New: func() any { b := make([]int16, bufSize/2); return &b }}
	BufOutAudioPool = sync.Pool{New: func() any { b := make([]int16, bufSize); return &b }}
)

// ResampleStretch does a simple stretching of audio samples.
func ResampleStretch(pcm []int16, size int) []int16 {
	hs := size >> 1
	r := *bufHalfPool.Get().(*[]int16)
	l := *bufHalfPool.Get().(*[]int16)
	ratio := float32(size) / float32(len(pcm))
	for i, o, n := 0, 0, len(pcm)-1; i < n; i += 2 {
		o = int(float32(i>>1) * ratio)
		r[o], l[o] = pcm[i], pcm[i+1]
	}
	audio := (*BufOutAudioPool.Get().(*[]int16))[:size]
	audio[0], audio[1] = r[0], l[0]
	for i, x := 1, 0; i < hs; i++ {
		if r[i] == 0 {
			r[i] = r[i-1]
		}
		if l[i] == 0 {
			l[i] = l[i-1]
		}
		x = i << 1
		audio[x], audio[x+1] = r[i], l[i]
		r[i-1] = 0
		l[i-1] = 0
	}
	r[hs-1] = 0
	l[hs-1] = 0
	r[0] = 0
	l[0] = 0
	bufHalfPool.Put(&r)
	bufHalfPool.Put(&l)
	return audio
}
