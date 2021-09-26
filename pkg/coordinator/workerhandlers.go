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
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Logf("error: broken room register request %v", err)
		return
	}
	rooms.Put(req, w)
}

func (w *Worker) HandleCloseRoom(data json.RawMessage, rooms *client.NetMap) {
	var req api.CloseRoomRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Logf("error: broken room remove request %v", err)
		return
	}
	rooms.RemoveByKey(req)
}

func (w *Worker) HandleIceCandidate(data json.RawMessage, crowd *client.NetMap) {
	var req api.WebrtcIceCandidateRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Logf("error: broken ice candidate request %v", err)
		return
	}
	usr, err := crowd.Find(string(req.Id))
	if err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(req.Candidate)
	} else {
		w.Logf("error: unknown session: %v", req.Id)
	}
}
