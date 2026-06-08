package coordinator

import (
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/config"
)

// CheckLatency sends a list of server addresses to the user
// and waits get back this list with tested ping times for each server.
func (u *User) CheckLatency(req api.CheckLatencyUserResponse) (api.CheckLatencyUserRequest, error) {
	dat, err := api.UnwrapChecked[api.CheckLatencyUserRequest](u.Send(api.CheckLatency, req))
	if dat == nil {
		return api.CheckLatencyUserRequest{}, err
	}
	return *dat, nil
}

// InitSession signals the user that the app is ready to go.
func (u *User) InitSession(wid string, ice []config.IceServer, games []api.AppMeta) {
	u.Notify(api.InitSession, api.InitSessionUserResponse{
		Ice:   *(*[]api.IceServer)(unsafe.Pointer(&ice)), // don't do this at home
		Games: games,
		Wid:   wid,
	})
}

// SendSignal sends webrtc signal back to the user.
func (u *User) SendSignal(sdp *string, ice *string) {
	u.Notify(api.WebrtcSignal, api.WebrtcSignalUser{Sdp: sdp, Ice: ice})
}

// StartGame signals the user that everything is ready to start a game.
func (u *User) StartGame(av *api.AppVideoInfo, kbMouse bool) {
	u.Notify(api.StartGame, api.GameStartUserResponse{RoomId: u.w.RoomId, Av: av, KbMouse: kbMouse})
}
