package opus

import (
	"bytes"
	"encoding/binary"
	"github.com/zaf/resample"
	"log"
)

type Resampler interface {
	Init(in, out int) error
	Resample(pcm []int16, size int) []int16
	Close() error
}

type (
	Giongto35LinearResampler struct{}
	SoxResampler             struct {
		backend resample.Resampler
		buffer  *bytes.Buffer
	}
)

// resampleFn does a simple linear interpolation of audio samples.
func (Giongto35LinearResampler) Resample(pcm []int16, size int) []int16 {
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

func (Giongto35LinearResampler) Init(_, _ int) (err error) { return }

func (Giongto35LinearResampler) Close() (err error) { return }

func (r *SoxResampler) Init(in, out int) error {
	r.buffer = bytes.NewBuffer([]byte{})
	res, err := resample.New(
		r.buffer,
		float64(in),
		float64(out),
		2,
		resample.I16,
		resample.VeryHighQ,
	)
	if err != nil {
		return err
	}
	r.backend = *res
	return nil
}

func (r *SoxResampler) Resample(pcm []int16, size int) []int16 {
	//defer r.buffer.Reset()
	n, _ := r.backend.Write(toBytes(pcm))
	log.Printf("pcm: %v, out: %v, n: %v, size: %v", len(pcm), r.buffer.Len(), n, size)
	dat := r.buffer.Bytes()
	return toInt16(dat)
}

func (r *SoxResampler) Close() error {
	return r.backend.Close()
}

func toBytes(pcm []int16) (bytes []byte) {
	bytes = make([]uint8, len(pcm)*2)
	for i, k := 0, 0; i < len(pcm); i++ {
		binary.LittleEndian.PutUint16(bytes[k:k+2], uint16(pcm[i]))
		k += 2
	}
	return
}

func toInt16(bytes []byte) (pcm []int16) {
	pcm = make([]int16, len(bytes)/2)
	for i, k := 0, 0; i < len(pcm); i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(bytes[k : k+2]))
		k += 2
	}
	return
}
