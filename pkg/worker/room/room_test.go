package room

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/pkg/config"
	"github.com/giongto35/cloud-game/pkg/config/worker"
	"github.com/giongto35/cloud-game/pkg/util/gamelist"
	storage "github.com/giongto35/cloud-game/pkg/worker/cloud-storage"
)

type roomMock struct {
	Room
}

type roomMockConfig struct {
	roomName string
	gameName string
	codec    string
}

func TestRoom(t *testing.T) {
	tests := []struct {
		roomName string
		gameName string
		codec    string
		frames   int
	}{
		{
			roomName: "",
			gameName: "Super Mario Bros",
			codec:    config.CODEC_VP8,
			frames:   5,
		},
	}

	for _, test := range tests {
		room := getRoomMock(roomMockConfig{
			roomName: test.roomName,
			gameName: test.gameName,
			codec:    test.codec,
		})
		t.Logf("The game [%v] has been loaded\n", test.gameName)

		var waitCounter sync.WaitGroup
		waitCounter.Add(test.frames)

		go func() {
			for range room.encoder.GetOutputChan() {
				t.Logf("image\n")
				waitCounter.Done()
			}
		}()

		waitCounter.Wait()
		room.Close()

		// TODO add save delete
	}
}

func getRoomMock(cfg roomMockConfig) *roomMock {
	roomStorage := storage.NewInitClient()
	workerConfig := worker.NewDefaultConfig()

	appPath := getAppPath()
	gamelist.GameList = gamelist.GetAllGames(appPath + "/assets")

	for k, conf := range config.EmulatorConfig {
		conf.Path = appPath + conf.Path
		config.EmulatorConfig[k] = conf
	}

	room := NewRoom(cfg.roomName, cfg.gameName, cfg.codec, roomStorage, workerConfig)

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

	return &roomMock{Room: *room}
}

// Returns absolute path to the assets directory.
func getAppPath() string {
	appName := "cloud-game"
	_, b, _, _ := runtime.Caller(0)
	return filepath.Dir(strings.SplitAfter(b, appName)[0]) + "/" + appName + "/"
}
