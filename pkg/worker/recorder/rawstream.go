package recorder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

type rawStream struct {
	dir string
	id  uint32
	wg  sync.WaitGroup
}

const videoFile = "f%07d__%dx%d__%d.raw"

func newRawStream(dir string) (*rawStream, error) {
	return &rawStream{dir: dir}, nil
}

func (p *rawStream) Close() error {
	atomic.StoreUint32(&p.id, 0)
	p.wg.Wait()
	return nil
}

func (p *rawStream) Write(data Video) {
	i := atomic.AddUint32(&p.id, 1)
	fileName := fmt.Sprintf(videoFile, i, data.Frame.W, data.Frame.H, data.Frame.Stride)
	p.wg.Add(1)
	go p.saveFrame(fileName, data.Frame)
}

func (p *rawStream) saveFrame(fileName string, frame Frame) {
	file, err := os.Create(filepath.Join(p.dir, fileName))
	if err != nil {
		log.Printf("c err: %v", err)
	}
	if _, err = file.Write(frame.Data); err != nil {
		log.Printf("f err: %v", err)
	}

	if err = file.Close(); err != nil {
		log.Printf("fc err: %v", err)
	}
	p.wg.Done()
}

func ExtractFileInfo(name string) (w, h, st string) {
	s1 := strings.Split(name, "__")
	if len(s1) > 1 {
		s12 := strings.Split(s1[1], "x")
		if len(s12) > 1 {
			w, h = s12[0], s12[1]
		}
		s21 := strings.TrimSuffix(s1[2], filepath.Ext(s1[2]))
		if s21 != "" {
			st = s21
		}
	}
	return
}
