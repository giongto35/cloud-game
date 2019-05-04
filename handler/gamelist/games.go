package gamelist

import (
	"encoding/json"
	"fmt"
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

// GetEncodedGameList returns game list in encoded wspacket format
func GetEncodedGameList(gamePath string) string {
	encodedList, _ := json.Marshal(GetGameList(gamePath))
	fmt.Println(encodedList)
	return string(encodedList)
}
