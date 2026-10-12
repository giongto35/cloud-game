import {
    pub,
    sub,
    AXIS_CHANGED,
    DPAD_TOGGLE,
    GAMEPAD_BUTTON_CAPTURED,
    GAMEPAD_CONNECTED,
    GAMEPAD_DISCONNECTED,
    KEY_PRESSED,
    KEY_RELEASED,
    SETTINGS_CHANGED,
} from 'event';
import {KEY} from 'input';
import {log} from 'log';
import {opts, settings} from 'settings';

/**
 * Default bindings for the W3C "standard" gamepad layout (button index -> key).
 * Physical positions follow the SNES/RetroPad convention:
 * bottom = B, right = A, left = Y, top = X.
 */
export const defaultGamepadMap = Object.freeze({
    0: KEY.B,
    1: KEY.A,
    2: KEY.Y,
    3: KEY.X,
    4: KEY.L,
    5: KEY.R,
    6: KEY.L2,
    7: KEY.R2,
    8: KEY.SELECT,
    9: KEY.START,
    10: KEY.L3,
    11: KEY.R3,
    12: KEY.UP,
    13: KEY.DOWN,
    14: KEY.LEFT,
    15: KEY.RIGHT,
    16: KEY.MENU,
});

const POLL_MS = 8;
const SELECT = 8, START = 9;

let map = {...defaultGamepadMap};
let deadZone = 0.15;
let dpadMode = true;

let timer = null;
let activeIndex = null;
let capturing = false;

// per-pad state
const pads = new Map(); // index -> {buttons: bool[], axes: number[], keys: Set}
const stickDpad = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};
let axesSent = [0, 0, 0, 0];
let menuChord = false;

export const padType = (id = '') => {
    const s = id.toLowerCase();
    if (/054c|playstation|dualshock|dualsense|ps[345]/.test(s)) return 'ps';
    if (/045e|xbox|xinput/.test(s)) return 'xbox';
    if (/057e|nintendo|switch|joy-con|pro controller/.test(s)) return 'nintendo';
    return 'generic';
};

const pressed = (b) => !!b && (b.pressed || b.value > 0.5);

const emit = (key, down) => key && pub(down ? KEY_PRESSED : KEY_RELEASED, {key, pad: true});

const setStickDpad = (key, on) => {
    if (stickDpad[key] === on) return;
    stickDpad[key] = on;
    emit(key, on);
};

const setAxis = (id, value) => {
    if (Math.abs(value) < deadZone) value = 0;
    if (axesSent[id] === value) return;
    axesSent[id] = value;
    pub(AXIS_CHANGED, {id, value});
};

const releasePad = (state) => {
    state.keys.forEach((k) => emit(k, false));
    state.keys.clear();
};

const poll = () => {
    const list = navigator.getGamepads ? navigator.getGamepads() : [];
    for (const gp of list) {
        if (!gp) continue;
        let state = pads.get(gp.index);
        if (!state) {
            state = {buttons: [], axes: [], keys: new Set()};
            pads.set(gp.index, state);
        }

        // the most recently used pad becomes the active one
        const anyPressed = gp.buttons.some(pressed) || gp.axes.some((a) => Math.abs(a) > 0.5);
        if (anyPressed && activeIndex !== gp.index) {
            if (activeIndex !== null && pads.get(activeIndex)) releasePad(pads.get(activeIndex));
            activeIndex = gp.index;
            pub(GAMEPAD_CONNECTED, {id: gp.id, type: padType(gp.id), index: gp.index, switched: true});
        }

        // remap capture mode: report the first newly pressed button
        if (capturing) {
            gp.buttons.forEach((b, i) => {
                const p = pressed(b);
                if (p && !state.buttons[i]) {
                    capturing = false;
                    pub(GAMEPAD_BUTTON_CAPTURED, {index: i, type: padType(gp.id)});
                }
                state.buttons[i] = p;
            });
            continue;
        }

        if (gp.index !== activeIndex) {
            gp.buttons.forEach((b, i) => (state.buttons[i] = pressed(b)));
            continue;
        }

        // Start + Select chord opens the game menu
        const chord = pressed(gp.buttons[START]) && pressed(gp.buttons[SELECT]);
        if (chord && !menuChord) {
            menuChord = true;
            // the game must not keep Start/Select held
            for (const i of [START, SELECT]) {
                const k = map[i];
                if (state.keys.delete(k)) emit(k, false);
            }
            emit(KEY.MENU, true);
            emit(KEY.MENU, false);
        } else if (!chord) {
            menuChord = false;
        }

        gp.buttons.forEach((b, i) => {
            const p = pressed(b);
            if (p === !!state.buttons[i]) return;
            state.buttons[i] = p;
            const key = map[i];
            if (!key || (menuChord && (i === START || i === SELECT))) return;
            p ? state.keys.add(key) : state.keys.delete(key);
            emit(key, p);
        });

        // analog sticks
        const [lx = 0, ly = 0, rx = 0, ry = 0] = gp.axes;
        if (dpadMode) {
            setStickDpad(KEY.LEFT, lx <= -0.5);
            setStickDpad(KEY.RIGHT, lx >= 0.5);
            setStickDpad(KEY.UP, ly <= -0.5);
            setStickDpad(KEY.DOWN, ly >= 0.5);
        } else {
            setAxis(0, lx);
            setAxis(1, ly);
        }
        setAxis(2, rx);
        setAxis(3, ry);
    }
};

const start = () => {
    if (timer === null) timer = setInterval(poll, POLL_MS);
};

const stop = () => {
    clearInterval(timer);
    timer = null;
};

const onConnected = (e) => {
    const gp = e.gamepad;
    log.info(`Gamepad connected at index ${gp.index}: ${gp.id} (${gp.mapping || 'no mapping'})`);
    if (activeIndex === null) activeIndex = gp.index;
    start();
    pub(GAMEPAD_CONNECTED, {id: gp.id, type: padType(gp.id), index: gp.index, standard: gp.mapping === 'standard'});
};

const onDisconnected = (e) => {
    const state = pads.get(e.gamepad.index);
    if (state) releasePad(state);
    pads.delete(e.gamepad.index);
    log.info(`Gamepad disconnected at index ${e.gamepad.index}`);
    if (activeIndex === e.gamepad.index) {
        activeIndex = [...pads.keys()][0] ?? null;
    }
    if (!pads.size) stop();
    pub(GAMEPAD_DISCONNECTED, {id: e.gamepad.id});
};

const onDpadToggle = (checked) => {
    if (dpadMode === checked) return;
    dpadMode = checked;
    // reset whatever was held in the previous mode
    Object.keys(stickDpad).forEach((k) => setStickDpad(k, false));
    setAxis(0, 0);
    setAxis(1, 0);
};

const loadSettings = () => {
    map = settings.get(opts.INPUT_GAMEPAD_MAP) || {...defaultGamepadMap};
    deadZone = settings.get(opts.INPUT_DEADZONE) ?? 0.15;
};

sub(DPAD_TOGGLE, (data) => onDpadToggle(data.checked));
sub(SETTINGS_CHANGED, loadSettings);

/**
 * Gamepad controls (Gamepad API).
 */
export const joystick = {
    init: () => {
        settings.loadOr(opts.INPUT_GAMEPAD_MAP, {...defaultGamepadMap});
        settings.loadOr(opts.INPUT_DEADZONE, 0.15);
        loadSettings();
        window.addEventListener('gamepadconnected', onConnected);
        window.addEventListener('gamepaddisconnected', onDisconnected);
        // pads that were connected before the page loaded
        for (const gp of navigator.getGamepads?.() || []) {
            if (gp) onConnected({gamepad: gp});
        }
        log.info('[input] joystick has been initialized');
    },
    /** Information about the active pad, or null. */
    active() {
        const gp = activeIndex !== null ? navigator.getGamepads?.()[activeIndex] : null;
        return gp ? {id: gp.id, type: padType(gp.id), standard: gp.mapping === 'standard'} : null;
    },
    /** Starts listening for the next pressed button (remapping). */
    capture(on = true) {
        capturing = on;
        if (on) start();
    },
    get map() {
        return {...map};
    },
    remap(m) {
        settings.set(opts.INPUT_GAMEPAD_MAP, m);
    },
    /** Short vibration feedback, when supported. */
    rumble(ms = 200, strong = 0.6) {
        const gp = activeIndex !== null ? navigator.getGamepads?.()[activeIndex] : null;
        gp?.vibrationActuator?.playEffect?.('dual-rumble', {
            duration: ms, strongMagnitude: strong, weakMagnitude: strong / 2,
        }).catch(() => ({}));
    },
};
