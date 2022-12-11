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
)

type pngStream struct {
	videoStream

	dir string
	e   *png.Encoder
	id  uint32
	wg  sync.WaitGroup
}

const videoFile = "f%07d.png"

type pool struct{ sync.Pool }

func pngBuf() *pool                      { return &pool{sync.Pool{New: func() any { return &png.EncoderBuffer{} }}} }
func (p *pool) Get() *png.EncoderBuffer  { return p.Pool.Get().(*png.EncoderBuffer) }
func (p *pool) Put(b *png.EncoderBuffer) { p.Pool.Put(b) }

func newPngStream(dir string, opts Options) (*pngStream, error) {
	return &pngStream{
		dir: dir,
		e: &png.Encoder{
			CompressionLevel: png.CompressionLevel(opts.ImageCompressionLevel),
			BufferPool:       pngBuf(),
		},
	}, nil
}

func (p *pngStream) Close() error {
	atomic.StoreUint32(&p.id, 0)
	p.wg.Wait()
	return nil
}

func (p *pngStream) Write(data Video) {
	fileName := fmt.Sprintf(videoFile, atomic.AddUint32(&p.id, 1))
	p.wg.Add(1)
	go p.saveImage(fileName, data.Image)
}

func (p *pngStream) saveImage(fileName string, img image.Image) {
	var buf bytes.Buffer
	x, y := (img).Bounds().Dx(), (img).Bounds().Dy()
	buf.Grow(x * y * 4)

	if err := p.e.Encode(&buf, img); err != nil {
		log.Printf("p err: %v", err)
	} else {
		file, err := os.Create(filepath.Join(p.dir, fileName))
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
	p.wg.Done()
}
