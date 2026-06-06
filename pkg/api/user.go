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
		Alias  string `json:"alias,omitempty"`
		Title  string `json:"title"`
		System string `json:"system"`
	}
	WebrtcSignalUser struct {
		Ice *string `json:"ice,omitempty"`
		Sdp *string `json:"sdp,omitempty"`
	}
	InitUserWebrtcStreamRequest struct {
		Initiator bool   `json:"initiator"`
		Sdp       string `json:"sdp,omitempty"`
	}
)
