package gamelist

import (
	"os"
	"path/filepath"

	"github.com/giongto35/cloud-game/config"
)

// getGameList returns list of games stored in games
func GetGameList(gamePath string) []string {
	var games []string
	filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && isValidGameType(path) {
			// Remove prefix to obtain file names
			path = path[len(gamePath)+1:]
			// Add to games list
			games = append(games, path)
		}
		return nil
	})

	return games
}

func isValidGameType(gamePath string) bool {
	ext := filepath.Ext(gamePath)[1:]
	_, ok := config.FileTypeToEmulator[ext]
	return ok
}
