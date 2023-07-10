package room

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

	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	canvas "github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/image"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/worker/media"
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

type testRoom struct {
	*Room[*GameSession]
	started bool
}

type codec = encoder.VideoCodec

type conf struct {
	roomName      string
	game          games.GameMetadata
	codec         codec
	autoGlContext bool
	autoAppStart  bool
	noLog         bool
}

func (r testRoom) Close() {
	r.Room.Close()
	time.Sleep(2 * time.Second) // hack: wait room destruction (atm impossible to tell)
}

func (r testRoom) WaitFrames(n int) canvas.Frame {
	var frame canvas.Frame
	var wg sync.WaitGroup
	wg.Add(n)
	WithEmulator(r.app).SetVideoCb(func(v app.Video) {
		if n > 0 {
			frame = (&canvas.Frame{RGBA: v.Frame}).Copy()
			wg.Done()
		}
		n--
	})
	if !r.started {
		r.StartApp()
	}
	wg.Wait()
	return frame
}

type testParams struct {
	system string
	game   games.GameMetadata
	codecs []codec
	frames int
}

// Store absolute path to test games
var testTempDir = filepath.Join(os.TempDir(), "cloud-game-core-tests")
var root = ""

// games
var (
	alwas = games.GameMetadata{Name: "Alwa's Awakening (Demo)", Type: "nes", Path: "Alwa's Awakening (Demo).nes", System: "nes"}
	sushi = games.GameMetadata{Name: "Sushi The Cat", Type: "gba", Path: "Sushi The Cat.gba", System: "gba"}
	fd    = games.GameMetadata{Name: "Florian Demo", Type: "n64", Path: "Sample Demo by Florian (PD).z64", System: "n64"}
)

func init() {
	runtime.LockOSThread()
	p, _ := filepath.Abs("../../../")
	root = p + string(filepath.Separator)
}

func TestMain(m *testing.M) {
	flag.BoolVar(&renderFrames, "renderFrames", false, "Render frames for eye testing purposes")
	flag.StringVar(&outputPath, "outputPath", "./", "Output path for generated files")
	flag.BoolVar(&autoGlContext, "autoGlContext", false, "Set auto GL context choose for headless machines")

	thread.Wrap(func() { os.Exit(m.Run()) })
}

func TestRoom(t *testing.T) {
	tests := []testParams{
		{game: alwas, codecs: []codec{encoder.H264}, frames: 300},
	}

	for _, test := range tests {
		room := room(conf{codec: test.codecs[0], game: test.game})
		room.WaitFrames(test.frames)
		room.Close()
	}
}

func TestAll(t *testing.T) {
	tests := []testParams{
		{game: sushi, frames: 150},
		{game: alwas, frames: 50},
		{game: fd, frames: 50, system: "main-thread"},
	}

	crc32q := crc32.MakeTable(0xD5828281)

	for _, test := range tests {
		room := room(conf{game: test.game, codec: encoder.VP8, autoGlContext: autoGlContext, autoAppStart: false})
		var frame canvas.Frame
		if test.system == "main-thread" {
			thread.Main(func() {
				frame = room.WaitFrames(test.frames)
				room.Close()
			})
		} else {
			frame = room.WaitFrames(test.frames)
			room.Close()
		}
		if renderFrames {
			tag := fmt.Sprintf("%v-%v-0x%08x", runtime.GOOS, test.game.Type, crc32.Checksum(frame.Pix, crc32q))
			dumpCanvas(&frame, tag, fmt.Sprintf("%v [%v]", tag, test.frames), outputPath)
		}
	}
}

func dumpCanvas(frame *canvas.Frame, name string, caption string, path string) {
	// slap 'em caption
	if caption != "" {
		draw.Draw(frame, image.Rect(8, 8, 8+len(caption)*7+3, 24), &image.Uniform{C: color.RGBA{}}, image.Point{}, draw.Src)
		(&font.Drawer{
			Dst:  frame,
			Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			Face: basicfont.Face7x13,
			Dot:  fixed.Point26_6{X: fixed.Int26_6(10 * 64), Y: fixed.Int26_6(20 * 64)},
		}).DrawString(caption)
	}

	outPath := testTempDir
	if path != "" {
		outPath = path
	}

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

// room returns mocked Room struct.
func room(cfg conf) testRoom {
	var conf config.WorkerConfig
	if _, err := config.LoadConfig(&conf, ""); err != nil {
		panic(err)
	}

	conf.Worker.Library.BasePath = filepath.FromSlash(root + "/assets/games")

	fixEmulators(&conf, cfg.autoGlContext)
	l := logger.NewConsole(conf.Worker.Debug, "w", false)
	if cfg.noLog {
		logger.SetGlobalLevel(logger.Disabled)
	}

	conf.Encoder.Video.Codec = string(cfg.codec)

	id := cfg.roomName
	if id == "" {
		id = games.GenerateRoomID(cfg.game.Name)
	}

	manager := caged.NewManager(l)
	if err := manager.Load(caged.Libretro, conf); err != nil {
		l.Fatal().Msgf("couldn't cage libretro: %v", err)
	}

	emu := WithEmulator(manager.Get(caged.Libretro))
	emu.ReloadFrontend()
	emu.SetSessionId(id)
	if err := emu.Load(cfg.game, conf.Worker.Library.BasePath); err != nil {
		l.Fatal().Err(err).Msgf("couldn't load the game %v", cfg.game)
	}

	m := media.NewWebRtcMediaPipe(conf.Encoder.Audio, conf.Encoder.Video, l)
	m.AudioSrcHz = emu.AudioSampleRate()
	m.AudioFrame = conf.Encoder.Audio.Frame
	m.VideoW, m.VideoH = emu.ViewportSize()
	if err := m.Init(); err != nil {
		l.Fatal().Err(err).Msgf("no init")
	}

	room := NewRoom[*GameSession](id, emu, &com.NetMap[string, *GameSession]{}, m)
	if cfg.autoAppStart {
		room.StartApp()
	}

	return testRoom{Room: room, started: cfg.autoAppStart}
}

// fixEmulators makes absolute game paths in global GameList and passes GL context config.
// hack: emulator paths should be absolute and visible to the tests.
func fixEmulators(config *config.WorkerConfig, autoGlContext bool) {
	config.Emulator.Libretro.Cores.Paths.Libs =
		filepath.FromSlash(root + config.Emulator.Libretro.Cores.Paths.Libs)
	config.Emulator.LocalPath = filepath.FromSlash(filepath.Join(root, "tests", config.Emulator.LocalPath))
	config.Emulator.Storage = filepath.FromSlash(filepath.Join(root, "tests", "storage"))

	for k, conf := range config.Emulator.Libretro.Cores.List {
		if conf.IsGlAllowed && autoGlContext {
			conf.AutoGlContext = true
		}
		config.Emulator.Libretro.Cores.List[k] = conf
	}
}

// Measures emulation performance of various
// emulators and encoding options.
func BenchmarkRoom(b *testing.B) {
	benches := []testParams{
		// warm up
		{system: "gba", game: sushi, codecs: []codec{encoder.VP8}, frames: 50},
		{system: "gba", game: sushi, codecs: []codec{encoder.VP8, encoder.H264}, frames: 100},
		{system: "nes", game: alwas, codecs: []codec{encoder.VP8, encoder.H264}, frames: 100},
	}

	for _, bench := range benches {
		for _, cod := range bench.codecs {
			b.Run(fmt.Sprintf("%s-%v-%d", bench.system, cod, bench.frames), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					room := room(conf{game: bench.game, codec: cod, noLog: true})
					b.StartTimer()
					room.WaitFrames(bench.frames)
					b.StopTimer()
					room.Room.Close()
				}
			})
		}
	}
}

type tSession struct{}

func (t tSession) SendAudio([]byte, int32) {}
func (t tSession) SendVideo([]byte, int32) {}
func (t tSession) SendData([]byte)         {}
func (t tSession) Disconnect()             {}
func (t tSession) Id() string              { return "1" }

func TestRouter(t *testing.T) {
	u := com.NewNetMap[string, *tSession]()
	router := Router[*tSession]{users: &u}

	var r *Room[*tSession]

	router.SetRoom(&Room[*tSession]{id: "test001"})
	room := router.FindRoom("test001")
	if room == nil {
		t.Errorf("no room, but should be")
	}
	router.SetRoom(r)
	room = router.FindRoom("x")
	if room != nil {
		t.Errorf("a room, but should not be")
	}
	router.SetRoom(nil)
	router.Close()
}
