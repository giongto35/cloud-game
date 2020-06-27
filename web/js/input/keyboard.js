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
        220: KEY.STATS, // backslash
        84: KEY.DTOGGLE, // t
        77: KEY.MULTITAP, // m
    };

    let dpadMode = true;
    let dpadState = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};

    function onDpadToggle(checked) {
      if (dpadMode === checked) {
        return //error?
      }
      if (dpadMode) {
        dpadMode = false;
        // reset dpad keys pressed before moving to analog stick mode
        for (const key in dpadState) {
            if (dpadState[key] === true) {
                dpadState[key] = false;
                event.pub(KEY_RELEASED, {key: key});
            }
        }
      } else {
        dpadMode = true;
        // reset analog stick axes before moving to dpad mode
        value = (dpadState[KEY.RIGHT] === true ? 1 : 0) - (dpadState[KEY.LEFT] === true ? 1 : 0)
        if (value !== 0) {
          event.pub(AXIS_CHANGED, {id: 0, value: 0});
        }
        value = (dpadState[KEY.DOWN] === true ? 1 : 0) - (dpadState[KEY.UP] === true ? 1 : 0)
        if (value !== 0) {
          event.pub(AXIS_CHANGED, {id: 1, value: 0});
        }
        dpadState = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};
      }
    }

    const onKey = (code, callback, state) => {
        if (code in KEYBOARD_MAP) {
          key = KEYBOARD_MAP[code]
          if (key in dpadState) {
            dpadState[key] = state
            if (dpadMode) {
              callback(key);
            } else {
              if (key === KEY.LEFT || key == KEY.RIGHT) {
                value = (dpadState[KEY.RIGHT] === true ? 1 : 0) - (dpadState[KEY.LEFT] === true ? 1 : 0)
                event.pub(AXIS_CHANGED, {id: 0, value: value});
              } else {
                value = (dpadState[KEY.DOWN] === true ? 1 : 0) - (dpadState[KEY.UP] === true ? 1 : 0)
                event.pub(AXIS_CHANGED, {id: 1, value: value});
              }
            }
          } else {
            callback(key);
          }
        }
    };

    event.sub(DPAD_TOGGLE, (data) => onDpadToggle(data.checked));

    return {
        init: () => {
            const body = $('body');
            body.on('keyup', ev => onKey(ev.keyCode, key => event.pub(KEY_RELEASED, {key: key}), false));
            body.on('keydown', ev => onKey(ev.keyCode, key => event.pub(KEY_PRESSED, {key: key}), true));
            log.info('[input] keyboard has been initialized');
        }
    }
})(event, KEY);
