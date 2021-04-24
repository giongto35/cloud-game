package api

import (
	"encoding/json"
	"errors"
	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
)

type InitPack struct {
	Ice   []webrtc.IceServer `json:"ice"`
	Games []string           `json:"games"`
}

var convertErr = errors.New("can't convert")

type Latencies map[string]int64

func (l *Latencies) FromResponse(response interface{}) error {
	if response == nil {
		return convertErr
	}
	if v, ok := response.(string); ok {
		return json.Unmarshal([]byte(v), &l)
	}
	return convertErr
}
