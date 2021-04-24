package coordinator

import (
	"errors"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/session"
)

func newNewGameStartCall(request api.GameStartRequest, library games.GameLibrary) (api.GameStartCall, error) {
	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := request.GameName
	if request.RoomId != "" {
		// ! should be moved into coordinator
		name := session.GetGameNameFromRoomID(request.RoomId)
		if name == "" {
			return api.GameStartCall{}, errors.New("couldn't decode game name from the room id")
		}
		game = name
	}

	gameInfo := library.FindGameByName(game)
	if gameInfo.Path == "" {
		return api.GameStartCall{}, fmt.Errorf("couldn't find game info for the game %v", game)
	}

	return api.GameStartCall{
		Name: gameInfo.Name,
		Path: gameInfo.Path,
		Type: gameInfo.Type,
	}, nil
}
