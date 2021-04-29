package coordinator

import (
	"encoding/json"
	"errors"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type WebrtcInitOutRequest struct {
	api.StatefulRequest
}
type WebrtcInitOutResponse = string

type TerminateSessionOutRequest struct {
	api.StatefulRequest
}

func (w *Worker) AssignId(id network.Uid) {
	_ = w.SendAndForget(api.IdentifyWorker, id)
}

func (w *Worker) WebrtcInit(id network.Uid) (WebrtcInitOutResponse, error) {
	data, err := w.Send(api.WebrtcInit, WebrtcInitOutRequest{api.StatefulRequest{Id: id}})
	if err != nil {
		return "", errors.New("request error")
	}
	var resp string
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", api.ConvertErr
	}
	return resp, nil
}

func (w *Worker) WebrtcAnswer(id network.Uid, sdp string) {
	_ = w.SendAndForget(api.WebrtcAnswer, api.WebrtcAnswerRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		Sdp:             sdp,
	})
}

func (w *Worker) WebrtcIceCandidate(id network.Uid, candidate string) {
	_ = w.SendAndForget(api.WebrtcIceCandidate, api.WebrtcIceCandidateRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		Candidate:       candidate,
	})
}

func (w *Worker) StartGame(id network.Uid, roomId string, idx int, game launcher.AppMeta) (api.StartGameResponse, error) {
	data, err := w.Send(api.StartGame, api.StartGameRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		Game: api.GameInfo{
			Name: game.Name,
			Path: game.Path,
			Type: game.Type,
		},
		RoomId:      roomId,
		PlayerIndex: idx,
	})
	var resp api.StartGameResponse
	if err != nil {
		return resp, errors.New("request error")
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return resp, api.ConvertErr
	}
	return resp, nil
}

func (w *Worker) QuitGame(id network.Uid, roomId string) {
	_, _ = w.Send(api.QuitGame, api.GameQuitRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		RoomId:          roomId,
	})
}

func (w *Worker) SaveGame(id network.Uid, roomId string) (api.SaveGameResponse, error) {
	data, err := w.Send(api.SaveGame, api.SaveGameRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		RoomId:          roomId,
	})
	var resp api.SaveGameResponse
	if err != nil {
		return resp, errors.New("request error")
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return resp, api.ConvertErr
	}
	return resp, nil
}

func (w *Worker) LoadGame(id network.Uid, roomId string) (api.LoadGameResponse, error) {
	data, err := w.Send(api.LoadGame, api.LoadGameRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		RoomId:          roomId,
	})
	var resp api.LoadGameResponse
	if err != nil {
		return resp, errors.New("request error")
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return resp, api.ConvertErr
	}
	return resp, nil
}

func (w *Worker) ChangePlayer(id network.Uid, roomId string, index string) (api.ChangePlayerResponse, error) {
	data, err := w.Send(api.ChangePlayer, api.ChangePlayerRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		RoomId:          roomId,
		Index:           index,
	})
	var resp api.ChangePlayerResponse
	if err != nil {
		return resp, errors.New("request error")
	}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return resp, api.ConvertErr
	}
	return resp, nil
}

func (w *Worker) ToggleMultitap(id network.Uid, roomId string) {
	_, _ = w.Send(api.ToggleMultitap, api.ToggleMultitapRequest{
		StatefulRequest: api.StatefulRequest{Id: id},
		RoomId:          roomId,
	})
}

func (w *Worker) TerminateSession(id network.Uid) {
	_, _ = w.Send(api.TerminateSession, TerminateSessionOutRequest{StatefulRequest: api.StatefulRequest{Id: id}})
}
