package coordinator

import (
	"encoding/base64"
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
)

func GetConnectionRequest(data string) (api.ConnectionRequest, error) {
	req := api.ConnectionRequest{}
	if data == "" {
		return req, nil
	}
	decodeString, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return req, err
	}
	err = json.Unmarshal(decodeString, &req)
	return req, err
}

func (w *Worker) HandleRegisterRoom(data json.RawMessage, rooms *client.NetMap) {
	var req api.RegisterRoomRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.log.Error().Err(err).Msg("malformed room register request")
		return
	}
	rooms.Put(req, w)
}

func (w *Worker) HandleCloseRoom(data json.RawMessage, rooms *client.NetMap) {
	var req api.CloseRoomRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.log.Error().Err(err).Msg("malformed room remove request")
		return
	}
	rooms.RemoveByKey(req)
}

func (w *Worker) HandleIceCandidate(data json.RawMessage, crowd *client.NetMap) {
	var req api.WebrtcIceCandidateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.log.Error().Err(err).Msg("malformed Ice candidate request")
		return
	}
	usr, err := crowd.Find(string(req.Id))
	if err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(req.Candidate)
	} else {
		w.log.Warn().Str("id", req.Id.String()).Msg("unknown session")
	}
}
