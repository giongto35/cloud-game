package coordinator

import (
	"encoding/json"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
)

func (u *User) HandleWebrtcInit() {
	if u.Worker == nil {
		return
	}
	resp, err := u.Worker.WebrtcInit(u.Id())
	if err != nil || resp == "" {
		u.Printf("error: webrtc init failed, %v", err)
		return
	}
	u.SendWebrtcOffer(resp)
}

func (u *User) HandleWebrtcAnswer(data json.RawMessage) {
	var req api.WebrtcAnswerUserRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		u.Printf("error: broken webrtc answer request %v", err)
		return
	}
	u.Worker.WebrtcAnswer(u.Id(), req)
}

func (u *User) HandleWebrtcIceCandidate(data json.RawMessage) {
	var req api.WebrtcIceCandidateUserRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		u.Printf("error: broken ICE candidate request %v", err)
		return
	}
	u.Worker.WebrtcIceCandidate(u.Id(), req)
}

func (u *User) HandleStartGame(data json.RawMessage, launcher launcher.Launcher) {
	var req api.GameStartUserRequest
	err := json.Unmarshal(data, &req)
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

	workerResp, err := u.Worker.StartGame(u.Id(), req.RoomId, req.PlayerIndex, gameInfo)
	if err != nil {
		u.Printf("err: %v", err)
		return
	}
	// Response from worker contains initialized roomID. Set roomID to the session
	u.AssignRoom(workerResp.RoomId)
	u.Printf("Received room response from worker: ", workerResp.RoomId)

	if err = u.StartGame(); err != nil {
		u.Printf("can't send back start request")
		return
	}
}

func (u *User) HandleQuitGame(data json.RawMessage) {
	var req api.GameQuitRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		u.Printf("error: broken game quit request %v", err)
		return
	}
	u.Worker.QuitGame(u.Id(), req.RoomId)
}

func (u *User) HandleSaveGame() {
	// TODO: Async
	resp, err := u.Worker.SaveGame(u.Id(), u.RoomID)
	if err != nil {
		u.Printf("error: broken game save request %v", err)
		return
	}
	u.Notify(api.SaveGame, resp)
}

func (u *User) HandleLoadGame() {
	// TODO: Async
	resp, err := u.Worker.LoadGame(u.Id(), u.RoomID)
	if err != nil {
		u.Printf("error: broken game load request %v", err)
		return
	}
	u.Notify(api.LoadGame, resp)
}

func (u *User) HandleChangePlayer(data json.RawMessage) {
	var req api.ChangePlayerUserRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		u.Printf("error: broken player change request %v", err)
		return
	}
	// TODO: Async
	resp, err := u.Worker.ChangePlayer(u.Id(), u.RoomID, req)
	if err != nil || resp == "error" {
		u.Printf("error: player switch failed for some reason")
	}
	idx, err := strconv.Atoi(resp)
	if err != nil {
		u.Printf("error: broken player change response %v", err)
		return
	}
	u.Notify(api.ChangePlayer, idx)
}

func (u *User) HandleToggleMultitap() {
	// TODO: Async
	u.Worker.ToggleMultitap(u.Id(), u.RoomID)
}
