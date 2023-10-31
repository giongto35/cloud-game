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
        GAME_RECORDING: 110,
        GET_WORKER_LIST: 111,
        GAME_ERROR_NO_FREE_SLOTS: 112,

        APP_VIDEO_CHANGE: 150,
    });

    const packet = (type, payload, id) => {
        const packet = {t: type};
        if (id !== undefined) packet.id = id;
        if (payload !== undefined) packet.p = payload;

        socket.send(packet);
    };

    const decodeBytes = (b) => String.fromCharCode.apply(null, new Uint8Array(b))

    return Object.freeze({
        endpoint: endpoints,
        decode: (b) => JSON.parse(decodeBytes(b)),
        server:
            {
                initWebrtc: () => packet(endpoints.INIT_WEBRTC),
                sendIceCandidate: (candidate) => packet(endpoints.ICE_CANDIDATE, btoa(JSON.stringify(candidate))),
                sendSdp: (sdp) => packet(endpoints.ANSWER, btoa(JSON.stringify(sdp))),
                latencyCheck: (id, list) => packet(endpoints.LATENCY_CHECK, list, id),
                getWorkerList: () => packet(endpoints.GET_WORKER_LIST),
            },
        game:
            {
                load: () => packet(endpoints.GAME_LOAD),
                save: () => packet(endpoints.GAME_SAVE),
                setPlayerIndex: (i) => packet(endpoints.GAME_SET_PLAYER_INDEX, i),
                start: (game, roomId, record, recordUser, player) => packet(endpoints.GAME_START, {
                    game_name: game,
                    room_id: roomId,
                    player_index: player,
                    record: record,
                    record_user: recordUser,
                }),
                toggleMultitap: () => packet(endpoints.GAME_TOGGLE_MULTITAP),
                toggleRecording: (active = false, userName = '') =>
                    packet(endpoints.GAME_RECORDING, {
                        active: active,
                        user: userName,
                    }),
                quit: (roomId) => packet(endpoints.GAME_QUIT, {room_id: roomId}),
            }
    })
})(socket);
