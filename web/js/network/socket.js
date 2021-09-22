/**
 * WebSocket connection module.
 *
 *  Needs init() call.
 *
 * @version 1
 *
 * Events:
 *   @link MESSAGE
 *
 */
const socket = (() => {
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

    const init = (roomId, zone) => {
        const url = buildUrl({room_id: roomId, zone: zone})
        console.info(`[ws] connecting to ${url}`);
        conn = new WebSocket(url.toString());
        conn.onopen = () => {
            log.info('[ws] <- open connection');
        };
        conn.onerror = () => log.error('[ws] some error!');
        conn.onclose = (event) => log.info(`[ws] closed (${event.code})`);
        conn.onmessage = response => {
            const data = JSON.parse(response.data);
            log.debug('[ws] <- ', data);
            event.pub(MESSAGE, data);
        };
    };

    const send = (data) => {
        if (conn.readyState === 1) {
            conn.send(JSON.stringify(data));
        }
    }

    return {
        init: init,
        send: send,
    }
})(event, log);
