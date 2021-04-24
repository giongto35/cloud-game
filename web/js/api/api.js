//

const LATENCY_CHECK = 3
const INIT = 4

const INIT_WEBRTC = 100
const OFFER = 101
const ANSWER = 102
const ICE_CANDIDATE = 103

const GAME_START = 104
const GAME_QUIT = 105
const GAME_SAVE = 106
const GAME_LOAD = 107
const GAME_SET_PLAYER_INDEX = 108
const GAME_TOGGLE_MULTITAP = 109

const gameStartRequest = (game = '', roomId = '', player = 0) => ({
    t: GAME_START,
    payload: JSON.stringify({
        game_name: game,
        room_id: roomId,
        player_index: player,
    })
})

const gameQuitRequest = (roomId = '') => (
    {
        t: GAME_QUIT,
        payload: JSON.stringify({
            room_id: roomId
        })
    }
)
