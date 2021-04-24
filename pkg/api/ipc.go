package api

const (
	PtUnknown uint8 = 0
	PtEcho    uint8 = 1
	ptInit    uint8 = 2

	P_latencyCheck uint8 = 3
	P_init         uint8 = 4

	P_webrtc_init          uint8 = 100
	P_webrtc_offer         uint8 = 101
	P_webrtc_answer        uint8 = 102
	P_webrtc_ice_candidate uint8 = 103

	P_game_start uint8 = 104
	P_game_quit  uint8 = 105
	P_game_save  uint8 = 106
	P_game_load  uint8 = 107

	P_game_set_player_index = 108
	P_game_toggle_multitap  = 109
)
