/**
 * Keyboard controls.
 *
 * @version 1
 */
const keyboard = (() => {
    const defaultMap = Object.freeze({
        37: KEY.LEFT,
        38: KEY.UP,
        39: KEY.RIGHT,
        40: KEY.DOWN,
        90: KEY.A, // z
        88: KEY.B, // x
        67: KEY.X, // c
        86: KEY.Y, // v
        65: KEY.L, // a
        83: KEY.R, // s
        13: KEY.START, // enter
        16: KEY.SELECT, // shift
        // non-game
        81: KEY.QUIT, // q
        87: KEY.JOIN, // w
        75: KEY.SAVE, // k
        76: KEY.LOAD, // l
        49: KEY.PAD1, // 1
        50: KEY.PAD2, // 2
        51: KEY.PAD3, // 3
        52: KEY.PAD4, // 4
        70: KEY.FULL, // f
        72: KEY.HELP, // h
        220: KEY.STATS, // backspace
    });

    const settingsKey = 'input.keyboard.map';
    const keyMap = settings.loadOr(settingsKey, defaultMap);

    const remap = (map = {}) => {
        // map = {38: KEY.DOWN, 40: KEY.UP};

        settings.set(settingsKey, map);
        console.log('remapped')
    }

    const onKey = (code, callback) => !keyMap[code] || callback(keyMap[code]);

    return {
        init: () => {
            const body = document.body;
            body.addEventListener('keyup', e => onKey(e.keyCode, key => event.pub(KEY_RELEASED, {key: key})));
            body.addEventListener('keydown', e => onKey(e.keyCode, key => event.pub(KEY_PRESSED, {key: key})));
            log.info('[input] keyboard has been initialized');
        },
        settings: {
            remap
        }
    }
})
(event, document, KEY, settings);
