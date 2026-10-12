package api

type (
	ChangePlayerUserRequest  int
	CheckLatencyUserResponse []string
	CheckLatencyUserRequest  map[string]int64
	GameStartUserRequest     struct {
		GameName    string `json:"game_name"`
		RoomId      string `json:"room_id"`
		RecordUser  string `json:"record_user,omitempty"`
		PlayerIndex int    `json:"player_index"`
		Record      bool   `json:"record,omitempty"`
		// Spectator joins an existing room without controls.
		Spectator bool `json:"spectator,omitempty"`
		// MatchId starts or joins a tournament match room.
		MatchId int64 `json:"match_id,omitempty"`
		// ReplayId starts or joins the playback of a recorded match.
		ReplayId int64 `json:"replay_id,omitempty"`
	}
	GameStartUserResponse struct {
		Av      *AppVideoInfo `json:"av"`
		RoomId  string        `json:"roomId"`
		KbMouse bool          `json:"kb_mouse"`
		// Slot is the player port (0-3) or -1 for spectators.
		Slot int  `json:"slot"`
		Host bool `json:"host"`
	}
	IceServer struct {
		Urls       string `json:"urls,omitempty"`
		Username   string `json:"username,omitempty"`
		Credential string `json:"credential,omitempty"`
	}
	InitSessionUserResponse struct {
		Wid   string      `json:"wid"`
		Ice   []IceServer `json:"ice"`
		Games []AppMeta   `json:"games"`
		You   *UserInfo   `json:"you,omitempty"`
		// Busy is true when the user has no server (all servers are busy).
		Busy bool `json:"busy,omitempty"`
	}
	UserInfo struct {
		Id      int64    `json:"id"`
		Name    string   `json:"name"`
		Email   string   `json:"email"`
		Role    string   `json:"role"`
		Perms   []string `json:"perms"`
		Primary bool     `json:"primary,omitempty"`
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
		Sdp       string `json:"sdp,omitempty"`
		Initiator bool   `json:"initiator"`
	}
)

// Lobby, queue and room types.
type (
	LobbyStateResponse struct {
		Rooms   []LobbyRoom  `json:"rooms"`
		Workers LobbyWorkers `json:"workers"`
		Queue   LobbyQueue   `json:"queue"`
	}
	LobbyRoom struct {
		Id         string        `json:"id"`
		Game       string        `json:"game"`
		System     string        `json:"system"`
		Host       string        `json:"host"`
		Players    []LobbyMember `json:"players"`
		Spectators int           `json:"spectators"`
		SlotsFree  int           `json:"slots_free"`
		Since      int64         `json:"since"`
	}
	LobbyMember struct {
		Name string `json:"name"`
		Slot int    `json:"slot"`
	}
	LobbyWorkers struct {
		Total int `json:"total"`
		Free  int `json:"free"`
	}
	LobbyQueue struct {
		Size int `json:"size"`
	}
	QueueJoinRequest struct {
		Game string `json:"game"`
	}
	QueueStateResponse struct {
		// Status is one of: none, waiting, called, expired.
		Status    string `json:"status"`
		Position  int    `json:"position,omitempty"`
		Size      int    `json:"size,omitempty"`
		Game      string `json:"game,omitempty"`
		ExpiresIn int    `json:"expires_in,omitempty"`
	}
	RoomStateResponse struct {
		RoomId  string       `json:"room_id"`
		Game    string       `json:"game"`
		Members []RoomMember `json:"members"`
	}
	RoomMember struct {
		Id      string `json:"id"`
		Account int64  `json:"account"`
		Name    string `json:"name"`
		Slot    int    `json:"slot"`
		Host    bool   `json:"host"`
		You     bool   `json:"you,omitempty"`
	}
	ChatSendRequest struct {
		Text string `json:"text"`
	}
	ChatMessageResponse struct {
		From   string `json:"from,omitempty"`
		Name   string `json:"name,omitempty"`
		Text   string `json:"text"`
		Ts     int64  `json:"ts"`
		System bool   `json:"system,omitempty"`
	}
	RoomKickRequest struct {
		User string `json:"user"`
	}
	RoomAssignRequest struct {
		User string `json:"user"`
		Slot int    `json:"slot"`
	}
	KickedResponse struct {
		Reason string `json:"reason"`
	}
	NoticeResponse struct {
		Kind  string `json:"kind"`
		Text  string `json:"text,omitempty"`
		Count int    `json:"count,omitempty"`
		Match int64  `json:"match,omitempty"`
	}
)
