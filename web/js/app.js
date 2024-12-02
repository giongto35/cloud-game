import {log} from 'log';
import {opts, settings} from 'settings';
import {api} from 'api';
import {
    APP_VIDEO_CHANGED,
    AXIS_CHANGED,
    CONTROLLER_UPDATED,
    DPAD_TOGGLE,
    FULLSCREEN_CHANGE,
    GAME_ERROR_NO_FREE_SLOTS,
    GAME_PLAYER_IDX,
    GAME_PLAYER_IDX_SET,
    GAME_ROOM_AVAILABLE,
    GAME_SAVED,
    GAMEPAD_CONNECTED,
    GAMEPAD_DISCONNECTED,
    HELP_OVERLAY_TOGGLED,
    KB_MOUSE_FLAG,
    KEY_PRESSED,
    KEY_RELEASED,
    KEYBOARD_KEY_DOWN,
    KEYBOARD_KEY_UP,
    LATENCY_CHECK_REQUESTED,
    MESSAGE,
    MOUSE_MOVED,
    MOUSE_PRESSED,
    POINTER_LOCK_CHANGE,
    RECORDING_STATUS_CHANGED,
    RECORDING_TOGGLED,
    REFRESH_INPUT,
    SETTINGS_CHANGED,
    WEBRTC_CONNECTION_CLOSED,
    WEBRTC_CONNECTION_READY,
    WEBRTC_ICE_CANDIDATE_FOUND,
    WEBRTC_ICE_CANDIDATE_RECEIVED,
    WEBRTC_ICE_CANDIDATES_FLUSH,
    WEBRTC_NEW_CONNECTION,
    WEBRTC_SDP_ANSWER,
    WEBRTC_SDP_OFFER,
    WORKER_LIST_FETCHED,
    pub,
    sub,
} from 'event';
import {gui} from 'gui';
import {input, KEY} from 'input';
import {socket, webrtc} from 'network';
import {debounce} from 'utils';

import {gameList} from './gameList.js?v=3';
import {menu} from './menu.js?v=3';
import {message} from './message.js?v=3';
import {recording} from './recording.js?v=3';
import {room} from './room.js?v=3';
import {screen} from './screen.js?v=3';
import {stats} from './stats.js?v=3';
import {stream} from './stream.js?v=3';
import {workerManager} from "./workerManager.js?v=3";

settings.init();
log.level = settings.loadOr(opts.LOG_LEVEL, log.DEFAULT);

// application display state
let state;
let lastState;

// first user interaction
let interacted = false;

const helpOverlay = document.getElementById('help-overlay');
const playerIndex = document.getElementById('playeridx');

// screen init
screen.add(menu, stream);

// keymap
const keyButtons = {};
Object.keys(KEY).forEach(button => {
    keyButtons[KEY[button]] = document.getElementById(`btn-${KEY[button]}`);
});

/**
 * State machine transition.
 * @param newState A new state strictly from app.state.*
 * @example
 * setState(app.state.eden)
 */
const setState = (newState = app.state.eden) => {
    if (newState === state) return;

    const prevState = state;

    // keep the current state intact for one of the "uber" states
    if (state && state._uber) {
        // if we are done with the uber state
        if (lastState === newState) state = newState;
        lastState = newState;
    } else {
        lastState = state
        state = newState;
    }

    if (log.level === log.DEBUG) {
        const previous = prevState ? prevState.name : '???';
        const current = state ? state.name : '???';
        const kept = lastState ? lastState.name : '???';

        log.debug(`[state] ${previous} -> ${current} [${kept}]`);
    }
};

const onConnectionReady = () => room.id ? startGame() : state.menuReady()

const onLatencyCheck = async (data) => {
    message.show('Connecting to fastest server...');
    const servers = await workerManager.checkLatencies(data);
    const latencies = Object.assign({}, ...servers);
    log.info('[ping] <->', latencies);
    api.server.latencyCheck(data.packetId, latencies);
};

const helpScreen = {
    shown: false,
    show: function (show, event) {
        if (this.shown === show) return;

        const isGameScreen = state === app.state.game
        screen.toggle(undefined, !show);

        gui.toggle(keyButtons[KEY.SAVE], show || isGameScreen);
        gui.toggle(keyButtons[KEY.LOAD], show || isGameScreen);

        gui.toggle(helpOverlay, show)

        this.shown = show;

        if (event) pub(HELP_OVERLAY_TOGGLED, {shown: show});
    }
};

const showMenuScreen = () => {
    log.debug('[control] loading menu screen');

    gui.hide(keyButtons[KEY.SAVE]);
    gui.hide(keyButtons[KEY.LOAD]);

    gameList.show();
    screen.toggle(menu);

    setState(app.state.menu);
};

const startGame = () => {
    if (!webrtc.isConnected()) {
        message.show('Game cannot load. Please refresh');
        return;
    }

    if (!webrtc.isInputReady()) {
        message.show('Game is not ready yet. Please wait');
        return;
    }

    log.info('[control] game start');

    setState(app.state.game);

    screen.toggle(stream)

    api.game.start(
        gameList.selected,
        room.id,
        recording.isActive(),
        recording.getUser(),
        +playerIndex.value - 1,
    )

    gameList.disable()
    input.retropad.toggle(false)
    gui.show(keyButtons[KEY.SAVE]);
    gui.show(keyButtons[KEY.LOAD]);
    input.retropad.toggle(true)
};

const saveGame = debounce(() => api.game.save(), 1000);
const loadGame = debounce(() => api.game.load(), 1000);

const onMessage = (m) => {
    const {id, t, p: payload} = m;
    switch (t) {
        case api.endpoint.INIT:
            pub(WEBRTC_NEW_CONNECTION, payload);
            break;
        case api.endpoint.OFFER:
            pub(WEBRTC_SDP_OFFER, {sdp: payload});
            break;
        case api.endpoint.ICE_CANDIDATE:
            pub(WEBRTC_ICE_CANDIDATE_RECEIVED, {candidate: payload});
            break;
        case api.endpoint.GAME_START:
            payload.av && pub(APP_VIDEO_CHANGED, payload.av)
            payload.kb_mouse && pub(KB_MOUSE_FLAG)
            pub(GAME_ROOM_AVAILABLE, {roomId: payload.roomId});
            break;
        case api.endpoint.GAME_SAVE:
            pub(GAME_SAVED);
            break;
        case api.endpoint.GAME_LOAD:
            break;
        case api.endpoint.GAME_SET_PLAYER_INDEX:
            pub(GAME_PLAYER_IDX_SET, payload);
            break;
        case api.endpoint.GET_WORKER_LIST:
            pub(WORKER_LIST_FETCHED, payload);
            break;
        case api.endpoint.LATENCY_CHECK:
            pub(LATENCY_CHECK_REQUESTED, {packetId: id, addresses: payload});
            break;
        case api.endpoint.GAME_RECORDING:
            pub(RECORDING_STATUS_CHANGED, payload);
            break;
        case api.endpoint.GAME_ERROR_NO_FREE_SLOTS:
            pub(GAME_ERROR_NO_FREE_SLOTS);
            break;
        case api.endpoint.APP_VIDEO_CHANGE:
            pub(APP_VIDEO_CHANGED, {...payload})
            break;
    }
}

const _dpadArrowKeys = [KEY.UP, KEY.DOWN, KEY.LEFT, KEY.RIGHT];

// pre-state key press handler
const onKeyPress = (data) => {
    const button = keyButtons[data.key];

    if (_dpadArrowKeys.includes(data.key)) {
        button.classList.add('dpad-pressed');
    } else {
        if (button) button.classList.add('pressed');
    }

    if (state !== app.state.settings) {
        if (KEY.HELP === data.key) helpScreen.show(true, event);
    }

    state.keyPress(data.key, data.code)
};

// pre-state key release handler
const onKeyRelease = data => {
    const button = keyButtons[data.key];

    if (_dpadArrowKeys.includes(data.key)) {
        button.classList.remove('dpad-pressed');
    } else {
        if (button) button.classList.remove('pressed');
    }

    if (state !== app.state.settings) {
        if (KEY.HELP === data.key) helpScreen.show(false, event);
    }

    // maybe move it somewhere
    if (!interacted) {
        // unmute when there is user interaction
        stream.audio.mute(false);
        interacted = true;
    }

    // change app state if settings
    if (KEY.SETTINGS === data.key) setState(app.state.settings);

    state.keyRelease(data.key, data.code);
};

const updatePlayerIndex = (idx, not_game = false) => {
    playerIndex.value = idx + 1;
    !not_game && api.game.setPlayerIndex(idx);
};

// noop function for the state
const _nil = () => ({/*_*/})

const onAxisChanged = (data) => {
    // maybe move it somewhere
    if (!interacted) {
        // unmute when there is user interaction
        stream.audio.mute(false);
        interacted = true;
    }

    state.axisChanged(data.id, data.value);
};

const handleToggle = (force = false) => {
    const toggle = document.getElementById('dpad-toggle');

    force && toggle.setAttribute('checked', '')
    toggle.checked = !toggle.checked;
    pub(DPAD_TOGGLE, {checked: toggle.checked});
};

const handleRecording = (data) => {
    const {recording, userName} = data;
    api.game.toggleRecording(recording, userName);
}

const handleRecordingStatus = (data) => {
    if (data === 'ok') {
        message.show(`Recording ${recording.isActive() ? 'on' : 'off'}`)
        if (recording.isActive()) {
            recording.setIndicator(true)
        }
    } else {
        message.show(`Recording failed ):`)
        recording.setIndicator(false)
    }
    log.debug("recording is ", recording.isActive())
}

const _default = {
    name: 'default',
    axisChanged: _nil,
    keyPress: _nil,
    keyRelease: _nil,
    menuReady: _nil,
}
const app = {
    state: {
        eden: {
            ..._default,
            name: 'eden',
            menuReady: showMenuScreen
        },

        settings: {
            ..._default,
            _uber: true,
            name: 'settings',
            keyRelease: (() => {
                settings.ui.onToggle = (o) => !o && setState(lastState);
                return (key) => key === KEY.SETTINGS && settings.ui.toggle()
            })(),
            menuReady: showMenuScreen
        },

        menu: {
            ..._default,
            name: 'menu',
            axisChanged: (id, val) => id === 1 && gameList.scroll(val < -.5 ? -1 : val > .5 ? 1 : 0),
            keyPress: (key) => {
                switch (key) {
                    case KEY.UP:
                    case KEY.DOWN:
                        gameList.scroll(key === KEY.UP ? -1 : 1)
                        break;
                }
            },
            keyRelease: (key) => {
                switch (key) {
                    case KEY.UP:
                    case KEY.DOWN:
                        gameList.scroll(0);
                        break;
                    case KEY.JOIN:
                    case KEY.A:
                    case KEY.B:
                    case KEY.X:
                    case KEY.Y:
                    case KEY.START:
                    case KEY.SELECT:
                        startGame();
                        break;
                    case KEY.QUIT:
                        message.show('You are already in menu screen!');
                        break;
                    case KEY.LOAD:
                        message.show('Loading the game.');
                        break;
                    case KEY.SAVE:
                        message.show('Saving the game.');
                        break;
                    case KEY.STATS:
                        stats.toggle();
                        break;
                    case KEY.SETTINGS:
                        break;
                    case KEY.DTOGGLE:
                        handleToggle();
                        break;
                }
            },
        },

        game: {
            ..._default,
            name: 'game',
            axisChanged: (id, value) => input.retropad.setAxisChanged(id, value),
            keyboardInput: (pressed, e) => api.game.input.keyboard.press(pressed, e),
            mouseMove: (e) => api.game.input.mouse.move(e.dx, e.dy),
            mousePress: (e) => api.game.input.mouse.press(e.b, e.p),
            keyPress: (key) => input.retropad.setKeyState(key, true),
            keyRelease: function (key) {
                input.retropad.setKeyState(key, false);

                switch (key) {
                    case KEY.JOIN: // or SHARE
                        // save when click share
                        saveGame();
                        room.copyToClipboard();
                        message.show('Shared link copied to the clipboard!');
                        break;
                    case KEY.SAVE:
                        saveGame();
                        break;
                    case KEY.LOAD:
                        loadGame();
                        break;
                    case KEY.FULL:
                        screen.fullscreen();
                        break;
                    case KEY.PAD1:
                        updatePlayerIndex(0);
                        break;
                    case KEY.PAD2:
                        updatePlayerIndex(1);
                        break;
                    case KEY.PAD3:
                        updatePlayerIndex(2);
                        break;
                    case KEY.PAD4:
                        updatePlayerIndex(3);
                        break;
                    case KEY.QUIT:
                        input.retropad.toggle(false)
                        api.game.quit(room.id)
                        room.reset();
                        window.location = window.location.pathname;
                        break;
                    case KEY.RESET:
                        api.game.reset(room.id)
                        break;
                    case KEY.STATS:
                        stats.toggle();
                        break;
                    case KEY.DTOGGLE:
                        handleToggle();
                        break;
                }
            },
        }
    }
};

// switch keyboard+mouse / retropad
const kbmEl = document.getElementById('kbm')
const kbmEl2 = document.getElementById('kbm2')
let kbmSkip = false
const kbmCb = () => {
    input.kbm = kbmSkip
    kbmSkip = !kbmSkip
    pub(REFRESH_INPUT)
}
gui.multiToggle([kbmEl, kbmEl2], {
    list: [
        {caption: '⌨️+🖱️', cb: kbmCb},
        {caption: ' 🎮 ', cb: kbmCb}
    ]
})
sub(KB_MOUSE_FLAG, () => {
    gui.show(kbmEl, kbmEl2)
    handleToggle(true)
    message.show('Keyboard and mouse work in fullscreen')
})

// Browser lock API
document.onpointerlockchange = () => pub(POINTER_LOCK_CHANGE, document.pointerLockElement)
document.onfullscreenchange = () => pub(FULLSCREEN_CHANGE, document.fullscreenElement)

// subscriptions
sub(MESSAGE, onMessage);

sub(GAME_ROOM_AVAILABLE, async () => {
    stream.play()
}, 2)
sub(GAME_SAVED, () => message.show('Saved'));
sub(GAME_PLAYER_IDX, data => {
    updatePlayerIndex(+data.index, state !== app.state.game);
});
sub(GAME_PLAYER_IDX_SET, idx => {
    if (!isNaN(+idx)) message.show(+idx + 1);
});
sub(GAME_ERROR_NO_FREE_SLOTS, () => message.show("No free slots :(", 2500));
sub(WEBRTC_NEW_CONNECTION, (data) => {
    workerManager.whoami(data.wid);
    webrtc.onData = (x) => onMessage(api.decode(x.data))
    webrtc.start(data.ice);
    api.server.initWebrtc()
    gameList.set(data.games);
});
sub(WEBRTC_ICE_CANDIDATE_FOUND, (data) => api.server.sendIceCandidate(data.candidate));
sub(WEBRTC_SDP_ANSWER, (data) => api.server.sendSdp(data.sdp));
sub(WEBRTC_SDP_OFFER, (data) => webrtc.setRemoteDescription(data.sdp, stream.video.el));
sub(WEBRTC_ICE_CANDIDATE_RECEIVED, (data) => webrtc.addCandidate(data.candidate));
sub(WEBRTC_ICE_CANDIDATES_FLUSH, () => webrtc.flushCandidates());
sub(WEBRTC_CONNECTION_READY, onConnectionReady);
sub(WEBRTC_CONNECTION_CLOSED, () => {
    input.retropad.toggle(false)
    webrtc.stop();
});
sub(LATENCY_CHECK_REQUESTED, onLatencyCheck);
sub(GAMEPAD_CONNECTED, () => message.show('Gamepad connected'));
sub(GAMEPAD_DISCONNECTED, () => message.show('Gamepad disconnected'));

// keyboard handler in the Screen Lock mode
sub(KEYBOARD_KEY_DOWN, (v) => state.keyboardInput?.(true, v))
sub(KEYBOARD_KEY_UP, (v) => state.keyboardInput?.(false, v))

// mouse handler in the Screen Lock mode
sub(MOUSE_MOVED, (e) => state.mouseMove?.(e))
sub(MOUSE_PRESSED, (e) => state.mousePress?.(e))

// general keyboard handler
sub(KEY_PRESSED, onKeyPress);
sub(KEY_RELEASED, onKeyRelease);

sub(SETTINGS_CHANGED, () => message.show('Settings have been updated'));
sub(AXIS_CHANGED, onAxisChanged);
sub(CONTROLLER_UPDATED, data => webrtc.input(data));
sub(RECORDING_TOGGLED, handleRecording);
sub(RECORDING_STATUS_CHANGED, handleRecordingStatus);

sub(SETTINGS_CHANGED, () => {
    const s = settings.get();
    log.level = s[opts.LOG_LEVEL];
});

// initial app state
setState(app.state.eden);

input.init()

stream.init();
screen.init();

let [roomId, zone] = room.loadMaybe();
// find worker id if present
const wid = new URLSearchParams(document.location.search).get('wid');
// if from URL -> start game immediately!
socket.init(roomId, wid, zone);
api.transport = {
    send: socket.send,
    keyboard: webrtc.keyboard,
    mouse: webrtc.mouse,
}

// stats
let WEBRTC_STATS_RTT;
let VIDEO_BITRATE;
let GET_V_CODEC, SET_CODEC;

const bitrate = (() => {
    let bytesPrev, timestampPrev
    const w = [0, 0, 0, 0, 0, 0]
    const n = w.length
    let i = 0
    return (now, bytes) => {
        w[i++ % n] = timestampPrev ? Math.floor(8 * (bytes - bytesPrev) / (now - timestampPrev)) : 0
        bytesPrev = bytes
        timestampPrev = now
        return Math.floor(w.reduce((a, b) => a + b) / n)
    }
})()

stats.modules = [
    {
        mui: stats.mui('', '<1'),
        init() {
            WEBRTC_STATS_RTT = (v) => (this.val = v)
        },
    },
    {
        mui: stats.mui('', '', false, () => ''),
        init() {
            GET_V_CODEC = (v) => (this.val = v + ' @ ')
        }
    },
    {
        mui: stats.mui('', '', false, () => ''),
        init() {
            sub(APP_VIDEO_CHANGED, (payload) => (this.val = `${payload.w}x${payload.h}`))
        },
    },
    {
        mui: stats.mui('', '', false, () => ' kb/s', 'stats-bitrate'),
        init() {
            VIDEO_BITRATE = (v) => (this.val = v)
        }
    },
    {
        async stats() {
            const stats = await webrtc.stats();
            if (!stats) return;

            stats.forEach(report => {
                if (!SET_CODEC && report.mimeType?.startsWith('video/')) {
                    GET_V_CODEC(report.mimeType.replace('video/', '').toLowerCase())
                    SET_CODEC = 1
                }
                const {nominated, currentRoundTripTime, type, kind} = report;
                if (nominated && currentRoundTripTime !== undefined) {
                    WEBRTC_STATS_RTT(currentRoundTripTime * 1000);
                }
                if (type === 'inbound-rtp' && kind === 'video') {
                    VIDEO_BITRATE(bitrate(report.timestamp, report.bytesReceived))
                }
            });
        },
        enable() {
            this.interval = window.setInterval(this.stats, 999);
        },
        disable() {
            window.clearInterval(this.interval);
        },
    }]

stats.toggle()
