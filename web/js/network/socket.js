import {
    pub,
    MESSAGE
} from 'event';
import {log} from 'log';

let conn;

const buildUrl = (params = {}) => {
    const url = new URL(window.location);
    url.protocol = location.protocol !== 'https:' ? 'ws' : 'wss';
    url.pathname = "/ws";
    Object.keys(params).forEach(k => {
        if (!!params[k]) url.searchParams.set(k, params[k])
    })
    return url
}

const init = (roomId, wid, zone) => {
    let objParams = {room_id: roomId, zone: zone};
    if (wid) objParams.wid = wid;
    const url = buildUrl(objParams)
    log.info(`[ws] connecting to ${url}`);
    conn = new WebSocket(url.toString());
    conn.onopen = () => {
        log.info('[ws] <- open connection');
    };
    conn.onerror = () => log.error('[ws] some error!');
    conn.onclose = (event) => log.info(`[ws] closed (${event.code})`);
    conn.onmessage = response => {
        const data = JSON.parse(response.data);
        log.debug('[ws] <- ', data);
        pub(MESSAGE, data);
    };
};

const send = (data) => {
    if (conn.readyState === 1) {
        conn.send(JSON.stringify(data));
    }
}

/**
 * WebSocket connection module.
 *
 *  Needs init() call.
 */
export const socket = {
    init,
    send
}
