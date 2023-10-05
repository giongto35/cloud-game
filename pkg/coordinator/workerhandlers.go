package coordinator

import (
	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
)

func (w *Worker) HandleRegisterRoom(rq api.RegisterRoomRequest) { w.RoomId = string(rq) }

func (w *Worker) HandleCloseRoom(rq api.CloseRoomRequest) {
	if string(rq) == w.RoomId {
		w.RoomId = ""
	}
}

func (w *Worker) HandleIceCandidate(rq api.WebrtcIceCandidateRequest[com.Uid], users HasUserRegistry) error {
	if usr := users.Find(rq.Id); usr != nil {
		usr.SendWebrtcIceCandidate(rq.Candidate)
	} else {
		w.log.Warn().Str("id", rq.Id.String()).Msg("unknown session")
	}
	return nil
}
