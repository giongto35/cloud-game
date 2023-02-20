package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/games"
)

func (w *Worker) WebrtcInit(id api.Uid) (*api.WebrtcInitResponse, error) {
	return com.UnwrapChecked[api.WebrtcInitResponse](
		w.Send(api.WebrtcInit, api.WebrtcInitRequest{Stateful: api.Stateful{Id: id}}))
}

func (w *Worker) WebrtcAnswer(id api.Uid, sdp string) {
	w.Notify(api.WebrtcAnswer, api.WebrtcAnswerRequest{Stateful: api.Stateful{Id: id}, Sdp: sdp})
}

func (w *Worker) WebrtcIceCandidate(id api.Uid, can string) {
	w.Notify(api.WebrtcIce, api.WebrtcIceCandidateRequest{Stateful: api.Stateful{Id: id}, Candidate: can})
}

func (w *Worker) StartGame(id api.Uid, app games.AppMeta, req api.GameStartUserRequest) (*api.StartGameResponse, error) {
	return com.UnwrapChecked[api.StartGameResponse](w.Send(api.StartGame, api.StartGameRequest{
		StatefulRoom: StateRoom(id, req.RoomId),
		Game:         api.GameInfo{Name: app.Name, Base: app.Base, Path: app.Path, Type: app.Type},
		PlayerIndex:  req.PlayerIndex,
		Record:       req.Record,
		RecordUser:   req.RecordUser,
	}))
}

func (w *Worker) QuitGame(id api.Uid) {
	w.Notify(api.QuitGame, api.GameQuitRequest{StatefulRoom: StateRoom(id, w.RoomId)})
}

func (w *Worker) SaveGame(id api.Uid) (*api.SaveGameResponse, error) {
	return com.UnwrapChecked[api.SaveGameResponse](
		w.Send(api.SaveGame, api.SaveGameRequest{StatefulRoom: StateRoom(id, w.RoomId)}))
}

func (w *Worker) LoadGame(id api.Uid) (*api.LoadGameResponse, error) {
	return com.UnwrapChecked[api.LoadGameResponse](
		w.Send(api.LoadGame, api.LoadGameRequest{StatefulRoom: StateRoom(id, w.RoomId)}))
}

func (w *Worker) ChangePlayer(id api.Uid, index int) (*api.ChangePlayerResponse, error) {
	return com.UnwrapChecked[api.ChangePlayerResponse](
		w.Send(api.ChangePlayer, api.ChangePlayerRequest{StatefulRoom: StateRoom(id, w.RoomId), Index: index}))
}

func (w *Worker) ToggleMultitap(id api.Uid) {
	_, _ = w.Send(api.ToggleMultitap, api.ToggleMultitapRequest{StatefulRoom: StateRoom(id, w.RoomId)})
}

func (w *Worker) RecordGame(id api.Uid, rec bool, recUser string) (*api.RecordGameResponse, error) {
	return com.UnwrapChecked[api.RecordGameResponse](
		w.Send(api.RecordGame, api.RecordGameRequest{StatefulRoom: StateRoom(id, w.RoomId), Active: rec, User: recUser}))
}

func (w *Worker) TerminateSession(id api.Uid) {
	_, _ = w.Send(api.TerminateSession, api.TerminateSessionRequest{Stateful: api.Stateful{Id: id}})
}

func StateRoom(id api.Uid, rid string) api.StatefulRoom {
	return api.StatefulRoom{Stateful: api.Stateful{Id: id}, Room: api.Room{Rid: rid}}
}
