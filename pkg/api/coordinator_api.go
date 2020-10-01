package api

import (
	"encoding/json"
)

// This list of postfixes is used in the API:
// - *Request postfix denotes clients calls (i.e. from a browser to the HTTP-server).
// - *Call postfix denotes IPC calls (from the coordinator to a worker).

type GameStartRequest struct {
	GameName string `json:"game_name"`
	IsMobile bool   `json:"is_mobile"`
}

func (packet *GameStartRequest) From(data string) error { return from(packet, data) }

type GameStartCall struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

func (packet *GameStartCall) From(data string) error { return from(packet, data) }
func (packet *GameStartCall) To() (string, error)    { return to(packet) }

func from(source interface{}, data string) error {
	err := json.Unmarshal([]byte(data), source)
	if err != nil {
		return err
	}
	return nil
}

func to(target interface{}) (string, error) {
	b, err := json.Marshal(target)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
