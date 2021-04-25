package user

import (
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

func (u *User) HandleWebrtcInit() {
	if u.Worker == nil {
		return
	}
	resp := u.Worker.SyncSend(cws.WSPacket{ID: api.InitWebrtc, SessionID: u.Id})
	if resp != cws.EmptyPacket && resp.ID == api.Offer {
		u.SendWebrtcOffer(resp.Data)
	}
}

func (u *User) HandleWebrtcAnswer(data interface{}) {
	req, err := webrtcAnswerInRequest(data)
	if err != nil {
		u.Printf("error: broken webrtc answer request %v", err)
		return
	}
	u.Worker.SendPacket(cws.WSPacket{ID: api.Answer, SessionID: u.Id, Data: req})
}

func (u *User) HandleWebrtcIceCandidate(data interface{}) {
	req, err := webrtcIceCandidateInRequest(data)
	if err != nil {
		u.Printf("error: broken webrtc answer request %v", err)
		return
	}
	u.Worker.SendPacket(cws.WSPacket{ID: api.IceCandidate, SessionID: u.Id, Data: req})
}

func (u *User) HandleStartGame(data interface{}, launcher launcher.Launcher) {
	req, err := gameStartInRequest(data)
	if err != nil {
		u.Printf("error: broken game start request %v", err)
		return
	}
	// +injects game data into the original game request
	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := req.GameName
	if req.RoomId != "" {
		name := launcher.ExtractAppNameFromUrl(req.RoomId)
		if name == "" {
			u.Printf("couldn't decode game name from the room id")
			return
		}
		game = name
	}

	gameInfo, err := launcher.FindAppByName(game)
	if err != nil {
		u.Printf("couldn't find game info for the game %v", game)
		return
	}

	gameStartCall := api.GameStartCall{Name: gameInfo.Name, Path: gameInfo.Path, Type: gameInfo.Type}
	packet, err := gameStartCall.To()
	if err != nil {
		u.Printf("err: %v", err)
		return
	}

	workerResp := u.Worker.SyncSend(cws.WSPacket{ID: api.Start, SessionID: u.Id, RoomID: req.RoomId, Data: packet})
	// Response from worker contains initialized roomID. Set roomID to the session
	u.AssignRoom(workerResp.RoomID)
	u.Println("Received room response from worker: ", workerResp.RoomID)

	if err = u.StartGame(); err != nil {
		u.Printf("can't send back start request")
		return
	}
}

func (u *User) HandleQuitGame(data interface{}) {
	req, err := gameQuitInRequest(data)
	if err != nil {
		u.Printf("error: broken game quit request %v", err)
		return
	}
	u.Worker.SyncSend(cws.WSPacket{ID: api.GameQuit, SessionID: u.Id, RoomID: req.RoomId})
}

func (u *User) HandleSaveGame() {
	// TODO: Async
	resp := u.Worker.SyncSend(cws.WSPacket{ID: api.GameSave, SessionID: u.Id, RoomID: u.RoomID})
	u.Notify(SaveGame, resp.Data)
}

func (u *User) HandleLoadGame() {
	// TODO: Async
	resp := u.Worker.SyncSend(cws.WSPacket{ID: api.GameLoad, SessionID: u.Id, RoomID: u.RoomID})
	u.Notify(LoadGame, resp.Data)
}

func (u *User) HandleChangePlayer(data interface{}) {
	req, err := changePlayerInRequest(data)
	if err != nil {
		u.Printf("error: broken player change request %v", err)
		return
	}
	// TODO: Async
	resp := u.Worker.SyncSend(
		cws.WSPacket{ID: api.GamePlayerSelect, SessionID: u.Id, RoomID: u.RoomID, Data: req})
	if resp.Data == "error" {
		u.Printf("error: player switch failed for some reason")
	}
	idx, err := strconv.Atoi(resp.Data)
	if err != nil {
		u.Printf("error: broken player change response %v", err)
		return
	}
	u.Notify(ChangePlayer, idx)
}

func (u *User) HandleToggleMultitap() {
	// TODO: Async
	_ = u.Worker.SyncSend(cws.WSPacket{ID: api.GameMultitap, SessionID: u.Id, RoomID: u.RoomID})
}
