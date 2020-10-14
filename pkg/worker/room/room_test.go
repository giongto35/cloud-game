package room

import (
	"fmt"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	storage "github.com/giongto35/cloud-game/v2/pkg/worker/cloud-storage"
)

type roomMock struct {
	Room
}

type roomMockConfig struct {
	roomName string
	game     games.GameMetadata
	codec    string
}

// Restricts a re-config call
// to only one invocation.
var configOnce sync.Once

func TestRoom(t *testing.T) {
	appPath := getAppPath() + "/assets/games/"

	tests := []struct {
		roomName string
		game     games.GameMetadata
		codec    string
		frames   int
	}{
		{
			roomName: "",
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: appPath + "Super Mario Bros.nes",
			},
			codec:  config.CODEC_VP8,
			frames: 5,
		},
	}

	for _, test := range tests {
		room := getRoomMock(roomMockConfig{
			roomName: test.roomName,
			game:     test.game,
			codec:    test.codec,
		})
		t.Logf("The game [%v] has been loaded\n", test.game.Name)

		var waitCounter sync.WaitGroup
		waitCounter.Add(test.frames)

		go func() {
			for range room.encoder.GetOutputChan() {
				waitCounter.Done()
			}
		}()

		waitCounter.Wait()
		room.Close()
	}
}

// getRoomMock returns mocked Room struct.
func getRoomMock(cfg roomMockConfig) roomMock {
	roomStorage := storage.NewInitClient()
	workerConfig := worker.NewDefaultConfig()
	configOnce.Do(fixEmulatorPaths)
	room := NewRoom(cfg.roomName, cfg.game, cfg.codec, roomStorage, workerConfig)

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
	//gamelist.GameList = gamelist.GetAllGames(appPath + "/assets")

	for k, conf := range config.EmulatorConfig {
		conf.Path = appPath + conf.Path
		config.EmulatorConfig[k] = conf
	}
}

// getAppPath returns absolute path to the assets directory.
func getAppPath() string {
	appName := "cloud-game"
	_, b, _, _ := runtime.Caller(0)
	return filepath.Dir(strings.SplitAfter(b, appName)[0]) + "/" + appName + "/"
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
			roomName: "",
			game:     rom,
			codec:    codec,
		})

		var waitCounter sync.WaitGroup
		waitCounter.Add(frames)

		go func() {
			for range room.encoder.GetOutputChan() {
				waitCounter.Done()
			}
		}()

		waitCounter.Wait()
		room.Close()
	}
}

// Measures emulation performance of various
// emulators and encoding options.
func BenchmarkRoom(b *testing.B) {
	appPath := getAppPath() + "/assets/games/"

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
				Path: appPath + "Sushi The Cat.gba",
			},
			codecs: []string{"vp8"},
			frames: 50,
		},
		{
			system: "gba",
			game: games.GameMetadata{
				Name: "Sushi The Cat",
				Type: "gba",
				Path: appPath + "Sushi The Cat.gba",
			},
			codecs: []string{"vp8", "x264"},
			frames: 100,
		},
		{
			system: "nes",
			game: games.GameMetadata{
				Name: "Super Mario Bros",
				Type: "nes",
				Path: appPath + "Super Mario Bros.nes",
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
