package launcher

import (
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/session"
)

type GameLauncher struct {
	lib games.GameLibrary
}

func NewGameLauncher(lib games.GameLibrary) GameLauncher {
	return GameLauncher{lib: lib}
}

func (gl GameLauncher) FindAppByName(name string) (AppMeta, error) {
	game := gl.lib.FindGameByName(name)
	if game.Path == "" {
		return AppMeta{}, fmt.Errorf("couldn't find game info for the game %v", name)
	}
	return AppMeta{Name: game.Name, Base: game.Base, Type: game.Type, Path: game.Path}, nil
}

func (gl GameLauncher) ExtractAppNameFromUrl(name string) string {
	return session.GetGameNameFromRoomID(name)
}

func (gl GameLauncher) GetAppNames() []string {
	var gameList []string
	for _, game := range gl.lib.GetAll() {
		gameList = append(gameList, game.Name)
	}
	return gameList
}
