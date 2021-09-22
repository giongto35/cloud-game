/**
 * Server API.
 *
 * @version 1
 *
 */
const api = (() => {
    const endpoints = Object.freeze({
        LATENCY_CHECK: 3,
        INIT: 4,
        INIT_WEBRTC: 100,
        OFFER: 101,
        ANSWER: 102,
        ICE_CANDIDATE: 103,
        GAME_START: 104,
        GAME_QUIT: 105,
        GAME_SAVE: 106,
        GAME_LOAD: 107,
        GAME_SET_PLAYER_INDEX: 108,
        GAME_TOGGLE_MULTITAP: 109,
    });

    const packet = (type, payload, id) => {
        const packet = {t: type};
        if (id !== undefined) packet.id = id;
        if (payload !== undefined) packet.p = payload;

        socket.send(packet);
    };

    return Object.freeze({
        endpoint: endpoints,
        server:
            Object.freeze({
                initWebrtc: () => packet(endpoints.INIT_WEBRTC),
                sendIceCandidate: (candidate) => packet(endpoints.ICE_CANDIDATE, btoa(JSON.stringify(candidate))),
                sendSdp: (sdp) => packet(endpoints.ANSWER, btoa(JSON.stringify(sdp))),
                latencyCheck: (id, list) => packet(endpoints.LATENCY_CHECK, list, id),
            }),
        game:
            Object.freeze({
                load: () => packet(endpoints.GAME_LOAD),
                save: () => packet(endpoints.GAME_SAVE),
                setPlayerIndex: (i) => packet(endpoints.GAME_SET_PLAYER_INDEX, '' + i),
                start: (game, roomId, player) => packet(endpoints.GAME_START, {
                    game_name: game,
                    room_id: roomId,
                    player_index: player
                }),
                toggleMultitap: () => packet(endpoints.GAME_TOGGLE_MULTITAP),
                quit: (roomId) => packet(endpoints.GAME_QUIT, {room_id: roomId}),
            })
    })
})(socket);
