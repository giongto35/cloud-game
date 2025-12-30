import {pub, CONTROLLER_UPDATED} from 'event';
import {JOYPAD_KEYS} from 'input';

/*
 * [BUTTONS, LEFT_X, LEFT_Y, RIGHT_X, RIGHT_Y]
 *
 * Buttons are packed into a 16-bit bitmask where each bit is one button.
 * Axes are signed 16-bit values ranging from -32768 to 32767.
 * The whole thing is 10 bytes when sent over the wire.
 */
const state = new Int16Array(5);
let buttons = 0;
let dirty = false;
let rafId = 0;

/*
 * Polls controller state using requestAnimationFrame which gives us
 * ~60Hz update rate that syncs with the display. As a bonus,
 * it automatically pauses when the tab goes to background.
 * We only send data when something actually changed.
 */
const poll = () => {
    if (dirty) {
        state[0] = buttons;
        pub(CONTROLLER_UPDATED, new Uint16Array(state.buffer));
        dirty = false;
    }
    rafId = requestAnimationFrame(poll);
};

/*
 * Toggles a button on or off in the bitmask. The button's position
 * in JOYPAD_KEYS determines which bit gets flipped. For example,
 * if A is at index 8, pressing it sets bit 8.
 */
const setKeyState = (key, pressed) => {
    const idx = JOYPAD_KEYS.indexOf(key);
    if (idx < 0) return;

    const prev = buttons;
    buttons = pressed ? buttons | (1 << idx) : buttons & ~(1 << idx);
    dirty ||= buttons !== prev;
};

/*
 * Updates an analog stick axis. Axes 0-1 are the left stick (X and Y),
 * axes 2-3 are the right stick. Input should be a float from -1 to 1
 * which gets converted to a signed 16-bit integer for transmission.
 */
const setAxisChanged = (axis, value) => {
    if (axis < 0 || axis > 3) return;

    const v = Math.trunc(Math.max(-1, Math.min(1, value)) * 32767);
    dirty ||= state[++axis] !== v;
    state[axis] = v;
};

// Starts or stops the polling loop
const toggle = (on) => {
    if (on === !!rafId) return;
    rafId = on ? requestAnimationFrame(poll) : (cancelAnimationFrame(rafId), 0);
};

export const retropad = {toggle, setKeyState, setAxisChanged};