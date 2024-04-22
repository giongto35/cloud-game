package recorder

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand/v2"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

func TestName(t *testing.T) {
	dir, err := os.MkdirTemp("", "rec_test_")
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
		logger.Default(),
		Options{
			Dir:       dir,
			Fps:       60,
			Frequency: 10,
			Game:      fmt.Sprintf("test_game_%v", rand.Int()),
			Name:      "test",
			Zip:       false,
		})
	recorder.Set(true, "test_user")

	iterations := 222

	var imgWg, audioWg sync.WaitGroup
	imgWg.Add(iterations)
	audioWg.Add(iterations)
	frame := genFrame(100, 100)

	for i := 0; i < 222; i++ {
		go func() {
			recorder.WriteVideo(Video{Frame: frame, Duration: 16 * time.Millisecond})
			imgWg.Done()
		}()
		go func() {
			recorder.WriteAudio(Audio{[]int16{0, 0, 0, 0, 0, 1, 11, 11, 11, 1}, 1})
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
	benchmarkRecorder(100, 100, b)
}

func BenchmarkNewRecording320x240(b *testing.B) {
	benchmarkRecorder(320, 240, b)
}

func benchmarkRecorder(w, h int, b *testing.B) {
	b.StopTimer()

	dir, err := os.MkdirTemp("", "rec_bench_")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			b.Fatal(err)
		}
	}()

	frame1 := genFrame(w, h)
	frame2 := genFrame(w, h)

	var bytes int64 = 0

	var ticks sync.WaitGroup
	ticks.Add(b.N * 2)

	b.StartTimer()

	recorder := NewRecording(
		Meta{UserName: "test"},
		logger.Default(),
		Options{
			Dir:       dir,
			Fps:       60,
			Frequency: 10,
			Game:      fmt.Sprintf("test_game_%v", rand.Int()),
			Name:      "",
			Zip:       false,
		})
	recorder.Set(true, "test_user")
	samples := []int16{0, 0, 0, 0, 0, 1, 11, 11, 11, 1}

	for i := 0; i < b.N; i++ {
		f := frame1
		if i%2 == 0 {
			f = frame2
		}
		go func() {
			recorder.WriteVideo(Video{Frame: f, Duration: 16 * time.Millisecond})
			atomic.AddInt64(&bytes, int64(len(f.Data)))
			ticks.Done()
		}()
		go func() {
			recorder.WriteAudio(Audio{samples, 1})
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

func genFrame(w, h int) Frame {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, randomColor())
		}
	}
	return Frame{
		Data:   img.Pix,
		Stride: img.Stride,
		W:      img.Bounds().Dx(),
		H:      img.Bounds().Dy(),
	}
}

func randomColor() color.RGBA {
	return color.RGBA{
		R: uint8(rand.IntN(256)),
		G: uint8(rand.IntN(256)),
		B: uint8(rand.IntN(256)),
		A: 255,
	}
}
