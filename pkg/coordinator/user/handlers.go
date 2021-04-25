package user

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/session"
)

func (u *User) HandleWebrtcInit() {
	if u.Worker == nil {
		return
	}
	// initWebrtc now only sends signal to worker, asks it to createOffer
	// relay request to target worker
	// worker creates a PeerConnection, and createOffer
	// send SDP back to browser

	defer u.Println("Received SDP from worker -> sending back to browser")
	resp := u.Worker.SyncSend(cws.WSPacket{ID: api.InitWebrtc, SessionID: u.Id})

	if resp != cws.EmptyPacket && resp.ID == api.Offer {
		u.SendWebrtcOffer(resp.Data)
	}
}

func (u *User) HandleWebrtcAnswer(data interface{}) {
	u.Worker.SendPacket(cws.WSPacket{ID: api.Answer, SessionID: u.Id, Data: data.(string)})
}

func (u *User) HandleWebrtcIceCandidate(data interface{}) {
	u.Worker.SendPacket(cws.WSPacket{ID: api.IceCandidate, SessionID: u.Id, Data: data.(string)})
}

func (u *User) HandleStartGame(data interface{}, library games.GameLibrary) {
	// +injects game data into the original game request
	request := api.GameStartRequest{}
	if err := request.From(data.(string)); err != nil {
		u.Printf("err: %v", err)
		return
	}
	gameStartCall, err := newNewGameStartCall(request, library)
	if err != nil {
		u.Printf("err: %v", err)
		return
	}
	packet, err := gameStartCall.To()
	if err != nil {
		u.Printf("err: %v", err)
		return
	}

	workerResp := u.Worker.SyncSend(cws.WSPacket{
		ID: api.Start, SessionID: u.Id, RoomID: request.RoomId, Data: packet})

	// Response from worker contains initialized roomID. Set roomID to the session
	u.RoomID = workerResp.RoomID
	u.Println("Received room response from browser: ", workerResp.RoomID)

	if err = u.StartGame(); err != nil {
		u.Printf("can't send back start request")
		return
	}
}

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

func (u *User) HandleQuitGame(data interface{}) {
	request := api.GameQuitRequest{}
	if err := request.From(data.(string)); err != nil {
		u.Printf("err: %v", err)
		return
	}
	u.Worker.SyncSend(cws.WSPacket{ID: api.GameQuit, SessionID: u.Id, RoomID: request.RoomId})
}

func (u *User) HandleSaveGame() {
	// TODO: Async
	response := u.Worker.SyncSend(cws.WSPacket{ID: api.GameSave, SessionID: u.Id, RoomID: u.RoomID})
	u.Printf("SAVE result: %v", response.Data) // TODO: Async
	u.Notify(SaveGame, response.Data)
}

func (u *User) HandleLoadGame() {
	// TODO: Async
	response := u.Worker.SyncSend(cws.WSPacket{ID: api.GameLoad, SessionID: u.Id, RoomID: u.RoomID})
	u.Printf("LOAD result: %v", response.Data)
	u.Notify(LoadGame, response.Data)
}

func (u *User) HandleChangePlayer(data interface{}) {
	v, ok := data.(string)
	if !ok {
		u.Printf("can't convert %v", v)
		return
	}
	// TODO: Async
	response := u.Worker.SyncSend(cws.WSPacket{
		ID:        api.GamePlayerSelect,
		SessionID: u.Id,
		RoomID:    u.RoomID,
		Data:      v,
	})
	u.Printf("Player index result: %v", response.Data)

	if response.Data == "error" {
		u.Printf("Player switch failed for some reason")
	}
	idx, _ := strconv.Atoi(response.Data)
	u.Notify(ChangePlayer, idx)
}

func (u *User) HandleToggleMultitap() {
	// TODO: Async
	response := u.Worker.SyncSend(cws.WSPacket{ID: api.GameMultitap, SessionID: u.Id, RoomID: u.RoomID})
	u.Printf("MULTITAP result: %v", response.Data)
}
