// Represents API interface for bi-directional user connections.
//
// Prefixes in the names are such:
// in:  user ---> server
// out: user <--- server
//
// As example, DoThingOutRequest / DoThingOutResponse mean outgoing request
// and incoming response initiated by the server, so DoThingInX means requests
// and responses from the other side.

package coordinator

import (
	"encoding/json"
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/api"
)

type CheckLatencyOutRequest []string
type CheckLatencyOutResponse map[string]int64
type InitSessionOutRequest struct {
	Ice   []IceServer `json:"ice"`
	Games []string    `json:"games"`
}
type IceServer struct {
	Urls       string `json:"urls,omitempty"`
	Username   string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}
type WebrtcAnswerInRequest = string
type WebrtcIceCandidateInRequest = string

type ChangePlayerInRequest = string
type GameStartInRequest struct {
	GameName    string `json:"game_name"`
	RoomId      string `json:"room_id"`
	PlayerIndex int    `json:"player_index"`
}

// CheckLatency (3) sends a list of server addresses to the user
// and waits get back this list with tested ping times for each server.
func (u *User) CheckLatency(req CheckLatencyOutRequest) (CheckLatencyOutResponse, error) {
	var response CheckLatencyOutResponse
	u.Printf("Ping addresses: %v", req)
	data, err := u.Send(api.CheckLatency, req)
	if err != nil {
		log.Printf("can't get a response with latencies %v", err)
		return response, err
	}
	if data == nil {
		return response, api.ConvertErr
	}
	if v, ok := data.([]byte); ok {
		err := json.Unmarshal(v, &response)
		if err != nil {
			return response, err
		}
	}
	return response, nil
}

// InitSession (4) signals the user that the app is ready to go.
func (u *User) InitSession(req InitSessionOutRequest) {
	_ = u.SendAndForget(api.InitSession, req)
}

// SendWebrtcOffer (101) sends SDP offer back to the user.
func (u *User) SendWebrtcOffer(sdp string) {
	_ = u.SendAndForget(api.WebrtcOffer, sdp)
}

// SendWebrtcIceCandidate (103) sends remote ICE candidate back to the user.
func (u *User) SendWebrtcIceCandidate(candidate string) {
	_ = u.SendAndForget(api.WebrtcIceCandidate, candidate)
}

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame() error {
	return u.SendAndForget(api.StartGame, u.RoomID)
}

// Notify unconditionally sends the result of some operation.
func (u *User) Notify(endpoint uint8, result interface{}) {
	_ = u.SendAndForget(endpoint, result)
}

func webrtcAnswerInRequest(data json.RawMessage) (WebrtcAnswerInRequest, error) {
	var req WebrtcAnswerInRequest
	err := json.Unmarshal(data, &req)
	return req, err
}

func webrtcIceCandidateInRequest(data json.RawMessage) (WebrtcIceCandidateInRequest, error) {
	var req WebrtcIceCandidateInRequest
	err := json.Unmarshal(data, &req)
	return req, err
}

func gameStartInRequest(data json.RawMessage) (GameStartInRequest, error) {
	var req GameStartInRequest
	err := json.Unmarshal(data, &req)
	return req, err
}

func gameQuitInRequest(data json.RawMessage) (api.GameQuitRequest, error) {
	var req api.GameQuitRequest
	err := json.Unmarshal(data, &req)
	return req, err
}

func changePlayerInRequest(data json.RawMessage) (ChangePlayerInRequest, error) {
	var req ChangePlayerInRequest
	err := json.Unmarshal(data, &req)
	return req, err
}
