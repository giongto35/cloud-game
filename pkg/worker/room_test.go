package worker

import (
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/worker/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/worker/storage"
	"github.com/giongto35/cloud-game/v2/pkg/worker/thread"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var (
	renderFrames  bool
	outputPath    string
	autoGlContext bool
)

type roomMock struct {
	*Room
	startEmulator bool
}

type roomMockConfig struct {
	roomName          string
	gamesPath         string
	game              games.GameMetadata
	vCodec            encoder.VideoCodec
	autoGlContext     bool
	dontStartEmulator bool
}

// Store absolute path to test games
var whereIsGames = getRootPath() + "assets/games/"
var whereIsConfigs = getRootPath() + "configs/"
var testTempDir = filepath.Join(os.TempDir(), "cloud-game-core-tests")

func init() {
	runtime.LockOSThread()
}

func TestMain(m *testing.M) {
	flag.BoolVar(&renderFrames, "renderFrames", false, "Render frames for eye testing purposes")
	flag.StringVar(&outputPath, "outputPath", "./", "Output path for generated files")
	flag.BoolVar(&autoGlContext, "autoGlContext", false, "Set auto GL context choose for headless machines")

	thread.Wrap(func() { os.Exit(m.Run()) })
}

func TestRoom(t *testing.T) {
	tests := []struct {
		roomName string
		game     games.GameMetadata
		vCodec   encoder.VideoCodec
		frames   int
	}{
		{
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: "Super Mario Bros.nes",
			},
			vCodec: encoder.VP8,
			frames: 5,
		},
	}

	for _, test := range tests {
		room := getRoomMock(roomMockConfig{
			roomName:  test.roomName,
			gamesPath: whereIsGames,
			game:      test.game,
			vCodec:    test.vCodec,
		})

		t.Logf("The game [%v] has been loaded", test.game.Name)
		waitNFrames(test.frames, room)
		room.Close()
	}
	// hack: wait room destruction
	time.Sleep(2 * time.Second)
}

func TestRoomWithGL(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		vCodec encoder.VideoCodec
		frames int
	}{
		{
			game: games.GameMetadata{
				Name: "Sample Demo by Florian (PD)",
				Type: "n64",
				Path: "Sample Demo by Florian (PD).z64",
			},
			vCodec: encoder.VP8,
			frames: 50,
		},
	}

	run := func() {
		for _, test := range tests {
			room := getRoomMock(roomMockConfig{
				gamesPath: whereIsGames,
				game:      test.game,
				vCodec:    test.vCodec,
			})
			t.Logf("The game [%v] has been loaded", test.game.Name)
			waitNFrames(test.frames, room)
			room.Close()
		}
		// hack: wait room destruction
		time.Sleep(2 * time.Second)
	}

	thread.Main(run)
}

func TestAllEmulatorRooms(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		frames int
	}{
		{
			game:   games.GameMetadata{Name: "Sushi", Type: "gba", Path: "Sushi The Cat.gba"},
			frames: 150,
		},
		{
			game:   games.GameMetadata{Name: "Mario", Type: "nes", Path: "Super Mario Bros.nes"},
			frames: 50,
		},
		{
			game:   games.GameMetadata{Name: "Florian Demo", Type: "n64", Path: "Sample Demo by Florian (PD).z64"},
			frames: 50,
		},
	}

	crc32q := crc32.MakeTable(0xD5828281)

	for _, test := range tests {
		room := getRoomMock(roomMockConfig{
			gamesPath:         whereIsGames,
			game:              test.game,
			vCodec:            encoder.VP8,
			autoGlContext:     autoGlContext,
			dontStartEmulator: true,
		})
		t.Logf("The game [%v] has been loaded", test.game.Name)
		frame := waitNFrames(test.frames, room)

		if renderFrames {
			tag := fmt.Sprintf("%v-%v-0x%08x", runtime.GOOS, test.game.Type, crc32.Checksum(frame.Data.Pix, crc32q))
			dumpCanvas(frame.Data, tag, fmt.Sprintf("%v [%v]", tag, test.frames), outputPath)
		}

		room.Close()
		// hack: wait room destruction
		time.Sleep(1 * time.Second)
	}
}

type OpaqueHack struct{ *image.RGBA }

func (o *OpaqueHack) Opaque() bool { return true }

func dumpCanvas(f *image.RGBA, name string, caption string, path string) {
	frame := OpaqueHack{f}

	// slap 'em caption
	if len(caption) > 0 {
		draw.Draw(&frame, image.Rect(8, 8, 8+len(caption)*7+3, 24), &image.Uniform{C: color.RGBA{}}, image.Point{}, draw.Src)
		(&font.Drawer{
			Dst:  &frame,
			Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			Face: basicfont.Face7x13,
			Dot:  fixed.Point26_6{X: fixed.Int26_6(10 * 64), Y: fixed.Int26_6(20 * 64)},
		}).DrawString(caption)
	}

	var outPath string
	if len(path) > 0 {
		outPath = path
	} else {
		outPath = testTempDir
	}

	// really like Go's error handling
	if err := os.MkdirAll(outPath, 0770); err != nil {
		log.Printf("Couldn't create target dir for the output images, %v", err)
		return
	}

	if f, err := os.Create(filepath.Join(outPath, name+".png")); err == nil {
		if err = png.Encode(f, &frame); err != nil {
			log.Printf("Couldn't encode the image, %v", err)
		}
		_ = f.Close()
	} else {
		log.Printf("Couldn't create the image, %v", err)
	}
}

// getRoomMock returns mocked Room struct.
func getRoomMock(cfg roomMockConfig) roomMock {
	cfg.game.Path = cfg.gamesPath + cfg.game.Path

	var conf worker.Config
	if err := config.LoadConfig(&conf, whereIsConfigs); err != nil {
		panic(err)
	}
	fixEmulators(&conf, cfg.autoGlContext)
	l := logger.NewConsole(conf.Worker.Debug, "w", true)

	// sync cores
	coreManager := remotehttp.NewRemoteHttpManager(conf.Emulator.Libretro, l)
	if err := coreManager.Sync(); err != nil {
		log.Printf("error: cores sync has failed, %v", err)
	}
	conf.Encoder.Video.Codec = string(cfg.vCodec)

	cloudStore, _ := storage.NewNoopCloudStorage()

	room := NewRoom(cfg.roomName, cfg.game, cloudStore, nil, false, "", conf, l)

	if !cfg.dontStartEmulator {
		room.StartEmulator()
	}

	// loop-wait the room initialization
	var init sync.WaitGroup
	init.Add(1)
	wasted := 0
	go func() {
		sleepDeltaMs := 10
		for room.emulator == nil {
			time.Sleep(time.Duration(sleepDeltaMs) * time.Millisecond)
			wasted++
			if wasted > 1000 {
				break
			}
		}
		init.Done()
	}()
	init.Wait()
	return roomMock{Room: room, startEmulator: !cfg.dontStartEmulator}
}

// fixEmulators makes absolute game paths in global GameList and passes GL context config.
// hack: emulator paths should be absolute and visible to the tests.
func fixEmulators(config *worker.Config, autoGlContext bool) {
	rootPath := getRootPath()

	config.Emulator.Libretro.Cores.Paths.Libs =
		filepath.FromSlash(rootPath + config.Emulator.Libretro.Cores.Paths.Libs)
	config.Emulator.Libretro.Cores.Paths.Configs =
		filepath.FromSlash(rootPath + config.Emulator.Libretro.Cores.Paths.Configs)
	config.Emulator.LocalPath = filepath.FromSlash(filepath.Join(rootPath, "tests", config.Emulator.LocalPath))
	config.Emulator.Storage = filepath.FromSlash(filepath.Join(rootPath, "tests", "storage"))

	for k, conf := range config.Emulator.Libretro.Cores.List {
		if conf.IsGlAllowed && autoGlContext {
			conf.AutoGlContext = true
		}
		config.Emulator.Libretro.Cores.List[k] = conf
	}
}

// getRootPath returns absolute path to the assets.
func getRootPath() string {
	p, _ := filepath.Abs("../../")
	return p + string(filepath.Separator)
}

func waitNFrames(n int, room roomMock) *emulator.GameFrame {
	var i = int32(n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	var frame *emulator.GameFrame
	room.emulator.SetVideo(func(video *emulator.GameFrame) {
		if atomic.AddInt32(&i, -1) >= 0 {
			frame = video
			wg.Done()
		}
	})
	if !room.startEmulator {
		room.StartEmulator()
	}
	wg.Wait()
	return frame
}

// benchmarkRoom measures app performance for n emulation frames.
// Measure period: the room initialization, n emulated and encoded frames, the room shutdown.
func benchmarkRoom(rom games.GameMetadata, codec encoder.VideoCodec, frames int, suppressOutput bool, b *testing.B) {
	if suppressOutput {
		log.SetOutput(io.Discard)
		os.Stdout, _ = os.Open(os.DevNull)
	}

	for i := 0; i < b.N; i++ {
		room := getRoomMock(roomMockConfig{
			gamesPath: whereIsGames,
			game:      rom,
			vCodec:    codec,
		})
		waitNFrames(frames, room)
		room.Close()
	}
}

// Measures emulation performance of various
// emulators and encoding options.
func BenchmarkRoom(b *testing.B) {
	benches := []struct {
		system string
		game   games.GameMetadata
		codecs []encoder.VideoCodec
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
			codecs: []encoder.VideoCodec{encoder.VP8},
			frames: 50,
		},
		{
			system: "gba",
			game: games.GameMetadata{
				Name: "Sushi The Cat",
				Type: "gba",
				Path: "Sushi The Cat.gba",
			},
			codecs: []encoder.VideoCodec{encoder.VP8, encoder.H264},
			frames: 100,
		},
		{
			system: "nes",
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: "Super Mario Bros.nes",
			},
			codecs: []encoder.VideoCodec{encoder.VP8, encoder.H264},
			frames: 100,
		},
	}

	for _, bench := range benches {
		for _, cod := range bench.codecs {
			b.Run(fmt.Sprintf("%s-%v-%d", bench.system, cod, bench.frames), func(b *testing.B) {
				benchmarkRoom(bench.game, cod, bench.frames, true, b)
			})
			// hack: wait room destruction
			time.Sleep(5 * time.Second)
		}
	}
}
