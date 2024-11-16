package coordinator

import (
	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
)

func (w *Worker) WebrtcInit(id com.Uid) (*api.WebrtcInitResponse, error) {
	return api.UnwrapChecked[api.WebrtcInitResponse](
		w.Send(api.WebrtcInit, api.WebrtcInitRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: id}}))
}

func (w *Worker) WebrtcAnswer(id com.Uid, sdp string) {
	w.Notify(api.WebrtcAnswer, api.WebrtcAnswerRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: id}, Sdp: sdp})
}

func (w *Worker) WebrtcIceCandidate(id com.Uid, can string) {
	w.Notify(api.WebrtcIce, api.WebrtcIceCandidateRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: id}, Candidate: can})
}

func (w *Worker) StartGame(id com.Uid, req api.GameStartUserRequest) (*api.StartGameResponse, error) {
	return api.UnwrapChecked[api.StartGameResponse](
		w.Send(api.StartGame, api.StartGameRequest[com.Uid]{
			StatefulRoom: StateRoom(id, req.RoomId),
			Game:         req.GameName,
			PlayerIndex:  req.PlayerIndex,
			Record:       req.Record,
			RecordUser:   req.RecordUser,
		}))
}

func (w *Worker) QuitGame(id com.Uid) {
	w.Notify(api.QuitGame, api.GameQuitRequest[com.Uid]{StatefulRoom: StateRoom(id, w.RoomId)})
}

func (w *Worker) SaveGame(id com.Uid) (*api.SaveGameResponse, error) {
	return api.UnwrapChecked[api.SaveGameResponse](
		w.Send(api.SaveGame, api.SaveGameRequest[com.Uid]{StatefulRoom: StateRoom(id, w.RoomId)}))
}

func (w *Worker) LoadGame(id com.Uid) (*api.LoadGameResponse, error) {
	return api.UnwrapChecked[api.LoadGameResponse](
		w.Send(api.LoadGame, api.LoadGameRequest[com.Uid]{StatefulRoom: StateRoom(id, w.RoomId)}))
}

func (w *Worker) ChangePlayer(id com.Uid, index int) (*api.ChangePlayerResponse, error) {
	return api.UnwrapChecked[api.ChangePlayerResponse](
		w.Send(api.ChangePlayer, api.ChangePlayerRequest[com.Uid]{StatefulRoom: StateRoom(id, w.RoomId), Index: index}))
}

func (w *Worker) RecordGame(id com.Uid, rec bool, recUser string) (*api.RecordGameResponse, error) {
	return api.UnwrapChecked[api.RecordGameResponse](
		w.Send(api.RecordGame, api.RecordGameRequest[com.Uid]{StatefulRoom: StateRoom(id, w.RoomId), Active: rec, User: recUser}))
}

func (w *Worker) TerminateSession(id com.Uid) {
	_, _ = w.Send(api.TerminateSession, api.TerminateSessionRequest[com.Uid]{Stateful: api.Stateful[com.Uid]{Id: id}})
}

func StateRoom[T api.Id](id T, rid string) api.StatefulRoom[T] {
	return api.StatefulRoom[T]{Stateful: api.Stateful[T]{Id: id}, Room: api.Room{Rid: rid}}
}
