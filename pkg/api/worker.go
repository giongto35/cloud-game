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
		Name string `json:"name"`
		Base string `json:"base"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	StartGameResponse struct {
		Room
		Record bool
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
	ToggleMultitapRequest[T Id] struct {
		StatefulRoom[T]
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
)
