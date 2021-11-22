package room

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/img"
)

type Recording struct {
	sync.Mutex

	active bool
	User   string

	path string

	audio *os.File

	// audio
	abuf chan []int16

	// video
	vbuf chan *image.Image

	counter int32
}

func NewRecording(game string, conf shared.Recording) Recording {
	path := filepath.Join(filepath.Dir(conf.Folder), game)

	log.Printf("[recording] path is [%v]", path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.Mkdir(path, os.ModeDir)
		// TODO: handle error
	}

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filepath.Join(path, fmt.Sprintf("%v.pcm", game)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return Recording{
		audio: f,
		path:  path,
	}
}

func (r *Recording) Stop() {
	r.Lock()
	r.active = false
	r.Unlock()
	if r.audio != nil {
		r.audio.Close()
	}
	r.resetImageNum()
}

func (r *Recording) Set(active bool, user string) {
	r.Lock()
	r.active = active
	r.User = user
	r.Unlock()
}

// ffmpeg -framerate 60 -i ./bad_apple_2_5/image_%d.png -f s16le -ac 2 -ar 48K -i ./bad_apple_2_5/bad_apple_2_5.pcm -c:v
// libx264 -c:a mp3 out.mp4
func (r *Recording) SaveImage(image *image.RGBA, dur time.Duration) {
	f, _ := os.Create(filepath.Join(r.path, fmt.Sprintf("image_%v.png", r.nextImageNum())))
	_ = png.Encode(f, &img.ORGBA{RGBA: image})
}

func (r *Recording) SaveAudio(pcm []int16) {
	b := make([]byte, 2)
	for _, p := range pcm {
		binary.LittleEndian.PutUint16(b, uint16(p))
		if _, err := r.audio.Write(b); err != nil {
			log.Fatal(err)
		}
	}
}

func (r *Recording) nextImageNum() int32 { return atomic.AddInt32(&r.counter, 1) }
func (r *Recording) resetImageNum()      { atomic.AddInt32(&r.counter, 1) }
