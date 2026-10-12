import {
    pub,
    sub,
    AXIS_CHANGED,
    DPAD_TOGGLE,
    KEY_PRESSED,
    KEY_RELEASED,
    KEYBOARD_KEY_PRESSED,
    KEYBOARD_KEY_DOWN,
    KEYBOARD_KEY_UP,
    KEYBOARD_TOGGLE_FILTER_MODE,
    SETTINGS_CHANGED,
} from 'event';
import {KEY} from 'input';
import {log} from 'log';
import {opts, settings} from 'settings';

// default keyboard bindings (code -> key)
export const defaultKeyboardMap = Object.freeze({
    ArrowUp: KEY.UP,
    ArrowDown: KEY.DOWN,
    ArrowLeft: KEY.LEFT,
    ArrowRight: KEY.RIGHT,
    KeyX: KEY.A,
    KeyZ: KEY.B,
    KeyS: KEY.X,
    KeyA: KEY.Y,
    KeyQ: KEY.L,
    KeyW: KEY.R,
    KeyE: KEY.L2,
    KeyR: KEY.R2,
    KeyC: KEY.L3,
    KeyV: KEY.R3,
    Enter: KEY.START,
    ShiftRight: KEY.SELECT,
    // non-game
    Escape: KEY.MENU,
    F1: KEY.MENU,
    F2: KEY.SAVE,
    F4: KEY.LOAD,
    KeyF: KEY.FULL,
    F8: KEY.STATS,
    KeyT: KEY.CHAT,
});

let keyMap = {};
// when false, the next key press is reported for remapping instead of playing
let isKeysFilteredMode = true;
// game mode: mapped keys don't trigger browser actions (scrolling, F1 help...)
let capture = false;
// if the browser supports Keyboard Lock API (Firefox does not)
const hasKeyboardLock = ('keyboard' in navigator) && navigator.keyboard && ('lock' in navigator.keyboard);

let locked = false;

const isTyping = (t) => {
    if (!t) return false;
    const tag = t.tagName;
    return t.isContentEditable || tag === 'TEXTAREA' || tag === 'SELECT' ||
        (tag === 'INPUT' && !['checkbox', 'radio', 'range', 'button'].includes(t.type));
};

sub(KEYBOARD_TOGGLE_FILTER_MODE, data => {
    isKeysFilteredMode = data.mode !== undefined ? data.mode : !isKeysFilteredMode;
    log.debug(`New keyboard filter mode: ${isKeysFilteredMode}`);
});

let dpadMode = true;
let dpadState = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};

function onDpadToggle(checked) {
    if (dpadMode === checked) return;

    dpadMode = !dpadMode;
    if (dpadMode) {
        // reset dpad keys pressed before moving to analog stick mode
        for (const key in dpadState) {
            if (dpadState[key]) {
                dpadState[key] = false;
                pub(KEY_RELEASED, {key: key});
            }
        }
    } else {
        // reset analog stick axes before moving to dpad mode
        if (!!dpadState[KEY.RIGHT] - !!dpadState[KEY.LEFT] !== 0) {
            pub(AXIS_CHANGED, {id: 0, value: 0});
        }
        if (!!dpadState[KEY.DOWN] - !!dpadState[KEY.UP] !== 0) {
            pub(AXIS_CHANGED, {id: 1, value: 0});
        }
        dpadState = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};
    }
}

const lock = async (lock) => {
    locked = lock;
    if (hasKeyboardLock) {
        try {
            lock ? await navigator.keyboard.lock() : navigator.keyboard.unlock();
        } catch {
            // not in fullscreen
        }
    }
    // if the browser doesn't support keyboard lock, it will be emulated
};

const onKey = (code, evt, state) => {
    const key = keyMap[code];
    if (!key) return;

    if (dpadState[key] !== undefined) {
        dpadState[key] = state;
        if (!dpadMode) {
            const LR = key === KEY.LEFT || key === KEY.RIGHT;
            pub(AXIS_CHANGED, {
                id: +!LR,
                value: !!dpadState[LR ? KEY.RIGHT : KEY.DOWN] - !!dpadState[LR ? KEY.LEFT : KEY.UP]
            });
            return;
        }
    }
    pub(evt, {key: key, code: code});
};

// releases everything (e.g. when the window loses focus)
const releaseAll = () => {
    for (const code of Object.keys(keyMap)) onKey(code, KEY_RELEASED, false);
};

sub(DPAD_TOGGLE, (data) => onDpadToggle(data.checked));
sub(SETTINGS_CHANGED, () => {
    keyMap = settings.get(opts.INPUT_KEYBOARD_MAP) || {...defaultKeyboardMap};
});

/**
 * Keyboard controls.
 */
export const keyboard = {
    init: () => {
        keyMap = settings.loadOr(opts.INPUT_KEYBOARD_MAP, {...defaultKeyboardMap});
        const body = document.body;

        body.addEventListener('keyup', e => {
            if (!isKeysFilteredMode) {
                e.preventDefault();
                return;
            }
            if (isTyping(e.target) && !locked) return;
            !hasKeyboardLock && locked && e.preventDefault();

            let lock = locked;
            // hack with Esc up when outside of lock
            if (e.code === 'Escape' && locked) lock = true;

            lock ? pub(KEYBOARD_KEY_UP, e) : onKey(e.code, KEY_RELEASED, false);
        }, false);

        body.addEventListener('keydown', e => {
            if (!isKeysFilteredMode) {
                e.preventDefault();
                e.stopPropagation();
                pub(KEYBOARD_KEY_PRESSED, {key: e.code});
                return;
            }
            if (isTyping(e.target) && !locked) return;
            if ((capture && keyMap[e.code]) || (!hasKeyboardLock && locked)) e.preventDefault();
            if (e.repeat && !locked) return;

            locked ? pub(KEYBOARD_KEY_DOWN, e) : onKey(e.code, KEY_PRESSED, true);
        });

        window.addEventListener('blur', releaseAll);

        log.info('[input] keyboard has been initialized');
    },
    /** In game mode the mapped keys are captured (no browser default actions). */
    set capture(v) {
        capture = v;
    },
    get map() {
        return {...keyMap};
    },
    remap(map) {
        settings.set(opts.INPUT_KEYBOARD_MAP, map);
    },
    lock,
};
