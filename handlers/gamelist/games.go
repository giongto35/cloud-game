package gamelist

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const gamePath = "games"

// getGameList returns list of games stored in games
func GetGameList() []string {
	var games []string
	filepath.Walk(gamePath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			// remove prefix
			path = path[len(gamePath)+1:]
			// Add to games list
			games = append(games, path)
		}
		return nil
	})

	return games
}

// GetEncodedGameList returns game list in encoded wspacket format
func GetEncodedGameList() string {
	encodedList, _ := json.Marshal(GetGameList())
	return string(encodedList)
}
