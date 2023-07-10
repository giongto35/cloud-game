package games

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type Launcher interface {
	FindAppByName(name string) (AppMeta, error)
	ExtractAppNameFromUrl(name string) string
	GetAppNames() []AppMeta
}

type AppMeta struct {
	Base   string
	Name   string
	Path   string
	System string
	Type   string
}

type GameLauncher struct {
	lib GameLibrary
}

func NewGameLauncher(lib GameLibrary) GameLauncher { return GameLauncher{lib: lib} }

func (gl GameLauncher) FindAppByName(name string) (AppMeta, error) {
	game := gl.lib.FindGameByName(name)
	if game.Path == "" {
		return AppMeta{}, fmt.Errorf("couldn't find game info for the game %v", name)
	}
	return AppMeta(game), nil
}

func (gl GameLauncher) ExtractAppNameFromUrl(name string) string { return ExtractGame(name) }

func (gl GameLauncher) GetAppNames() (apps []AppMeta) {
	for _, game := range gl.lib.GetAll() {
		apps = append(apps, AppMeta{Name: game.Name, System: game.System})
	}
	return
}

const separator = "___"

// ExtractGame parses game room link returning the name of the game "encoded" there.
func ExtractGame(roomID string) string {
	parts := strings.Split(roomID, separator)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// GenerateRoomID generate a unique room ID containing 16 digits.
// RoomID contains random number + gameName
// Next time when we only get roomID, we can launch game based on gameName
func GenerateRoomID(title string) string {
	return strconv.FormatInt(rand.Int63(), 16) + separator + title
}
