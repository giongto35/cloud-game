package api

type (
	ChangePlayerUserRequest  = string
	CheckLatencyUserResponse []string
	CheckLatencyUserRequest  map[string]int64
	GameStartUserRequest     struct {
		GameName    string `json:"game_name"`
		RoomId      string `json:"room_id"`
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
	}
	WebrtcAnswerUserRequest       = string
	WebrtcIceCandidateUserRequest = string
)
