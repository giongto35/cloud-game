package worker

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
)

func (c *Coordinator) terminateSession(data json.RawMessage) (api.TerminateSessionRequest, error) {
	var v api.TerminateSessionRequest
	err := json.Unmarshal(data, &v)
	return v, err
}

func (c *Coordinator) webrtcInit(data json.RawMessage) (api.WebrtcInitRequest, error) {
	var v api.WebrtcInitRequest
	err := json.Unmarshal(data, &v)
	return v, err
}

func fromJson(data json.RawMessage, value interface{}) error {
	return json.Unmarshal(data, &value)
}
