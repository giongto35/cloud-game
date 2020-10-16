package room

import (
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/faiface/mainthread"
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	storage "github.com/giongto35/cloud-game/v2/pkg/worker/cloud-storage"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type roomMock struct {
	Room
}

type roomMockConfig struct {
	roomName  string
	gamesPath string
	game      games.GameMetadata
	codec     string
}

// Restricts a re-config call
// to only one invocation.
var configOnce sync.Once

// Store absolute path to test games
var whereIsGames = getAppPath() + "assets/games/"

func TestRoom(t *testing.T) {
	tests := []struct {
		roomName string
		game     games.GameMetadata
		codec    string
		frames   int
	}{
		{
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: "Super Mario Bros.nes",
			},
			codec:  config.CODEC_VP8,
			frames: 5,
		},
	}

	for _, test := range tests {
		room := getRoomMock(roomMockConfig{
			roomName:  test.roomName,
			gamesPath: whereIsGames,
			game:      test.game,
			codec:     test.codec,
		})
		t.Logf("The game [%v] has been loaded\n", test.game.Name)
		waitUntilRenderedFrames(test.frames, room.encoder.GetOutputChan())
		room.Close()
	}
}

func TestRoomWithGL(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		codec  string
		frames int
	}{
		{
			game: games.GameMetadata{
				Name: "Sample Demo by Florian (PD)",
				Type: "n64",
				Path: "Sample Demo by Florian (PD).z64",
			},
			codec:  config.CODEC_VP8,
			frames: 50,
		},
	}

	run := func() {
		for _, test := range tests {
			room := getRoomMock(roomMockConfig{
				gamesPath: whereIsGames,
				game:      test.game,
				codec:     test.codec,
			})
			t.Logf("The game [%v] has been loaded\n", test.game.Name)
			waitUntilRenderedFrames(test.frames, room.encoder.GetOutputChan())
			room.Close()
		}
	}

	mainthread.Run(run)
}

func TestAllEmulatorRooms(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		codec  string
		frames int
	}{
		{
			game: games.GameMetadata{
				Name: "Sushi The Cat",
				Type: "gba",
				Path: "Sushi The Cat.gba",
			},
			codec:  config.CODEC_VP8,
			frames: 100,
		},
		{
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: "Super Mario Bros.nes",
			},
			codec:  config.CODEC_VP8,
			frames: 200,
		},
		{
			game: games.GameMetadata{
				Name: "Sample Demo by Florian (PD)",
				Type: "n64",
				Path: "Sample Demo by Florian (PD).z64",
			},
			codec:  config.CODEC_VP8,
			frames: 50,
		},
	}
	crc32q := crc32.MakeTable(0xD5828281)

	run := func() {
		for _, test := range tests {
			room := getRoomMock(roomMockConfig{
				gamesPath: whereIsGames,
				game:      test.game,
				codec:     test.codec,
			})
			t.Logf("The game [%v] has been loaded\n", test.game.Name)
			waitUntilRenderedFrames(test.frames, room.encoder.GetOutputChan())

			img := room.director.GetViewport().(*image.RGBA)

			hash := fmt.Sprintf("%08x", crc32.Checksum(img.Pix, crc32q))
			addLabel(img, 10, 20,
				fmt.Sprintf("%v-%v-0x%v [%v]", runtime.GOOS, test.game.Type, hash, test.frames))
			dumpCanvas(img, fmt.Sprintf("D:/%v-%v.png", runtime.GOOS, test.game.Type))

			room.Close()
			// hack: wait room destruction
			time.Sleep(3 * time.Second)
		}
	}

	mainthread.Run(run)
}

func addLabel(img *image.RGBA, x, y int, label string) {
	(&font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)},
	}).DrawString(label)
}

// enforce image.RGBA to remove alpha channel when encoding PNGs
type opaqueRGBA struct {
	*image.RGBA
}

func (*opaqueRGBA) Opaque() bool {
	return true
}

func dumpCanvas(frame *image.RGBA, name string) {
	f, err := os.Create(name)
	if err != nil {
		// Handle error
	}
	defer f.Close()

	err = png.Encode(f, &opaqueRGBA{frame})
	if err != nil {
		log.Printf("Errr: %v", err)
		// Handle error
	}
}

// getRoomMock returns mocked Room struct.
func getRoomMock(cfg roomMockConfig) roomMock {
	configOnce.Do(fixEmulatorPaths)
	cfg.game.Path = cfg.gamesPath + cfg.game.Path
	room := NewRoom(cfg.roomName, cfg.game, cfg.codec, storage.NewInitClient(), worker.NewDefaultConfig())

	// loop-wait the room initialization
	var waitCounter sync.WaitGroup
	waitCounter.Add(1)
	wasted := 0
	go func() {
		sleepDeltaMs := 10
		for room.director == nil || room.encoder == nil {
			time.Sleep(time.Duration(sleepDeltaMs) * time.Millisecond)
			wasted++
		}
		waitCounter.Done()
	}()
	waitCounter.Wait()

	return roomMock{*room}
}

// fixEmulatorPaths makes absolute game paths in global GameList.
func fixEmulatorPaths() {
	appPath := getAppPath()

	for k, conf := range config.EmulatorConfig {
		conf.Path = appPath + conf.Path
		if len(conf.Config) > 0 {
			conf.Config = appPath + conf.Config
		}
		config.EmulatorConfig[k] = conf
	}
}

// getAppPath returns absolute path to the assets directory.
func getAppPath() string {
	appName := "cloud-game"
	_, b, _, _ := runtime.Caller(0)
	return filepath.Dir(strings.SplitAfter(b, appName)[0]) + "/" + appName + "/"
}

func waitUntilRenderedFrames(n int, ch chan encoder.OutFrame) {
	var waitCounter sync.WaitGroup
	waitCounter.Add(n)

	done := false
	go func() {
		for range ch {
			if done {
				break
			}
			waitCounter.Done()
		}
	}()

	waitCounter.Wait()
	done = true
}

// benchmarkRoom measures app performance for n emulation frames.
// Measure period: the room initialization, n emulated and encoded frames, the room shutdown.
func benchmarkRoom(rom games.GameMetadata, codec string, frames int, suppressOutput bool, b *testing.B) {
	if suppressOutput {
		log.SetOutput(ioutil.Discard)
		os.Stdout, _ = os.Open(os.DevNull)
	}

	for i := 0; i < b.N; i++ {
		room := getRoomMock(roomMockConfig{
			gamesPath: whereIsGames,
			game:      rom,
			codec:     codec,
		})
		waitUntilRenderedFrames(frames, room.encoder.GetOutputChan())
		room.Close()
	}
}

// Measures emulation performance of various
// emulators and encoding options.
func BenchmarkRoom(b *testing.B) {
	benches := []struct {
		system string
		game   games.GameMetadata
		codecs []string
		frames int
	}{
		// warm up
		{
			system: "gba",
			game: games.GameMetadata{
				Name: "Sushi The Cat",
				Type: "gba",
				Path: "Sushi The Cat.gba",
			},
			codecs: []string{"vp8"},
			frames: 50,
		},
		{
			system: "gba",
			game: games.GameMetadata{
				Name: "Sushi The Cat",
				Type: "gba",
				Path: "Sushi The Cat.gba",
			},
			codecs: []string{"vp8", "x264"},
			frames: 100,
		},
		{
			system: "nes",
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: "Super Mario Bros.nes",
			},
			codecs: []string{"vp8", "x264"},
			frames: 100,
		},
	}

	for _, bench := range benches {
		for _, codec := range bench.codecs {
			b.Run(fmt.Sprintf("%s-%s-%d", bench.system, codec, bench.frames), func(b *testing.B) {
				benchmarkRoom(bench.game, codec, bench.frames, true, b)
			})
			// hack: wait room destruction
			time.Sleep(5 * time.Second)
		}
	}
}
