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
	"github.com/giongto35/cloud-game/v3/pkg/encoder"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/color/bgra"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/color/rgb565"
	"github.com/giongto35/cloud-game/v3/pkg/encoder/color/rgba"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/media"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	_ "github.com/giongto35/cloud-game/v3/test"
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

func (r testRoom) WaitFrame(n int) app.RawFrame {
	var wg sync.WaitGroup
	wg.Add(1)
	target := app.RawFrame{}
	WithEmulator(r.app).SetVideoCb(func(v app.Video) {
		if n == 1 {
			target = v.Frame
			target.Data = make([]byte, len(v.Frame.Data))
			copy(target.Data, v.Frame.Data)
			wg.Done()
		}
		n--
	})
	if !r.started {
		r.StartApp()
	}
	wg.Wait()
	return target
}

type testParams struct {
	system string
	game   games.GameMetadata
	codecs []codec
	frames int
	color  int
}

// Store absolute path to test games
var testTempDir = filepath.Join(os.TempDir(), "cloud-game-core-tests")

// games
var (
	alwas = games.GameMetadata{Name: "Alwa's Awakening (Demo)", Type: "nes", Path: "nes/Alwa's Awakening (Demo).nes", System: "nes"}
	sushi = games.GameMetadata{Name: "Sushi The Cat", Type: "gba", Path: "gba/Sushi The Cat.gba", System: "gba"}
	fd    = games.GameMetadata{Name: "Florian Demo", Type: "n64", Path: "n64/Sample Demo by Florian (PD).z64", System: "n64"}
	rogue = games.GameMetadata{Name: "Rogue", Type: "dos", Path: "dos/rogue.zip", System: "dos"}
)

func TestMain(m *testing.M) {
	flag.BoolVar(&renderFrames, "renderFrames", false, "Render frames for eye testing purposes")
	flag.StringVar(&outputPath, "outputPath", "./", "Output path for generated files")
	flag.BoolVar(&autoGlContext, "autoGlContext", false, "Set auto GL context choose for headless machines")

	thread.Wrap(func() { os.Exit(m.Run()) })
}

func TestRoom(t *testing.T) {
	tests := []testParams{
		{game: alwas, codecs: []codec{encoder.H264, encoder.VP8, encoder.VP9}, frames: 300},
	}

	for _, test := range tests {
		for _, codec := range test.codecs {
			room := room(conf{codec: codec, game: test.game})
			room.WaitFrame(test.frames)
			room.Close()
		}
	}
}

func TestAll(t *testing.T) {
	tests := []testParams{
		{game: sushi, frames: 150, color: 2},
		{game: alwas, frames: 50, color: 1},
		{game: fd, frames: 50, system: "gl", color: 1},
		{game: rogue, frames: 33, color: 1},
	}

	crc32q := crc32.MakeTable(0xD5828281)

	for _, test := range tests {
		var frame app.RawFrame
		room := room(conf{game: test.game, codec: encoder.VP8, autoGlContext: autoGlContext, autoAppStart: false})
		flip := test.system == "gl"
		thread.Main(func() { frame = room.WaitFrame(test.frames) })
		room.Close()

		if renderFrames {
			rect := image.Rect(0, 0, frame.W, frame.H)
			var src image.Image
			src1 := bgra.NewBGRA(rect)
			src1.Pix = frame.Data
			src1.Stride = frame.Stride
			src = src1
			if test.color == 2 {
				src2 := rgb565.NewRGB565(rect)
				src2.Pix = frame.Data
				src2.Stride = frame.Stride
				src = src2
			}
			dst := rgba.ToRGBA(src, flip)
			tag := fmt.Sprintf("%v-%v-0x%08x", runtime.GOOS, test.game.Type, crc32.Checksum(frame.Data, crc32q))
			dumpCanvas(dst, tag, fmt.Sprintf("%v [%v]", tag, test.frames), outputPath)
		}
	}
}

func dumpCanvas(frame *image.RGBA, name string, caption string, path string) {
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

	conf.Emulator.Libretro.Cores.Repo.ExtLock = expand("tests", ".cr", "cloud-game.lock")
	conf.Emulator.LocalPath = expand("tests", conf.Emulator.LocalPath)
	conf.Emulator.Storage = expand("tests", "storage")

	conf.Encoder.Video.Codec = string(cfg.codec)

	l := logger.NewConsole(conf.Worker.Debug, "w", false)
	if cfg.noLog {
		logger.SetGlobalLevel(logger.Disabled)
	}

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
	if err := emu.Load(cfg.game, conf.Library.BasePath); err != nil {
		l.Fatal().Err(err).Msgf("couldn't load the game %v", cfg.game)
	}

	m := media.NewWebRtcMediaPipe(conf.Encoder.Audio, conf.Encoder.Video, l)
	m.AudioSrcHz = emu.AudioSampleRate()
	m.AudioFrames = conf.Encoder.Audio.Frames
	m.VideoW, m.VideoH = emu.ViewportSize()
	m.VideoScale = emu.Scale()
	if err := m.Init(); err != nil {
		l.Fatal().Err(err).Msgf("no init")
	}

	room := NewRoom[*GameSession](id, emu, &com.NetMap[string, *GameSession]{}, m)
	if cfg.autoAppStart {
		room.StartApp()
	}

	return testRoom{Room: room, started: cfg.autoAppStart}
}

// Measures emulation performance of various
// emulators and encoding options.
func BenchmarkRoom(b *testing.B) {
	benches := []testParams{
		// warm up
		{system: "gba", game: sushi, codecs: []codec{encoder.VP8, encoder.VP9}, frames: 50},
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
					room.WaitFrame(bench.frames)
					b.StopTimer()
					room.Room.Close()
				}
			})
		}
	}
}

// expand joins a list of file path elements.
func expand(p ...string) string {
	ph, _ := filepath.Abs(filepath.FromSlash(filepath.Join(p...)))
	return ph
}
