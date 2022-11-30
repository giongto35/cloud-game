package coordinator

import (
	"encoding/base64"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/comm"
)

func GetConnectionRequest(data string) (*api.ConnectionRequest, error) {
	if data == "" {
		return nil, fmt.Errorf("no data")
	}
	return api.UnwrapChecked[api.ConnectionRequest](base64.URLEncoding.DecodeString(data))
}

func (w *Worker) HandleRegisterRoom(rq api.RegisterRoomRequest, rooms *comm.NetMap) {
	rooms.Put(rq, w)
}

func (w *Worker) HandleCloseRoom(rq api.CloseRoomRequest, rooms *comm.NetMap) {
	rooms.RemoveByKey(rq)
}

func (w *Worker) HandleIceCandidate(rq api.WebrtcIceCandidateRequest, crowd *comm.NetMap) {
	if usr, err := crowd.Find(string(rq.Id)); err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(rq.Candidate)
	} else {
		w.Log.Warn().Str("id", rq.Id.String()).Msg("unknown session")
	}
}
