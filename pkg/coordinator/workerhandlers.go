package coordinator

import (
	"encoding/base64"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
)

func GetConnectionRequest(data string) (*api.ConnectionRequest, error) {
	if data == "" {
		return nil, fmt.Errorf("no data")
	}
	return api.UnwrapChecked[api.ConnectionRequest](base64.URLEncoding.DecodeString(data))
}

func (w *Worker) HandleRegisterRoom(rq api.RegisterRoomRequest, rooms *client.NetMap) {
	rooms.Put(rq, w)
}

func (w *Worker) HandleCloseRoom(rq api.CloseRoomRequest, rooms *client.NetMap) {
	rooms.RemoveByKey(rq)
}

func (w *Worker) HandleIceCandidate(rq api.WebrtcIceCandidateRequest, crowd *client.NetMap) {
	if usr, err := crowd.Find(string(rq.Id)); err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(rq.Candidate)
	} else {
		w.log.Warn().Str("id", rq.Id.String()).Msg("unknown session")
	}
}
