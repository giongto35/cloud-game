package recorder

import (
	"encoding/binary"
	"log"
)

type wavStream struct {
	Stream

	buf       chan Audio
	frequency int
	wav       *fileStream
}

const (
	audioFile         = "audio.wav"
	audioFileRIFFSize = 44
)

func NewWavAudioStream(dir string, frequency int) (*wavStream, error) {
	wav, err := newFileStream(dir, audioFile)
	if err != nil {
		return nil, err
	}
	// add pad for RIFF
	if err = wav.Write(make([]byte, audioFileRIFFSize)); err != nil {
		return nil, err
	}
	return &wavStream{
		frequency: frequency,
		wav:       wav,
		buf:       make(chan Audio, 1),
	}, nil
}

func (w *wavStream) Start() {
	go func() {
		for audio := range w.buf {
			if err := w.Save(*audio.Samples); err != nil {
				log.Printf("wav write err: %v", err)
			}
		}
	}()
}

func (w *wavStream) Stop() (err error) {
	close(w.buf)
	err = w.wav.Flush()
	fsize, er := w.wav.Size()
	if er != nil {
		err = er
	}
	if fsize > 0 {
		// write actual RIFF header
		err = w.wav.WriteAtStart(rIFFWavHeader(uint32(fsize), w.frequency))
		err = w.wav.Flush()
	}
	err = w.wav.Close()
	return
}

func (w *wavStream) Pause() {}

func (w *wavStream) Save(pcm []int16) error {
	pcmSize := len(pcm) * 2
	bs := make([]byte, pcmSize)
	// int & 0xFF + (int >> 8) & 0xFF
	for i, ln := 0, len(pcm); i < ln; i++ {
		binary.LittleEndian.PutUint16(bs[i*2:i*2+2], uint16(pcm[i]))
	}
	return w.wav.Write(bs)
}

// rIFFWavHeader creates RIFF WAV header.
// See: http://soundfile.sapp.org/doc/WaveFormat
func rIFFWavHeader(fSize uint32, fq int) []byte {
	const (
		bits  = 16
		ch    = 2
		chunk = 36
	)
	aSize := fSize - audioFileRIFFSize
	bitrate := uint32(fq*ch*bits) >> 3
	size := aSize + chunk
	header := [audioFileRIFFSize]byte{
		'R', 'I', 'F', 'F',
		byte(size & 0xff), byte((size >> 8) & 0xff), byte((size >> 16) & 0xff), byte((size >> 24) & 0xff),
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		byte(bits), 0, 0, 0, 1, 0,
		byte(ch), 0,
		byte(fq & 0xff), byte((fq >> 8) & 0xff), byte((fq >> 16) & 0xff), byte((fq >> 24) & 0xff),
		byte(bitrate & 0xff), byte((bitrate >> 8) & 0xff), byte((bitrate >> 16) & 0xff), byte((bitrate >> 24) & 0xff),
		byte(ch * bits >> 3),
		0, 16, 0,
		'd', 'a', 't', 'a',
		byte(aSize & 0xff), byte((aSize >> 8) & 0xff), byte((aSize >> 16) & 0xff), byte((aSize >> 24) & 0xff),
	}
	return header[:]
}
