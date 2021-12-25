package media

// ResampleStretch does a simple stretching of audio samples.
func ResampleStretch(pcm []int16, size int) []int16 {
	r, l, audio := make([]int16, size/2), make([]int16, size/2), make([]int16, size)
	// ratio is basically the destination sample rate
	// divided by the origin sample rate (i.e. 48000/44100)
	ratio := float32(size) / float32(len(pcm))
	for i, n := 0, len(pcm)-1; i < n; i += 2 {
		idx := int(float32(i/2) * ratio)
		r[idx], l[idx] = pcm[i], pcm[i+1]
	}
	for i, n := 1, len(r); i < n; i++ {
		if r[i] == 0 {
			r[i] = r[i-1]
		}
		if l[i] == 0 {
			l[i] = l[i-1]
		}
	}
	for i := 0; i < size-1; i += 2 {
		audio[i], audio[i+1] = r[i/2], l[i/2]
	}
	return audio
}
