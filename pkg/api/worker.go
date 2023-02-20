package api

type GameInfo struct {
	Name string `json:"name"`
	Base string `json:"base"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type (
	ChangePlayerRequest = struct {
		StatefulRoom
		Index int `json:"index"`
	}
	ChangePlayerResponse int
	GameQuitRequest      struct {
		StatefulRoom
	}
	LoadGameRequest struct {
		StatefulRoom
	}
	LoadGameResponse string
	SaveGameRequest  struct {
		StatefulRoom
	}
	SaveGameResponse string
	StartGameRequest struct {
		StatefulRoom
		Record      bool
		RecordUser  string
		Game        GameInfo `json:"game"`
		PlayerIndex int      `json:"player_index"`
	}
	StartGameResponse struct {
		Room
		Record bool
	}
	RecordGameRequest struct {
		StatefulRoom
		Active bool   `json:"active"`
		User   string `json:"user"`
	}
	RecordGameResponse      string
	TerminateSessionRequest struct {
		Stateful
	}
	ToggleMultitapRequest struct {
		StatefulRoom
	}
	WebrtcAnswerRequest struct {
		Stateful
		Sdp string `json:"sdp"`
	}
	WebrtcIceCandidateRequest struct {
		Stateful
		Candidate string `json:"candidate"` // Base64-encoded ICE candidate
	}
	WebrtcInitRequest struct {
		Stateful
	}
	WebrtcInitResponse string
)

func NewWebrtcIceCandidateRequest(id Uid, can string) (PT, any) {
	return WebrtcIce, WebrtcIceCandidateRequest{Stateful: Stateful{id}, Candidate: can}
}
