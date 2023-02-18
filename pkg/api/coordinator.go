package api

import (
	"encoding/base64"
	"fmt"

	"github.com/giongto35/cloud-game/v2/pkg/network"
)

type (
	CloseRoomRequest  string
	ConnectionRequest struct {
		Addr    string `json:"addr,omitempty"`
		Id      string `json:"id,omitempty"`
		IsHTTPS bool   `json:"is_https,omitempty"`
		PingURL string `json:"ping_url,omitempty"`
		Port    string `json:"port,omitempty"`
		Tag     string `json:"tag,omitempty"`
		Zone    string `json:"zone,omitempty"`
	}
	GetWorkerListRequest  struct{}
	GetWorkerListResponse struct {
		Servers []Server `json:"servers"`
	}
	RegisterRoomRequest string
)

const (
	DataQueryParam   = "data"
	RoomIdQueryParam = "room_id"
	ZoneQueryParam   = "zone"
	WorkerIdParam    = "wid"
)

func RequestToHandshake(data string) (*ConnectionRequest, error) {
	if data == "" {
		return nil, ErrMalformed
	}
	handshake, err := UnwrapChecked[ConnectionRequest](base64.URLEncoding.DecodeString(data))
	if err != nil || handshake == nil {
		return nil, fmt.Errorf("%v (%v)", err, handshake)
	}
	return handshake, nil
}

func (c ConnectionRequest) HasUID() (bool, network.Uid) {
	hid := network.Uid(c.Id)
	if !(c.Id == "" || !network.ValidUid(hid)) {
		return true, hid
	}
	return false, ""
}

// Server contains a list of server groups.
// Server is a separate machine that may contain
// multiple sub-processes.
type Server struct {
	Addr     string `json:"addr,omitempty"`
	Id       string `json:"id,omitempty"`
	IsBusy   bool   `json:"is_busy,omitempty"`
	InGroup  bool   `json:"in_group,omitempty"`
	Machine  string `json:"machine,omitempty"`
	PingURL  string `json:"ping_url"`
	Port     string `json:"port,omitempty"`
	Replicas uint32 `json:"replicas,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Zone     string `json:"zone,omitempty"`
}

type HasServerInfo interface {
	GetServerList() []Server
}
