/**
 * Keyboard controls.
 *
 * @version 1
 */
const keyboard = (() => {

    const KEYBOARD_MAP = {
        37: KEY.LEFT,
        38: KEY.UP,
        39: KEY.RIGHT,
        40: KEY.DOWN,
        90: KEY.A, // z
        88: KEY.B, // x
        67: KEY.X, // c
        86: KEY.Y, // v
        13: KEY.START, // enter
        16: KEY.SELECT, // shift
        // non-game
        81: KEY.QUIT, // q
        83: KEY.SAVE, // s
        87: KEY.JOIN, // w
        65: KEY.LOAD, // a
        70: KEY.FULL, // f
        72: KEY.HELP, // h
    };

    const onKey = (code, callback) => {
        if (code in KEYBOARD_MAP) callback(KEYBOARD_MAP[code]);
    };

    return {
        init: () => {
            const body = $('body');
            body.on('keyup', ev => onKey(ev.keyCode, key => event.pub(KEY_RELEASED, {key: key})));
            body.on('keydown', ev => onKey(ev.keyCode, key => event.pub(KEY_PRESSED, {key: key})));
            log.info('[input] keyboard has been initialized');
        }
    }
})(event, KEY);
