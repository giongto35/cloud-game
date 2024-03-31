/**
 * Event publishing / subscribe module.
 * Just a simple observer pattern.
 */

const topics = {};

// internal listener index
let _index = 0;

/**
 * Subscribes onto some event.
 *
 * @param topic The name of the event.
 * @param listener A callback function to call during the event.
 * @param order A number in a queue of event handlers to run callback in ordered manner.
 * @returns {{unsub: unsub}} The function to remove this subscription.
 * @example
 * const sub01 = event.sub('rapture', () => {a}, 1)
 * ...
 * sub01.unsub()
 */
export const sub = (topic, listener, order = undefined) => {
    if (!topics[topic]) topics[topic] = {};
    // order index * big pad + next internal index (e.g. 1*100+1=101)
    // use some arbitrary big number to not overlap with non-ordered
    let i = (order !== undefined ? order * 1000000 : 0) + _index++;
    topics[topic][i] = listener;
    return {
        unsub: () => {
            delete topics[topic][i]
        }
    }
}

/**
 * Publishes some event for handling.
 *
 * @param topic The name of the event.
 * @param data Additional data for the event handling.
 * Because of compatibility we have to use a dumb obj wrapper {a: a, b: b} for params instead of (topic, ...data).
 * @example
 * event.pub('rapture', {time: now()})
 */
export const pub = (topic, data) => {
    if (!topics[topic]) return;
    Object.keys(topics[topic]).forEach((ls) => {
        topics[topic][ls](data !== undefined ? data : {})
    });
}

// events
export const LATENCY_CHECK_REQUESTED = 'latencyCheckRequested';

export const WORKER_LIST_FETCHED = 'workerListFetched';

export const GAME_ROOM_AVAILABLE = 'gameRoomAvailable';
export const GAME_SAVED = 'gameSaved';
export const GAME_LOADED = 'gameLoaded';
export const GAME_PLAYER_IDX = 'gamePlayerIndex';
export const GAME_PLAYER_IDX_SET = 'gamePlayerIndexSet'
export const GAME_ERROR_NO_FREE_SLOTS = 'gameNoFreeSlots'

export const WEBRTC_CONNECTION_CLOSED = 'webrtcConnectionClosed';
export const WEBRTC_CONNECTION_READY = 'webrtcConnectionReady';
export const WEBRTC_ICE_CANDIDATE_FOUND = 'webrtcIceCandidateFound'
export const WEBRTC_ICE_CANDIDATE_RECEIVED = 'webrtcIceCandidateReceived';
export const WEBRTC_ICE_CANDIDATES_FLUSH = 'webrtcIceCandidatesFlush';
export const WEBRTC_NEW_CONNECTION = 'webrtcNewConnection';
export const WEBRTC_SDP_ANSWER = 'webrtcSdpAnswer'
export const WEBRTC_SDP_OFFER = 'webrtcSdpOffer';

export const MESSAGE = 'message'

export const GAMEPAD_CONNECTED = 'gamepadConnected';
export const GAMEPAD_DISCONNECTED = 'gamepadDisconnected';

export const MENU_HANDLER_ATTACHED = 'menuHandlerAttached';
export const MENU_PRESSED = 'menuPressed';
export const MENU_RELEASED = 'menuReleased';

export const KEY_PRESSED = 'keyPressed';
export const KEY_RELEASED = 'keyReleased';
export const KEYBOARD_TOGGLE_FILTER_MODE = 'keyboardToggleFilterMode';
export const KEYBOARD_KEY_PRESSED = 'keyboardKeyPressed';
export const AXIS_CHANGED = 'axisChanged';
export const CONTROLLER_UPDATED = 'controllerUpdated';

export const DPAD_TOGGLE = 'dpadToggle';
export const STATS_TOGGLE = 'statsToggle';
export const HELP_OVERLAY_TOGGLED = 'helpOverlayToggled';

export const SETTINGS_CHANGED = 'settingsChanged';

export const RECORDING_TOGGLED = 'recordingToggle'
export const RECORDING_STATUS_CHANGED = 'recordingStatusChanged'

export const APP_VIDEO_CHANGED = 'appVideoChanged'
