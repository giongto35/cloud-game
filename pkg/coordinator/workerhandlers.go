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

func (w *Worker) HandleRegisterRoom(rq api.RegisterRoomRequest, rooms *comm.NetMap[comm.NetClient]) {
	rooms.Put(rq, w)
}

func (w *Worker) HandleCloseRoom(rq api.CloseRoomRequest, rooms *comm.NetMap[comm.NetClient]) {
	rooms.RemoveByKey(rq)
}

func (w *Worker) HandleIceCandidate(rq api.WebrtcIceCandidateRequest, crowd *comm.NetMap[*User]) {
	if usr, err := crowd.Find(string(rq.Id)); err == nil {
		usr.SendWebrtcIceCandidate(rq.Candidate)
	} else {
		w.Log.Warn().Str("id", rq.Id.String()).Msg("unknown session")
	}
}
