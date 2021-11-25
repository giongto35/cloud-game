package recorder

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestName(t *testing.T) {
	dir, err := ioutil.TempDir("", "rec_test_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config := shared.Recording{Enabled: true, Name: "", Folder: dir}
	recorder := NewRecording(fmt.Sprintf("test_game_%v", rand.Int()), 100, config)
	recorder.Set(true, "test_user")
	recorder.Start()

	iterations := 222

	var imgWg, audioWg sync.WaitGroup
	imgWg.Add(iterations)
	audioWg.Add(iterations)
	img := generateImage(100, 100)

	for i := 0; i < 222; i++ {
		go func() {
			recorder.WriteVideo(Video{Image: img, Duration: 16 * time.Millisecond})
			imgWg.Done()
		}()
		go func() {
			recorder.WriteAudio(Audio{&[]int16{0, 0, 0, 0, 0, 1, 11, 11, 11, 1}})
			audioWg.Done()
		}()
	}

	imgWg.Wait()
	audioWg.Wait()
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkNewRecording100x100(b *testing.B) {
	benchmarkRecorder(100, 100, 0, b)
}

func BenchmarkNewRecording320x240_compressed(b *testing.B) {
	benchmarkRecorder(320, 240, 0, b)
}
func BenchmarkNewRecording320x240_nocompression(b *testing.B) {
	benchmarkRecorder(320, 240, -1, b)
}

func benchmarkRecorder(w, h int, comp int, b *testing.B) {
	b.StopTimer()

	dir, err := ioutil.TempDir("", "rec_bench_")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			b.Fatal(err)
		}
	}()

	image1 := generateImage(w, h)
	image2 := generateImage(w, h)

	var bytes int64 = 0

	var imgWg, audioWg sync.WaitGroup
	imgWg.Add(b.N)
	audioWg.Add(b.N)

	b.StartTimer()

	config := shared.Recording{Enabled: true, Name: "", Folder: dir, CompressLevel: comp}
	recorder := NewRecording(fmt.Sprintf("test_game_%v", rand.Int()), 100, config)
	recorder.Set(true, "test_user")
	recorder.Start()
	samples := []int16{0, 0, 0, 0, 0, 1, 11, 11, 11, 1}

	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}
		go func(img image.Image) {
			recorder.WriteVideo(Video{Image: img, Duration: 16 * time.Millisecond})
			atomic.AddInt64(&bytes, int64(len(img.(*image.RGBA).Pix)))
			imgWg.Done()
		}(im)
		go func() {
			recorder.WriteAudio(Audio{&samples})
			atomic.AddInt64(&bytes, int64(len(samples)*2))
			audioWg.Done()
		}()
	}
	imgWg.Wait()
	audioWg.Wait()
	b.SetBytes(bytes / int64(b.N))
	if err := recorder.Stop(); err != nil {
		b.Fatal(err)
	}
}

func generateImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, randomColor())
		}
	}
	return img
}

var rnd = rand.New(rand.NewSource(time.Now().Unix()))

func randomColor() color.RGBA {
	return color.RGBA{
		R: uint8(rnd.Intn(256)),
		G: uint8(rnd.Intn(256)),
		B: uint8(rnd.Intn(256)),
		A: 255,
	}
}
