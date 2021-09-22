package api

type (
	ChangePlayerRequest = struct {
		StatefulRequest
		RoomId string `json:"room_id"`
		Index  string `json:"index"`
	}
	ChangePlayerResponse = string
	GameInfo             struct {
		Name string `json:"name"`
		Base string `json:"base"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	GameQuitRequest struct {
		StatefulRequest
		RoomId string `json:"room_id"`
	}
	LoadGameRequest struct {
		StatefulRequest
		RoomId string `json:"room_id"`
	}
	LoadGameResponse = string
	SaveGameRequest  struct {
		StatefulRequest
		RoomId string `json:"room_id"`
	}
	SaveGameResponse = string
	StartGameRequest struct {
		StatefulRequest
		Game        GameInfo `json:"game"`
		RoomId      string   `json:"room_id"`
		PlayerIndex int      `json:"player_index"`
	}
	StartGameResponse struct {
		RoomId string `json:"room_id"`
	}
	TerminateSessionRequest struct {
		StatefulRequest
	}
	ToggleMultitapRequest struct {
		StatefulRequest
		RoomId string `json:"room_id"`
	}
	WebrtcAnswerRequest struct {
		StatefulRequest
		Sdp string `json:"sdp"`
	}
	WebrtcIceCandidateRequest struct {
		StatefulRequest
		Candidate string `json:"candidate"`
	}
	WebrtcInitRequest struct {
		StatefulRequest
	}
	WebrtcInitResponse = string
)
