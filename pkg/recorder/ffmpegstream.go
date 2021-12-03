package recorder

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
)

type ffmpegStream struct {
	VideoStream

	demux *file

	buf      chan Video
	dir      string
	pnge     *png.Encoder
	sequence uint32
	fps      float64
	wg       sync.WaitGroup
}

const (
	demuxFile = "input.txt"
	videoFile = "f%v.png"
)

type pool struct{ sync.Pool }

func pngBuf() *pool                      { return &pool{sync.Pool{New: func() interface{} { return &png.EncoderBuffer{} }}} }
func (p *pool) Get() *png.EncoderBuffer  { return p.Pool.Get().(*png.EncoderBuffer) }
func (p *pool) Put(b *png.EncoderBuffer) { p.Pool.Put(b) }

func NewFfmpegStream(dir string, opts Options) (*ffmpegStream, error) {
	demux, err := newFile(dir, demuxFile)
	if err != nil {
		return nil, err
	}

	_, err = demux.WriteString(
		fmt.Sprintf("ffconcat version 1.0\n"+
			"# v: 1\n# date: %v\n# game: %v\n# fps: %v\n# freq (hz): %v\n\n",
			time.Now().Format("20060102"), opts.Game, opts.Fps, opts.Frequency))

	return &ffmpegStream{
		buf:   make(chan Video, 1),
		dir:   dir,
		demux: demux,
		fps:   opts.Fps,
		pnge: &png.Encoder{
			CompressionLevel: png.CompressionLevel(opts.ImageCompressionLevel),
			BufferPool:       pngBuf(),
		},
	}, nil
}

func (f *ffmpegStream) Start() {
	for frame := range f.buf {
		if err := f.Save(frame.Image, frame.Duration); err != nil {
			log.Printf("image write err: %v", err)
		}
	}
}

func (f *ffmpegStream) Stop() error {
	var result *multierror.Error
	close(f.buf)
	f.resetSeq()
	result = multierror.Append(result, f.demux.Flush())
	result = multierror.Append(result, f.demux.Close())
	f.wg.Wait()
	return result.ErrorOrNil()
}

func (f *ffmpegStream) Save(img image.Image, dur time.Duration) error {
	fileName := fmt.Sprintf(videoFile, f.nextSeq())
	f.wg.Add(1)
	go f.saveImage(fileName, img)
	// ffmpeg concat demuxer, see: https://ffmpeg.org/ffmpeg-formats.html#concat
	inf := fmt.Sprintf("file %v\nduration %v\n#delta %v\n", fileName, 1/f.fps, dur.Seconds())
	if _, err := f.demux.WriteString(inf); err != nil {
		return err
	}
	return nil
}

func (f *ffmpegStream) saveImage(fileName string, img image.Image) {
	defer f.wg.Done()

	var buf bytes.Buffer
	x, y := (img).Bounds().Dx(), (img).Bounds().Dy()
	buf.Grow(x * y * 4)

	if err := f.pnge.Encode(&buf, img); err != nil {
		log.Printf("p err: %v", err)
	} else {
		file, err := os.Create(filepath.Join(f.dir, fileName))
		if err != nil {
			log.Printf("c err: %v", err)
		}
		if _, err = file.Write(buf.Bytes()); err != nil {
			log.Printf("f err: %v", err)
		}
		if err = file.Close(); err != nil {
			log.Printf("fc err: %v", err)
		}
	}
}

func (f *ffmpegStream) nextSeq() uint32 { return atomic.AddUint32(&f.sequence, 1) }
func (f *ffmpegStream) resetSeq()       { atomic.StoreUint32(&f.sequence, 0) }

func (f *ffmpegStream) Write(data Video) { f.buf <- data }
