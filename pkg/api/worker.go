package api

type (
	ChangePlayerRequest[T Id] struct {
		StatefulRoom[T]
		Index int `json:"index"`
	}
	ChangePlayerResponse  int
	GameQuitRequest[T Id] struct {
		StatefulRoom[T]
	}
	LoadGameRequest[T Id] struct {
		StatefulRoom[T]
	}
	LoadGameResponse      string
	SaveGameRequest[T Id] struct {
		StatefulRoom[T]
	}
	SaveGameResponse       string
	StartGameRequest[T Id] struct {
		StatefulRoom[T]
		Record      bool
		RecordUser  string
		Game        GameInfo `json:"game"`
		PlayerIndex int      `json:"player_index"`
	}
	GameInfo struct {
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
	RecordGameRequest[T Id] struct {
		StatefulRoom[T]
		Active bool   `json:"active"`
		User   string `json:"user"`
	}
	RecordGameResponse            string
	TerminateSessionRequest[T Id] struct {
		Stateful[T]
	}
	WebrtcAnswerRequest[T Id] struct {
		Stateful[T]
		Sdp string `json:"sdp"`
	}
	WebrtcIceCandidateRequest[T Id] struct {
		Stateful[T]
		Candidate string `json:"candidate"` // Base64-encoded ICE candidate
	}
	WebrtcInitRequest[T Id] struct {
		Stateful[T]
	}
	WebrtcInitResponse string

	AppVideoInfo struct {
		W int     `json:"w"`
		H int     `json:"h"`
		S int     `json:"s"`
		A float32 `json:"a"`
	}
)
