// Represents API interface for bi-directional user connections.
//
// Prefixes in the names are such:
// in:  user ---> server
// out: user <--- server
//
// As example, DoThingOutRequest / DoThingOutResponse mean outgoing request
// and incoming response initiated by the server, so DoThingInX means requests
// and responses from the other side.

package user

import (
	"encoding/json"
	"errors"
	"log"
)

const (
	CheckLatency       uint8 = 3   // out
	InitSession        uint8 = 4   // out
	WebrtcInit         uint8 = 100 // in
	WebrtcOffer        uint8 = 101 // out
	WebrtcAnswer       uint8 = 102 // in
	WebrtcIceCandidate uint8 = 103 // in / out
	StartGame          uint8 = 104 // in / out
	ChangePlayer       uint8 = 108 // in / out
	QuitGame           uint8 = 105 // in
	SaveGame           uint8 = 106 // in
	LoadGame           uint8 = 107 // in
	ToggleMultitap     uint8 = 109 // in
)

var convertErr = errors.New("can't convert")

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
type GameQuitInRequest struct {
	RoomId string `json:"room_id"`
}
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
	data, err := u.Send(CheckLatency, req)
	if err != nil {
		log.Printf("can't get a response with latencies %v", err)
		return response, err
	}
	if data == nil {
		return response, convertErr
	}
	if v, ok := data.(string); ok {
		err := json.Unmarshal([]byte(v), &response)
		if err != nil {
			return response, err
		}
	}
	return response, nil
}

// InitSession (4) signals the user that the app is ready to go.
func (u *User) InitSession(req InitSessionOutRequest) {
	_, _ = u.SendAndForget(InitSession, req)
}

// SendWebrtcOffer (101) sends SDP offer back to the user.
func (u *User) SendWebrtcOffer(sdp string) {
	_, _ = u.SendAndForget(WebrtcOffer, sdp)
}

// SendWebrtcIceCandidate (103) sends remote ICE candidate back to the user.
func (u *User) SendWebrtcIceCandidate(candidate string) {
	_, _ = u.SendAndForget(WebrtcIceCandidate, candidate)
}

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame() error {
	_, err := u.SendAndForget(StartGame, u.RoomID)
	return err
}

// Notify unconditionally sends the result of some operation.
func (u *User) Notify(endpoint uint8, result interface{}) {
	_, _ = u.SendAndForget(endpoint, result)
}

func webrtcAnswerInRequest(data interface{}) (WebrtcAnswerInRequest, error) {
	v, ok := data.(WebrtcAnswerInRequest)
	if !ok {
		return v, convertErr
	}
	return v, nil
}

func webrtcIceCandidateInRequest(data interface{}) (WebrtcIceCandidateInRequest, error) {
	v, ok := data.(WebrtcIceCandidateInRequest)
	if !ok {
		return v, convertErr
	}
	return v, nil
}

func gameStartInRequest(data interface{}) (GameStartInRequest, error) {
	var r GameStartInRequest
	v, ok := data.(string)
	if !ok {
		return r, convertErr
	}
	err := json.Unmarshal([]byte(v), &r)
	if err != nil {
		return r, convertErr
	}
	return r, nil
}

func gameQuitInRequest(data interface{}) (GameQuitInRequest, error) {
	var r GameQuitInRequest
	v, ok := data.(string)
	if !ok {
		return r, convertErr
	}
	err := json.Unmarshal([]byte(v), &r)
	if err != nil {
		return r, convertErr
	}
	return r, nil
}

func changePlayerInRequest(data interface{}) (ChangePlayerInRequest, error) {
	v, ok := data.(ChangePlayerInRequest)
	if !ok {
		return v, convertErr
	}
	return v, nil
}
