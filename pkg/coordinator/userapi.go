package coordinator

import (
	"encoding/json"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
)

// CheckLatency sends a list of server addresses to the user
// and waits get back this list with tested ping times for each server.
func (u *User) CheckLatency(req api.CheckLatencyUserResponse) (api.CheckLatencyUserRequest, error) {
	u.Printf("servers to ping: %v", req)
	var response api.CheckLatencyUserRequest
	data, err := u.Send(api.CheckLatency, req)
	if err != nil || data == nil {
		return response, err
	}
	return response, json.Unmarshal(data, &response)
}

// InitSession signals the user that the app is ready to go.
func (u *User) InitSession(ice []webrtc.IceServer, games []string) {
	_ = u.SendAndForget(api.InitSession, api.InitSessionUserResponse{
		// don't do this at home
		Ice:   *(*[]api.IceServer)(unsafe.Pointer(&ice)),
		Games: games,
	})
}

// SendWebrtcOffer sends SDP offer back to the user.
func (u *User) SendWebrtcOffer(sdp string) { _ = u.SendAndForget(api.WebrtcOffer, sdp) }

// SendWebrtcIceCandidate sends remote ICE candidate back to the user.
func (u *User) SendWebrtcIceCandidate(candidate string) {
	_ = u.SendAndForget(api.WebrtcIceCandidate, candidate)
}

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame() error { return u.SendAndForget(api.StartGame, u.RoomID) }

// Notify unconditionally sends the result of some operation.
func (u *User) Notify(endpoint uint8, result interface{}) { _ = u.SendAndForget(endpoint, result) }
