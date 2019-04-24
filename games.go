package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const gamePath = "games"

// getGameList returns list of games stored in games
func getGameList() []string {
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

func getEncodedGameList() string {
	encodedList, _ := json.Marshal(getGameList())
	return string(encodedList)
}
