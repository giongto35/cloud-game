import {
    REFRESH_INPUT,
    KB_MOUSE_FLAG,
    pub,
    sub
} from 'event';

export {KEY, JOYPAD_KEYS} from './keys.js?v=3';

import {joystick} from './joystick.js?v=3';
import {keyboard} from './keyboard.js?v=3'
import {pointer} from './pointer.js?v=3';
import {retropad} from './retropad.js?v=3';
import {touch} from './touch.js?v=3';

export {joystick, keyboard, pointer, retropad, touch};

const input_state = {
    joystick: true,
    keyboard: false,
    pointer: true, // aka mouse
    retropad: true,
    touch: true,

    kbm: false,
}

const init = () => {
    keyboard.init()
    joystick.init()
    touch.init()
}

sub(KB_MOUSE_FLAG, () => {
    input_state.kbm = true
    pub(REFRESH_INPUT)
})

export const input = {
    state: input_state,
    init,
    retropad: {
        ...retropad,
        toggle(on = true) {
            if (on === input_state.retropad) return
            input_state.retropad = on
            retropad.toggle(on)
        }
    },
    set kbm(v) {
        input_state.kbm = v
    },
    get kbm() {
        return input_state.kbm
    }
}
