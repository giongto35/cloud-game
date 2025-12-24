package coordinator

import "github.com/giongto35/cloud-game/v3/pkg/api"

func (w *Worker) WebrtcInit(id string) (*api.WebrtcInitResponse, error) {
	return api.UnwrapChecked[api.WebrtcInitResponse](
		w.Send(api.WebrtcInit, api.WebrtcInitRequest{Id: id}))
}

func (w *Worker) WebrtcAnswer(id string, sdp string) {
	w.Notify(api.WebrtcAnswer,
		api.WebrtcAnswerRequest{Stateful: api.Stateful{Id: id}, Sdp: sdp})
}

func (w *Worker) WebrtcIceCandidate(id string, candidate string) {
	w.Notify(api.WebrtcIce,
		api.WebrtcIceCandidateRequest{Stateful: api.Stateful{Id: id}, Candidate: candidate})
}

func (w *Worker) StartGame(id string, req api.GameStartUserRequest) (*api.StartGameResponse, error) {
	return api.UnwrapChecked[api.StartGameResponse](
		w.Send(api.StartGame, api.StartGameRequest{
			StatefulRoom: api.StatefulRoom{Id: id, Rid: req.RoomId},
			Game:         req.GameName,
			PlayerIndex:  req.PlayerIndex,
			Record:       req.Record,
			RecordUser:   req.RecordUser,
		}))
}

func (w *Worker) QuitGame(id string) {
	w.Notify(api.QuitGame, api.GameQuitRequest{Id: id, Rid: w.RoomId})
}

func (w *Worker) SaveGame(id string) (*api.SaveGameResponse, error) {
	return api.UnwrapChecked[api.SaveGameResponse](
		w.Send(api.SaveGame, api.SaveGameRequest{Id: id, Rid: w.RoomId}))
}

func (w *Worker) LoadGame(id string) (*api.LoadGameResponse, error) {
	return api.UnwrapChecked[api.LoadGameResponse](
		w.Send(api.LoadGame, api.LoadGameRequest{Id: id, Rid: w.RoomId}))
}

func (w *Worker) ChangePlayer(id string, index int) (*api.ChangePlayerResponse, error) {
	return api.UnwrapChecked[api.ChangePlayerResponse](
		w.Send(api.ChangePlayer, api.ChangePlayerRequest{
			StatefulRoom: api.StatefulRoom{Id: id, Rid: w.RoomId},
			Index:        index,
		}))
}

func (w *Worker) ResetGame(id string) {
	w.Notify(api.ResetGame, api.ResetGameRequest{Id: id, Rid: w.RoomId})
}

func (w *Worker) RecordGame(id string, rec bool, recUser string) (*api.RecordGameResponse, error) {
	return api.UnwrapChecked[api.RecordGameResponse](
		w.Send(api.RecordGame, api.RecordGameRequest{
			StatefulRoom: api.StatefulRoom{Id: id, Rid: w.RoomId},
			Active:       rec,
			User:         recUser,
		}))
}

func (w *Worker) TerminateSession(id string) {
	_, _ = w.Send(api.TerminateSession, api.TerminateSessionRequest{Id: id})
}
