package recorder

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type ffmpegStream struct {
	VideoStream

	demux *fileStream

	buf       chan Video
	dir       string
	pnge      *png.Encoder
	sequence  uint32
	startTime time.Time
	wg        *sync.WaitGroup
}

const (
	demuxFile = "input.txt"
	videoFile = "f%v.png"
)

type pool struct{ sync.Pool }

func newPool() *pool {
	return &pool{sync.Pool{New: func() interface{} { return &png.EncoderBuffer{} }}}
}
func (p *pool) Get() *png.EncoderBuffer  { return p.Pool.Get().(*png.EncoderBuffer) }
func (p *pool) Put(b *png.EncoderBuffer) { p.Pool.Put(b) }

func NewFfmpegStream(dir, game string, frequency int, compress int) (*ffmpegStream, error) {
	demux, err := newFileStream(dir, demuxFile)
	if err != nil {
		return nil, err
	}

	_, err = demux.WriteString(fmt.Sprintf("ffconcat version 1.0\n# d: %v, g: %v, f: %vhz\n\n",
		time.Now().Format("20060102"), game, frequency))

	return &ffmpegStream{
		buf:   make(chan Video, 1),
		dir:   dir,
		demux: demux,
		pnge: &png.Encoder{
			CompressionLevel: png.CompressionLevel(compress),
			BufferPool:       newPool(),
		},
		wg: &sync.WaitGroup{},
	}, nil
}

func (f *ffmpegStream) Start() {
	f.startTime = time.Now()
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

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	point := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func CloneToRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func (f *ffmpegStream) Save(img image.Image, dur time.Duration) error {
	fileName := fmt.Sprintf(videoFile, f.nextSeq())
	f.wg.Add(1)
	go f.saveImage(fileName, img)
	// ffmpeg concat demuxer, see: https://ffmpeg.org/ffmpeg-formats.html#concat
	inf := fmt.Sprintf("file %v\nduration %v\n", fileName, dur.Seconds())
	if _, err := f.demux.WriteString(inf); err != nil {
		return err
	}
	return nil
}

func (f *ffmpegStream) saveImage(fileName string, img image.Image) {
	defer f.wg.Done()

	// copy the image
	//imgg := CloneToRGBA(*img)
	var buf bytes.Buffer
	x, y := (img).Bounds().Dx(), (img).Bounds().Dy()
	buf.Grow(x * y * 4)

	//now := time.Now()
	//timeDiff := now.Sub(f.startTime)
	//
	//time_ := fmt.Sprintf("%s", timeDiff)
	//log.Printf(time_)
	//time_ := fmt.Sprintf("%0f:%0f:%0f.%000d", timeDiff.Hours(), timeDiff.Minutes(), timeDiff.Seconds(), timeDiff.Milliseconds())
	//addLabel(imgg, 100, y-21, time_)
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
