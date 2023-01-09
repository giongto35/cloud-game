package games

import "fmt"

type Launcher interface {
	FindAppByName(name string) (AppMeta, error)
	ExtractAppNameFromUrl(name string) string
	GetAppNames() []string
}

type AppMeta struct {
	Name string
	Type string
	Base string
	Path string
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
	return AppMeta{Name: game.Name, Base: game.Base, Type: game.Type, Path: game.Path}, nil
}

func (gl GameLauncher) ExtractAppNameFromUrl(name string) string {
	return GetGameNameFromRoomID(name)
}

func (gl GameLauncher) GetAppNames() []string {
	var gameList []string
	for _, game := range gl.lib.GetAll() {
		gameList = append(gameList, game.Name)
	}
	return gameList
}
