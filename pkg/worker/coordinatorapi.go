package worker

import (
	"encoding/base64"
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

func fromJson(data json.RawMessage, value interface{}) error { return json.Unmarshal(data, &value) }

// toBase64Json encodes data to a URL-encoded Base64+JSON string.
func toBase64Json(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// fromBase64Json decodes data from a URL-encoded Base64+JSON string.
func fromBase64Json(data string, obj interface{}) error {
	b, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
}
