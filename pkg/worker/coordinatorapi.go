package worker

import (
	"encoding/json"
	"github.com/giongto35/cloud-game/v2/pkg/api"
)

type IdentifyWorkerInRequest = string
type TerminateSessionInRequest struct {
	api.StatefulRequest
}
type WebrtcInitInRequest struct {
	api.StatefulRequest
}
type WebrtcInitInResponse = string
type WebrtcAnswerInRequest struct {

}

// !to do nil check
func (c *Coordinator) identifyWorkerInRequest(data json.RawMessage) (IdentifyWorkerInRequest, error) {
	if data == nil {
		return "", nil
	}
	return IdentifyWorkerInRequest(data), nil
}

func (c *Coordinator) terminateSession(data json.RawMessage) (TerminateSessionInRequest, error) {
	var v TerminateSessionInRequest
	err := json.Unmarshal(data, &v)
	return v, err
}

func (c *Coordinator) webrtcInit(data json.RawMessage) (WebrtcInitInRequest, error) {
	var v WebrtcInitInRequest
	err := json.Unmarshal(data, &v)
	return v, err
}

func fromJson(data json.RawMessage, value interface{}) error {
	return json.Unmarshal(data, &value)
}
