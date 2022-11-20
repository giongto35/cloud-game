package worker

import (
	"encoding/base64"
	"encoding/json"

	"github.com/giongto35/cloud-game/v2/pkg/api"
)

func (c *Coordinator) webrtcInit(data []byte) (*api.WebrtcInitRequest, error) {
	return api.Unwrap[api.WebrtcInitRequest](data)
}

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
