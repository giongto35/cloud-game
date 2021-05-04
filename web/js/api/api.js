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

/**
 * Server API.
 *
 * @version 1
 *
 */
const api = (() => {
    const packet = (type, payload, id) => {
        const packet = {t: type};
        if (id !== undefined) packet.id = id;
        if (payload !== undefined) packet.p = payload;

        socket.send(packet);
    };

    return Object.freeze({
        server: Object.freeze({
            initWebrtc: () => packet(INIT_WEBRTC),
            sendIceCandidate: (candidate) => packet(ICE_CANDIDATE, btoa(JSON.stringify(candidate))),
            sendSdp: (sdp) => packet(ANSWER, btoa(JSON.stringify(sdp))),
            latencyCheck: (id, list) => packet(LATENCY_CHECK, list, id),
        }),
        game: Object.freeze({
            load: () => packet(GAME_LOAD),
            save: () => packet(GAME_SAVE),
            setPlayerIndex: (i) => packet(GAME_SET_PLAYER_INDEX, '' + i),
            start: (game, roomId, player) => packet(GAME_START, {
                game_name: game,
                room_id: roomId,
                player_index: player
            }),
            toggleMultitap: () => packet(GAME_TOGGLE_MULTITAP),
            quit: (roomId) => packet(GAME_QUIT, {room_id: roomId}),
        })
    })
})(socket);
