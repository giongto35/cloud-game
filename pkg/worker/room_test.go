package worker

import (
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/config/worker"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator"
	image2 "github.com/giongto35/cloud-game/v3/pkg/worker/emulator/image"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
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

func (rm roomMock) Close() {
	rm.Room.Close()
	// hack: wait room destruction
	time.Sleep(3 * time.Second)
}

func (rm roomMock) CloseNowait() { rm.Room.Close() }

type roomMockConfig struct {
	roomName          string
	gamesPath         string
	game              games.GameMetadata
	vCodec            encoder.VideoCodec
	autoGlContext     bool
	dontStartEmulator bool
	noLog             bool
}

// Store absolute path to test games
var whereIsGames = getRootPath() + "assets/games/"
var whereIsConfigs = getRootPath() + "configs/"
var testTempDir = filepath.Join(os.TempDir(), "cloud-game-core-tests")

// games
var (
	mario = games.GameMetadata{Name: "Super Mario Bros", Type: "nes", Path: "Super Mario Bros.nes"}
	sushi = games.GameMetadata{Name: "Sushi The Cat", Type: "gba", Path: "Sushi The Cat.gba"}
	fd    = games.GameMetadata{Name: "Florian Demo", Type: "n64", Path: "Sample Demo by Florian (PD).z64"}
)

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
			game:   mario,
			vCodec: encoder.H264,
			frames: 300,
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
}

func TestRoomWithGL(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		vCodec encoder.VideoCodec
		frames int
	}{
		{game: fd, vCodec: encoder.VP8, frames: 50},
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
	}

	thread.Main(run)
}

func TestAllEmulatorRooms(t *testing.T) {
	tests := []struct {
		game   games.GameMetadata
		frames int
	}{
		{game: sushi, frames: 150},
		{game: mario, frames: 50},
		{game: fd, frames: 50},
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
	}
}

func dumpCanvas(frame *image2.Frame, name string, caption string, path string) {
	// slap 'em caption
	if len(caption) > 0 {
		draw.Draw(frame, image.Rect(8, 8, 8+len(caption)*7+3, 24), &image.Uniform{C: color.RGBA{}}, image.Point{}, draw.Src)
		(&font.Drawer{
			Dst:  frame,
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
		if err = png.Encode(f, frame); err != nil {
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
	l := logger.NewConsole(conf.Worker.Debug, "w", false)
	if cfg.noLog {
		logger.SetGlobalLevel(logger.Disabled)
	}

	// sync cores
	coreManager := remotehttp.NewRemoteHttpManager(conf.Emulator.Libretro, l)
	if err := coreManager.Sync(); err != nil {
		log.Printf("error: cores sync has failed, %v", err)
	}
	conf.Encoder.Video.Codec = string(cfg.vCodec)

	room := NewRoom(cfg.roomName, cfg.game, nil, conf, l)

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
	var frame emulator.GameFrame
	var wg sync.WaitGroup
	wg.Add(n)
	handler := room.emulator.GetVideo()
	room.emulator.SetVideo(func(video *emulator.GameFrame) {
		handler(video)
		if n > 0 {
			v := video.Data.Copy()
			frame = emulator.GameFrame{Duration: video.Duration, Data: &v}
			wg.Done()
		}
		n--
	})
	if !room.startEmulator {
		room.StartEmulator()
	}
	wg.Wait()
	return &frame
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
			game:   sushi,
			codecs: []encoder.VideoCodec{encoder.VP8},
			frames: 50,
		},
		{
			system: "gba",
			game:   sushi,
			codecs: []encoder.VideoCodec{encoder.VP8, encoder.H264},
			frames: 100,
		},
		{
			system: "nes",
			game:   mario,
			codecs: []encoder.VideoCodec{encoder.VP8, encoder.H264},
			frames: 100,
		},
	}

	for _, bench := range benches {
		for _, cod := range bench.codecs {
			b.Run(fmt.Sprintf("%s-%v-%d", bench.system, cod, bench.frames), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					room := getRoomMock(
						roomMockConfig{gamesPath: whereIsGames, game: bench.game, vCodec: cod, noLog: true})
					b.StartTimer()
					waitNFrames(bench.frames, room)
					b.StopTimer()
					room.Room.Close()
				}
			})
		}
	}
}
