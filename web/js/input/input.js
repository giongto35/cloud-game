import {
    REFRESH_INPUT,
    KB_MOUSE_FLAG,
    pub,
    sub
} from 'event';

export {KEY, JOYPAD_KEYS, BINDABLE} from './keys.js?v=4';

import {joystick, defaultGamepadMap, padType} from './joystick.js?v=4';
import {keyboard, defaultKeyboardMap} from './keyboard.js?v=4';
import {pointer} from './pointer.js?v=4';
import {retropad} from './retropad.js?v=4';
import {touch, isTouchDevice} from './touch.js?v=4';

export {joystick, keyboard, pointer, retropad, touch, isTouchDevice, defaultGamepadMap, defaultKeyboardMap, padType};

const input_state = {
    retropad: false,
    kbm: false,
};

const init = () => {
    keyboard.init();
    joystick.init();
};

sub(KB_MOUSE_FLAG, () => {
    input_state.kbm = true;
    pub(REFRESH_INPUT);
});

export const input = {
    state: input_state,
    init,
    retropad: {
        ...retropad,
        toggle(on = true) {
            if (on === input_state.retropad) return;
            input_state.retropad = on;
            retropad.toggle(on);
        }
    },
    set kbm(v) {
        input_state.kbm = v;
    },
    get kbm() {
        return input_state.kbm;
    }
};
