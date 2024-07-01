package api

type (
	ChangePlayerUserRequest  int
	CheckLatencyUserResponse []string
	CheckLatencyUserRequest  map[string]int64
	GameStartUserRequest     struct {
		GameName    string `json:"game_name"`
		RoomId      string `json:"room_id"`
		Record      bool   `json:"record,omitempty"`
		RecordUser  string `json:"record_user,omitempty"`
		PlayerIndex int    `json:"player_index"`
	}
	GameStartUserResponse struct {
		RoomId  string        `json:"roomId"`
		Av      *AppVideoInfo `json:"av"`
		KbMouse bool          `json:"kb_mouse"`
	}
	IceServer struct {
		Urls       string `json:"urls,omitempty"`
		Username   string `json:"username,omitempty"`
		Credential string `json:"credential,omitempty"`
	}
	InitSessionUserResponse struct {
		Ice   []IceServer `json:"ice"`
		Games []AppMeta   `json:"games"`
		Wid   string      `json:"wid"`
	}
	AppMeta struct {
		Title  string `json:"title"`
		System string `json:"system"`
	}
	WebrtcAnswerUserRequest string
	WebrtcUserIceCandidate  string
)
