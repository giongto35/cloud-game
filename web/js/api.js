import {log} from 'log';

const endpoints = {
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
    GAME_RECORDING: 110,
    GET_WORKER_LIST: 111,
    GAME_ERROR_NO_FREE_SLOTS: 112,

    APP_VIDEO_CHANGE: 150,
}

/**
 * Server API.
 *
 * Requires the actual api.transport implementation.
 */
export const api = {
    set transport(t) {
        transport = t;
    },
    endpoint: endpoints,
    decode: (b) => JSON.parse(decodeBytes(b)),
    server: {
        initWebrtc: () => packet(endpoints.INIT_WEBRTC),
        sendIceCandidate: (candidate) => packet(endpoints.ICE_CANDIDATE, btoa(JSON.stringify(candidate))),
        sendSdp: (sdp) => packet(endpoints.ANSWER, btoa(JSON.stringify(sdp))),
        latencyCheck: (id, list) => packet(endpoints.LATENCY_CHECK, list, id),
        getWorkerList: () => packet(endpoints.GET_WORKER_LIST),
    },
    game: {
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
        toggleRecording: (active = false, userName = '') =>
            packet(endpoints.GAME_RECORDING, {active: active, user: userName}),
        quit: (roomId) => packet(endpoints.GAME_QUIT, {room_id: roomId}),
    }
}

let transport = {
    send: (packet) => {
        log.warn('Default transport is used! Change it with the api.transport variable.', packet)
    }
}

const packet = (type, payload, id) => {
    const packet = {t: type};
    if (id !== undefined) packet.id = id;
    if (payload !== undefined) packet.p = payload;
    transport.send(packet);
}

const decodeBytes = (b) => String.fromCharCode.apply(null, new Uint8Array(b))
