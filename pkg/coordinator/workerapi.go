package coordinator

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

func (w *Worker) WebrtcInit(id network.Uid) (api.WebrtcInitResponse, error) {
	data, err := w.Send(api.WebrtcInit, api.WebrtcInitRequest{Stateful: api.Stateful{Id: id}})
	var resp string
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (w *Worker) WebrtcAnswer(id network.Uid, sdp string) {
	_ = w.SendAndForget(api.WebrtcAnswer, api.WebrtcAnswerRequest{
		Stateful: api.Stateful{Id: id},
		Sdp:      sdp,
	})
}

func (w *Worker) WebrtcIceCandidate(id network.Uid, candidate string) {
	_ = w.SendAndForget(api.WebrtcIceCandidate, api.WebrtcIceCandidateRequest{
		Stateful:  api.Stateful{Id: id},
		Candidate: candidate,
	})
}

func (w *Worker) StartGame(id network.Uid, roomId string, idx int, app launcher.AppMeta) (api.StartGameResponse, error) {
	data, err := w.Send(api.StartGame, api.StartGameRequest{
		Stateful:    api.Stateful{Id: id},
		Game:        api.GameInfo{Name: app.Name, Base: app.Base, Path: app.Path, Type: app.Type},
		Room:        api.Room{Id: roomId},
		PlayerIndex: idx,
	})
	var resp api.StartGameResponse
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (w *Worker) QuitGame(id network.Uid, roomId string) {
	_ = w.SendAndForget(api.QuitGame, api.GameQuitRequest{
		Stateful: api.Stateful{Id: id},
		Room:     api.Room{Id: roomId},
	})
}

func (w *Worker) SaveGame(id network.Uid, roomId string) (api.SaveGameResponse, error) {
	data, err := w.Send(api.SaveGame, api.SaveGameRequest{
		Stateful: api.Stateful{Id: id},
		Room:     api.Room{Id: roomId},
	})
	var resp api.SaveGameResponse
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (w *Worker) LoadGame(id network.Uid, roomId string) (api.LoadGameResponse, error) {
	data, err := w.Send(api.LoadGame, api.LoadGameRequest{
		Stateful: api.Stateful{Id: id},
		Room:     api.Room{Id: roomId},
	})
	var resp api.LoadGameResponse
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (w *Worker) ChangePlayer(id network.Uid, roomId string, index string) (api.ChangePlayerResponse, error) {
	data, err := w.Send(api.ChangePlayer, api.ChangePlayerRequest{
		Stateful: api.Stateful{Id: id},
		Room:     api.Room{Id: roomId},
		Index:    index,
	})
	var resp api.ChangePlayerResponse
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(data, &resp)
	return resp, err
}

func (w *Worker) ToggleMultitap(id network.Uid, roomId string) {
	_, _ = w.Send(api.ToggleMultitap, api.ToggleMultitapRequest{
		Stateful: api.Stateful{Id: id},
		Room:     api.Room{Id: roomId},
	})
}

func (w *Worker) TerminateSession(id network.Uid) {
	_, _ = w.Send(api.TerminateSession, api.TerminateSessionRequest{Stateful: api.Stateful{Id: id}})
}
