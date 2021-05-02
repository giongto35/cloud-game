package coordinator

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/cache"
)

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
	w.Printf("SESSIONS: %v", crowd.List())
	usr, err := crowd.Find(string(req.Id))
	if err == nil {
		u := usr.(*User)
		u.SendWebrtcIceCandidate(req.Candidate)
	} else {
		w.Printf("error: unknown SessionID:", req.Id)
	}
}
