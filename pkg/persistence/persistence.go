package persistence

import (
	"log"
	"os/user"
)

var homeDir string

func init() {
	u, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}
	homeDir = u.HomeDir
}

// GetSavePath returns save directory for games.
// Path ends with /.
func GetSavePath() string { return homeDir + "/.cr/save/" }

// GetMainState returns main state file of the game in the room.
func GetMainState(hash string) string { return GetSavePath() + hash + ".dat" }
