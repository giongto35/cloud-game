package coordinator

import (
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
)

// CheckLatency sends a list of server addresses to the user
// and waits get back this list with tested ping times for each server.
func (u *User) CheckLatency(req api.CheckLatencyUserResponse) (api.CheckLatencyUserRequest, error) {
	data, err := u.Send(api.CheckLatency, req)
	if err != nil || data == nil {
		return nil, err
	}
	rs, err := api.Unwrap[api.CheckLatencyUserRequest](data)
	if err != nil {
		return api.CheckLatencyUserRequest{}, err
	}
	return *rs, err
}

// InitSession signals the user that the app is ready to go.
func (u *User) InitSession(wid string, ice []webrtc.IceServer, games []string) {
	// don't do this at home
	u.Notify(api.InitSessionResult(*(*[]api.IceServer)(unsafe.Pointer(&ice)), games, wid))
}

// SendWebrtcOffer sends SDP offer back to the user.
func (u *User) SendWebrtcOffer(sdp string) { u.Notify(api.WebrtcOffer, sdp) }

// SendWebrtcIceCandidate sends remote ICE candidate back to the user.
func (u *User) SendWebrtcIceCandidate(candidate string) { u.Notify(api.WebrtcIceCandidate, candidate) }

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame() error { return u.SendAndForget(api.StartGame, u.RoomID) }
