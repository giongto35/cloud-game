package coordinator

import (
	"encoding/base64"
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/cache"
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

func (w *Worker) HandleRegisterRoom(data json.RawMessage, rooms *cache.Cache) {
	var req api.RegisterRoomRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Printf("error: broken room register request %v", err)
		return
	}
	rooms.Add(req, w)
}

func (w *Worker) HandleCloseRoom(data json.RawMessage, rooms *cache.Cache) {
	var req api.CloseRoomRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Printf("error: broken room remove request %v", err)
		return
	}
	rooms.Remove(req)
}

func (w *Worker) HandleIceCandidate(data json.RawMessage, crowd *cache.Cache) {
	var req api.WebrtcIceCandidateRequest
	err := json.Unmarshal(data, &req)
	if err != nil {
		w.Printf("error: broken ice candidate request %v", err)
		return
	}
	usr, err := crowd.Find(string(req.Id))
	if err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(req.Candidate)
	} else {
		w.Printf("error: unknown session: %v", req.Id)
	}
}
