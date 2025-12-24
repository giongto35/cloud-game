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
	WebrtcAnswerRequest     struct {
		Stateful
		Sdp string `json:"sdp"`
	}
	WebrtcIceCandidateRequest struct {
		Stateful
		Candidate string `json:"candidate"` // Base64-encoded ICE candidate
	}
	WebrtcInitRequest  Stateful
	WebrtcInitResponse string

	AppVideoInfo struct {
		W int     `json:"w"`
		H int     `json:"h"`
		S int     `json:"s"`
		A float32 `json:"a"`
	}

	LibGameListInfo struct {
		T    int
		List []GameInfo
	}

	PrevSessionInfo struct {
		List []string
	}
)
