package api

type (
	GameInfo struct {
		Name string `json:"name"`
		Base string `json:"base"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	Room struct {
		Id string `json:"room_id"`
	}
)

type (
	ChangePlayerRequest = struct {
		Stateful
		Room
		Index string `json:"index"`
	}
	ChangePlayerResponse = string
	GameQuitRequest      struct {
		Stateful
		Room
	}
	LoadGameRequest struct {
		Stateful
		Room
	}
	LoadGameResponse = string
	SaveGameRequest  struct {
		Stateful
		Room
	}
	SaveGameResponse = string
	StartGameRequest struct {
		Stateful
		Room
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
		Stateful
		Room
		Active bool   `json:"active"`
		User   string `json:"user"`
	}
	RecordGameResponse      = string
	TerminateSessionRequest struct {
		Stateful
	}
	ToggleMultitapRequest struct {
		Stateful
		Room
	}
	WebrtcAnswerRequest struct {
		Stateful
		Sdp string `json:"sdp"`
	}
	WebrtcIceCandidateRequest struct {
		Stateful
		Candidate string `json:"candidate"`
	}
	WebrtcInitRequest struct {
		Stateful
	}
	WebrtcInitResponse = string
)
