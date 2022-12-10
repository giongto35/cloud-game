package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/network"
)

func (w *Worker) WebrtcInit(id network.Uid) (*api.WebrtcInitResponse, error) {
	return api.UnwrapChecked[api.WebrtcInitResponse](
		w.Send(api.WebrtcInit, api.WebrtcInitRequest{Stateful: api.Stateful{Id: id}}))
}

func (w *Worker) WebrtcAnswer(id network.Uid, sdp string) {
	w.Notify(api.WebrtcAnswer, api.WebrtcAnswerRequest{Stateful: api.Stateful{Id: id}, Sdp: sdp})
}

func (w *Worker) WebrtcIceCandidate(id network.Uid, can string) {
	w.Notify(api.NewWebrtcIceCandidateRequest(id, can))
}

func (w *Worker) StartGame(id network.Uid, app games.AppMeta, req api.GameStartUserRequest) (*api.StartGameResponse, error) {
	return api.UnwrapChecked[api.StartGameResponse](w.Send(api.StartGame, api.StartGameRequest{
		StatefulRoom: api.StateRoom(id, req.RoomId),
		Game:         api.GameInfo{Name: app.Name, Base: app.Base, Path: app.Path, Type: app.Type},
		PlayerIndex:  req.PlayerIndex,
		Record:       req.Record,
		RecordUser:   req.RecordUser,
	}))
}

func (w *Worker) QuitGame(id network.Uid, roomId string) {
	w.Notify(api.QuitGame, api.GameQuitRequest{StatefulRoom: api.StateRoom(id, roomId)})
}

func (w *Worker) SaveGame(id network.Uid, roomId string) (*api.SaveGameResponse, error) {
	return api.UnwrapChecked[api.SaveGameResponse](
		w.Send(api.SaveGame, api.SaveGameRequest{StatefulRoom: api.StateRoom(id, roomId)}))
}

func (w *Worker) LoadGame(id network.Uid, roomId string) (*api.LoadGameResponse, error) {
	return api.UnwrapChecked[api.LoadGameResponse](
		w.Send(api.LoadGame, api.LoadGameRequest{StatefulRoom: api.StateRoom(id, roomId)}))
}

func (w *Worker) ChangePlayer(id network.Uid, roomId string, index int) (*api.ChangePlayerResponse, error) {
	return api.UnwrapChecked[api.ChangePlayerResponse](
		w.Send(api.ChangePlayer, api.ChangePlayerRequest{StatefulRoom: api.StateRoom(id, roomId), Index: index}))
}

func (w *Worker) ToggleMultitap(id network.Uid, roomId string) {
	_, _ = w.Send(api.ToggleMultitap, api.ToggleMultitapRequest{StatefulRoom: api.StateRoom(id, roomId)})
}

func (w *Worker) RecordGame(id network.Uid, roomId string, rec bool, recUser string) (*api.RecordGameResponse, error) {
	return api.UnwrapChecked[api.RecordGameResponse](
		w.Send(api.RecordGame, api.RecordGameRequest{StatefulRoom: api.StateRoom(id, roomId), Active: rec, User: recUser}))
}

func (w *Worker) TerminateSession(id network.Uid) {
	_, _ = w.Send(api.TerminateSession, api.TerminateSessionRequest{Stateful: api.Stateful{Id: id}})
}
