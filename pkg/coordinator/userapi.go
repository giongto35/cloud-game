package coordinator

import (
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/config/webrtc"
)

// CheckLatency sends a list of server addresses to the user
// and waits get back this list with tested ping times for each server.
func (u *User) CheckLatency(req api.CheckLatencyUserResponse) (api.CheckLatencyUserRequest, error) {
	data, err := u.Send(api.CheckLatency, req)
	if err != nil || data == nil {
		return nil, err
	}
	dat := api.Unwrap[api.CheckLatencyUserRequest](data)
	if dat == nil {
		return api.CheckLatencyUserRequest{}, err
	}
	return *dat, err
}

// InitSession signals the user that the app is ready to go.
func (u *User) InitSession(wid string, ice []webrtc.IceServer, games []string) {
	u.Notify(api.InitSession, api.InitSessionUserResponse{
		// don't do this at home
		Ice:   *(*[]api.IceServer)(unsafe.Pointer(&ice)),
		Games: games,
		Wid:   wid,
	})
}

// SendWebrtcOffer sends SDP offer back to the user.
func (u *User) SendWebrtcOffer(sdp string) { u.Notify(api.WebrtcOffer, sdp) }

// SendWebrtcIceCandidate sends remote ICE candidate back to the user.
func (u *User) SendWebrtcIceCandidate(candidate string) { u.Notify(api.WebrtcIce, candidate) }

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame() { u.Notify(api.StartGame, u.w.RoomId) }
