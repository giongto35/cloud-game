package coordinator

import "github.com/giongto35/cloud-game/v3/pkg/api"

func (w *Worker) InitWebrtcStream(id string, initiator bool, sdp string) (*api.InitWebrtcStreamResponse, error) {
	return api.UnwrapChecked[api.InitWebrtcStreamResponse](
		w.Send(api.InitWebrtcStream, api.InitWebrtcStreamRequest{Id: id, Initiator: initiator, Sdp: sdp}))
}

func (w *Worker) WebrtcSignal(id string, sdp, ice *string) {
	w.Notify(api.WebrtcSignal, api.WebrtcSignalRequest{
		Stateful: api.Stateful{Id: id}, Ice: ice, Sdp: sdp,
	})
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

// StartSpecial starts a tournament match room or a replay playback room.
func (w *Worker) StartSpecial(id string, req api.GameStartUserRequest, t *api.TournamentRoom, replay []byte) (*api.StartGameResponse, error) {
	return api.UnwrapChecked[api.StartGameResponse](
		w.Send(api.StartGame, api.StartGameRequest{
			StatefulRoom: api.StatefulRoom{Id: id, Rid: req.RoomId},
			Game:         req.GameName,
			PlayerIndex:  req.PlayerIndex,
			Tournament:   t,
			Replay:       replay,
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

// TerminateSession drops the user session on the worker.
// The worker never answers this packet, so it is sent without waiting.
func (w *Worker) TerminateSession(id string) {
	w.Notify(api.TerminateSession, api.TerminateSessionRequest{Id: id})
}
