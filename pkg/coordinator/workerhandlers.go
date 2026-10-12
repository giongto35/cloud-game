package coordinator

import "github.com/giongto35/cloud-game/v3/pkg/api"

func (w *Worker) HandleRegisterRoom(rq api.RegisterRoomRequest) {
	w.RoomId = string(rq)
	if w.OnRoomChange != nil {
		w.OnRoomChange("", false)
	}
}

func (w *Worker) HandleCloseRoom(rq api.CloseRoomRequest) {
	if string(rq) == w.RoomId {
		w.RoomId = ""
		w.FreeSlots()
	}
	if w.OnRoomChange != nil {
		w.OnRoomChange(string(rq), true)
	}
}

func (w *Worker) HandleIceCandidate(rq api.WebrtcSignalRequest, users HasUserRegistry) error {
	if usr := users.Find(rq.Id); usr != nil {
		usr.SendWebrtcIceCandidate(*rq.Ice)
	} else {
		w.log.Warn().Str("id", rq.Id).Msg("unknown session")
	}
	return nil
}

func (w *Worker) HandleLibGameList(inf api.LibGameListInfo) error {
	w.SetLib(inf.List)
	if w.OnLibrary != nil {
		w.OnLibrary(inf.List)
	}
	return nil
}

func (w *Worker) HandlePrevSessionList(sess api.PrevSessionInfo) error {
	if len(sess.List) == 0 {
		return nil
	}

	m := make(map[string]struct{})
	for _, v := range sess.List {
		m[v] = struct{}{}
	}
	w.SetSessions(m)
	return nil
}
