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
)

func TestName(t *testing.T) {
	dir, err := ioutil.TempDir("", "rec_test_")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	recorder := NewRecording(
		Meta{UserName: "test"},
		Options{
			Dir:                   dir,
			Fps:                   60,
			Frequency:             10,
			Game:                  fmt.Sprintf("test_game_%v", rand.Int()),
			ImageCompressionLevel: 0,
			Name:                  "test",
			Zip:                   true,
		})
	recorder.Set(true, "test_user")

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

	var ticks sync.WaitGroup
	ticks.Add(b.N * 2)

	b.StartTimer()

	recorder := NewRecording(
		Meta{UserName: "test"},
		Options{
			Dir:                   dir,
			Fps:                   60,
			Frequency:             10,
			Game:                  fmt.Sprintf("test_game_%v", rand.Int()),
			ImageCompressionLevel: comp,
			Name:                  "",
			Zip:                   false,
		})
	recorder.Set(true, "test_user")
	samples := []int16{0, 0, 0, 0, 0, 1, 11, 11, 11, 1}

	for i := 0; i < b.N; i++ {
		im := image1
		if i%2 == 0 {
			im = image2
		}
		go func() {
			recorder.WriteVideo(Video{Image: im, Duration: 16 * time.Millisecond})
			atomic.AddInt64(&bytes, int64(len(im.(*image.RGBA).Pix)))
			ticks.Done()
		}()
		go func() {
			recorder.WriteAudio(Audio{&samples})
			atomic.AddInt64(&bytes, int64(len(samples)*2))
			ticks.Done()
		}()
	}
	ticks.Wait()
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
