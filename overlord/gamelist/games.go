package gamelist

import (
	"os"
	"path/filepath"
)

// getGameList returns list of games stored in games
// TODO: change to class
func GetGameList(gamePath string) []string {
	var games []string
	filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			// Remove prefix to obtain file names
			path = path[len(gamePath)+1:]
			// Add to games list
			games = append(games, path)
		}
		return nil
	})

	return games
}
