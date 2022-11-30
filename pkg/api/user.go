package api

type (
	ChangePlayerUserRequest  = int
	CheckLatencyUserResponse []string
	CheckLatencyUserRequest  map[string]int64
	GameStartUserRequest     struct {
		GameName    string `json:"game_name"`
		RoomId      string `json:"room_id"`
		Record      bool   `json:"record,omitempty"`
		RecordUser  string `json:"record_user,omitempty"`
		PlayerIndex int    `json:"player_index"`
	}
	IceServer struct {
		Urls       string `json:"urls,omitempty"`
		Username   string `json:"username,omitempty"`
		Credential string `json:"credential,omitempty"`
	}
	InitSessionUserResponse struct {
		Ice   []IceServer `json:"ice"`
		Games []string    `json:"games"`
		Wid   string      `json:"wid"`
	}
	WebrtcAnswerUserRequest = string
	WebrtcUserIceCandidate  = string
)

func InitSessionResult(ice []IceServer, games []string, wid string) (uint8, InitSessionUserResponse) {
	return InitSession, InitSessionUserResponse{Ice: ice, Games: games, Wid: wid}
}
