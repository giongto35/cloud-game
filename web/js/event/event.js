/**
 * Event publishing / subscribe module.
 * Just a simple observer pattern.
 * @version 1
 */
const event = (() => {
    const topics = {};

    return {
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
        sub: (topic, listener, order) => {
            if (!topics[topic]) topics[topic] = [];

            // order handling stuff
            const value = {order: order || 0, listener: listener};
            topics[topic].push(value);
            topics[topic].sort((a, b) => a.order - b.order);
            const index = topics[topic].indexOf(value);

            return {
                unsub: () => {
                    topics[topic].splice(index, 1);
                }
            };
        },

        /**
         * Publishes some event for handling.
         *
         * @param topic The name of the event.
         * @param data Additional data for the event handling.
         * Because of compatibility we have to use a dumb obj wrapper {a: a, b: b} for params instead of (topic, ...data).
         * @example
         * event.pub('rapture', {time: now()})
         */
        pub: (topic, data) => {
            if (!topics[topic] || topics[topic].length < 1) return;

            topics[topic].forEach((listener) => {
                listener.listener(data !== undefined ? data : {})
            });
        }
    }
})();

// events
const LATENCY_CHECK_REQUESTED = 'latencyCheckRequested';
const PING_REQUEST = 'pingRequest';
const PING_RESPONSE = 'pingResponse';

const GAME_ROOM_AVAILABLE = 'gameRoomAvailable';
const GAME_SAVED = 'gameSaved';
const GAME_LOADED = 'gameLoaded';
const GAME_PLAYER_IDX = 'gamePlayerIndex';

const CONNECTION_READY = 'connectionReady';
const CONNECTION_CLOSED = 'connectionClosed';

const MEDIA_STREAM_INITIALIZED = 'mediaStreamInitialized';
const MEDIA_STREAM_SDP_AVAILABLE = 'mediaStreamSdpAvailable';
const MEDIA_STREAM_CANDIDATE_ADD = 'mediaStreamCandidateAdd';
const MEDIA_STREAM_CANDIDATE_FLUSH = 'mediaStreamCandidateFlush';
const MEDIA_STREAM_READY = 'mediaStreamReady';

const GAMEPAD_CONNECTED = 'gamepadConnected';
const GAMEPAD_DISCONNECTED = 'gamepadDisconnected';

const MENU_HANDLER_ATTACHED = 'menuHandlerAttached';
const MENU_PRESSED = 'menuPressed';
const MENU_RELEASED = 'menuReleased';

const KEY_PRESSED = 'keyPressed';
const KEY_RELEASED = 'keyReleased';
const KEY_STATE_UPDATED = 'keyStateUpdated';
const KEYBOARD_TOGGLE_FILTER_MODE = 'keyboardToggleFilterMode';
const KEYBOARD_KEY_PRESSED = 'keyboardKeyPressed';
const AXIS_CHANGED = 'axisChanged';
const CONTROLLER_UPDATED = 'controllerUpdated';

const DPAD_TOGGLE = 'dpadToggle';
const STATS_TOGGLE = 'statsToggle';
const HELP_OVERLAY_TOGGLED = 'helpOverlayToggled';

const SETTINGS_CHANGED = 'settingsChanged';
const SETTINGS_CLOSED = 'settingsClosed';
