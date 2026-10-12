package api

type (
	ChangePlayerRequest struct {
		StatefulRoom
		Index int `json:"index"`
	}
	ChangePlayerResponse int
	GameQuitRequest      StatefulRoom
	LoadGameRequest      StatefulRoom
	LoadGameResponse     string
	ResetGameRequest     StatefulRoom
	ResetGameResponse    string
	SaveGameRequest      StatefulRoom
	SaveGameResponse     string
	StartGameRequest     struct {
		StatefulRoom
		Record      bool
		RecordUser  string
		Game        string `json:"game"`
		PlayerIndex int    `json:"player_index"`
		// Tournament makes a recorded match room with automatic result detection.
		Tournament *TournamentRoom `json:"tournament,omitempty"`
		// Replay makes a playback room for the given recorded match.
		Replay []byte `json:"replay,omitempty"`
	}
	TournamentRoom struct {
		Match int64  `json:"match"`
		Rule  string `json:"rule"`
	}
	// MatchResultRequest is sent by a worker when a tournament match is over.
	MatchResultRequest struct {
		Room   string `json:"room"`
		Match  int64  `json:"match"`
		Winner int    `json:"winner"` // 0 = player 1, 1 = player 2
		Score  [2]int `json:"score"`
		Frame  uint64 `json:"frame"`
		Replay []byte `json:"replay,omitempty"`
	}
	GameInfo struct {
		Alias  string `json:"alias"`
		Base   string `json:"base"`
		Name   string `json:"name"`
		Path   string `json:"path"`
		System string `json:"system"`
		Type   string `json:"type"`
	}
	StartGameResponse struct {
		Room
		AV      *AppVideoInfo `json:"av"`
		Record  bool          `json:"record"`
		KbMouse bool          `json:"kb_mouse"`
	}
	RecordGameRequest struct {
		StatefulRoom
		Active bool   `json:"active"`
		User   string `json:"user"`
	}
	RecordGameResponse      string
	TerminateSessionRequest Stateful
	WebrtcSignalRequest     struct {
		Stateful
		Sdp *string `json:"sdp,omitempty"`
		Ice *string `json:"ice,omitempty"`
	}
	InitWebrtcStreamRequest struct {
		// Stateful
		Id        string `json:"id"`
		Initiator bool   `json:"initiator"`
		Sdp       string `json:"sdp,omitempty"`
	}
	InitWebrtcStreamResponse string

	AppVideoInfo struct {
		W    int     `json:"w"`
		H    int     `json:"h"`
		A    float32 `json:"a"`
		S    int     `json:"s"`
		SM   string  `json:"sm,omitempty"`
		Flip bool    `json:"flip"`
		Rot  uint    `json:"rot"`
	}

	LibGameListInfo struct {
		T    int
		List []GameInfo
	}

	PrevSessionInfo struct {
		List []string
	}
)
