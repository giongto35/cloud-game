/**
 * WebSocket connection module.
 *
 *  Needs init() call.
 *
 * @version 1
 */
const socket = (() => {
    const pingIntervalMs = 1000 / 5;

    let conn;
    let curPacketId = '';

    const init = (roomId, zone) => {
        const paramString = new URLSearchParams({room_id: roomId, zone: zone})

        // if localhost, local LAN connection
        if (location.hostname === "localhost" || location.hostname === "127.0.0.1" || location.hostname.startsWith("192.168")) {
            scheme = "ws"
        } else {
            scheme = "wss"
        }
        conn = new WebSocket(`${scheme}://${location.host}/ws?${paramString.toString()}`);

        // Clear old roomID
        conn.onopen = () => {
            log.info('[ws] <- open connection');
            log.info(`[ws] -> setting ping interval to ${pingIntervalMs}ms`);
            // !to add destructor if SPA
            setInterval(ping, pingIntervalMs)
        };
        conn.onerror = error => log.error(`[ws] ${error}`);
        conn.onclose = () => log.info('[ws] closed');
        // Message received from server
        conn.onmessage = response => {
            const data = JSON.parse(response.data);
            const message = data.id;

            if (message !== 'heartbeat') log.debug(`[ws] <- message '${message}' `, data);

            switch (message) {
                case 'init':
                    // TODO: Read from struct
                    // init package has 2 part [stunturn, game1, game2, game3 ...]
                    // const [stunturn, ...games] = data;
                    let serverData = JSON.parse(data.data);
                    event.pub(MEDIA_STREAM_INITIALIZED, {stunturn: serverData.shift(), games: serverData});
                    break;
                case 'sdp':
                    event.pub(MEDIA_STREAM_SDP_AVAILABLE, {sdp: data.data});
                    break;
                case 'requestOffer':
                    // !to remove? wtf
                    curPacketId = data.packet_id;
                    event.pub(MEDIA_STREAM_READY);
                    break;
                case 'heartbeat':
                    event.pub(PING_RESPONSE);
                    break;
                case 'start':
                    event.pub(GAME_ROOM_AVAILABLE, {roomId: data.room_id});
                    break;
                case 'save':
                    event.pub(GAME_SAVED);
                    break;
                case 'load':
                    event.pub(GAME_LOADED);
                    break;
                case 'playerIdx':
                    event.pub(GAME_PLAYER_IDX, data.data);
                    break;
                case 'checkLatency':
                    curPacketId = data.packet_id;
                    const addresses = data.data.split(',');
                    event.pub(LATENCY_CHECK_REQUESTED, {packetId: curPacketId, addresses: addresses});
            }
        };
    };

    // TODO: format the package with time
    const ping = () => {
        const time = Date.now();
        send({"id": "heartbeat", "data": time.toString()});
        event.pub(PING_REQUEST, {interval: pingIntervalMs, time: time});
    }
    const send = (data) => conn.send(JSON.stringify(data));
    const latency = (workers, packetId) => send({
        "id": "checkLatency",
        "data": JSON.stringify(workers),
        "packet_id": packetId
    });
    const saveGame = () => send({"id": "save", "data": ""});
    const loadGame = () => send({"id": "load", "data": ""});
    const updatePlayerIndex = (idx) => send({"id": "playerIdx", "data": idx.toString()});
    const startGame = (gameName, isMobile, roomId, playerIndex) => send({
        "id": "start",
        "data": JSON.stringify({
            "game_name": gameName,
            "is_mobile": isMobile
        }),
        "room_id": roomId != null ? roomId : '',
        "player_index": playerIndex
    });
    const quitGame = (roomId) => send({"id": "quit", "data": "", "room_id": roomId});

    return {
        init: init,
        send: send,
        latency: latency,
        saveGame: saveGame,
        loadGame: loadGame,
        updatePlayerIndex: updatePlayerIndex,
        startGame: startGame,
        quitGame: quitGame
    }
})($, event, log);
