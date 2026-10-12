import {pub, MESSAGE} from "event";
import {log} from "log";

let conn;
let closing = false;
let onClose = () => ({});

const buildUrl = (params = {}) => {
    const url = new URL(window.location);
    url.protocol = location.protocol !== "https:" ? "ws" : "wss";
    url.pathname = "/ws";
    url.search = "";
    url.hash = "";
    Object.keys(params).forEach((k) => {
        if (params[k]) url.searchParams.set(k, params[k]);
    });
    return url;
};

/**
 * Opens the coordinator connection.
 * @param {Object} params query params (room_id, zone, wid)
 * @param {Object} handlers
 * @param {Function} [handlers.onOpen]
 * @param {Function} [handlers.onClose] called with (event, intentional)
 */
const init = (params = {}, handlers = {}) => {
    const url = buildUrl(params);
    log.debug(`[ws] connecting to ${url}`);
    closing = false;
    onClose = handlers.onClose || (() => ({}));
    const ws = new WebSocket(url.toString());
    conn = ws;
    ws.onopen = () => {
        log.debug("[ws] opened");
        handlers.onOpen?.();
    };
    ws.onerror = () => log.error("[ws] error");
    ws.onclose = (event) => {
        log.debug(`[ws] closed (${event.code})`);
        if (conn === ws) onClose(event, closing);
    };
    ws.onmessage = (response) => pub(MESSAGE, JSON.parse(response.data));
};

const send = (data) => {
    if (conn?.readyState === 1) conn.send(JSON.stringify(data));
};

/** Closes the connection on purpose (no reconnection). */
const close = () => {
    closing = true;
    const ws = conn;
    conn = null;
    if (ws && ws.readyState <= 1) ws.close(1000);
    onClose = () => ({});
};

const isOpen = () => conn?.readyState === 1;

/**
 * WebSocket connection module.
 */
export const socket = {
    init,
    send,
    close,
    isOpen,
};
