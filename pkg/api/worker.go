package api

type WebrtcAnswerRequest struct {
	StatefulRequest
	Sdp string `json:"sdp"`
}

type WebrtcIceCandidateRequest struct {
	StatefulRequest
	Candidate string `json:"candidate"`
}

type StartGameRequest struct {
	StatefulRequest
	Game        GameInfo `json:"game"`
	RoomId      string   `json:"room_id"`
	PlayerIndex int      `json:"player_index"`
}
type GameInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}
type StartGameResponse struct {
	RoomId string `json:"room_id"`
}
type GameQuitRequest struct {
	StatefulRequest
	RoomId string `json:"room_id"`
}

type SaveGameRequest struct {
	StatefulRequest
	RoomId string `json:"room_id"`
}
type SaveGameResponse = string
type LoadGameRequest struct {
	StatefulRequest
	RoomId string `json:"room_id"`
}
type LoadGameResponse = string

type ChangePlayerRequest = struct {
	StatefulRequest
	RoomId string `json:"room_id"`
	Index  string `json:"index"`
}
type ChangePlayerResponse = string
type ToggleMultitapRequest struct {
	StatefulRequest
	RoomId string `json:"room_id"`
}
