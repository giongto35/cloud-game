package gamelist

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/config"
)

type GameInfo struct {
	Name string
	Path string
	Type string
}

const gamePath = "games"

var GameList []GameInfo

func init() {
	GameList = getAllGames(gamePath)
	log.Println("gamelsit ", GameList)
}

// GetGameInfoFromName returns game info from a gameName
func GetGameInfoFromName(name string) GameInfo {
	for _, game := range GameList {
		if game.Name == name {
			return game
		}
	}

	return GameInfo{}
}

// getAllGames returns list of games stored in games. This call should be called when server start (package init)
// TODO: Maybe later we need to make realtime update without server restart
func getAllGames(gamePath string) []GameInfo {
	var games []GameInfo

	filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && isValidGameType(path) {
			// Add to games list
			gameInfo := getGameInfo(path)
			games = append(games, gameInfo)
		}
		return nil
	})

	return games
}

// isValidGameType check if a game is valid for cloud retro based on extension
func isValidGameType(gamePath string) bool {
	ext := filepath.Ext(gamePath)[1:]
	_, ok := config.FileTypeToEmulator[ext]
	return ok
}

// getGameInfo returns game info from a path
func getGameInfo(path string) GameInfo {
	// Remove prefix to obtain file names
	fileName := path[len(gamePath)+1:]
	ext := filepath.Ext(fileName)
	return GameInfo{
		Name: strings.TrimSuffix(fileName, ext),
		Type: ext[1:],
		Path: path,
	}
}
