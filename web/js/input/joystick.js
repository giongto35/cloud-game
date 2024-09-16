import {
    pub,
    sub,
    AXIS_CHANGED,
    DPAD_TOGGLE,
    GAMEPAD_CONNECTED,
    GAMEPAD_DISCONNECTED,
    KEY_PRESSED,
    KEY_RELEASED
} from 'event';
import {env, browser as br, platform} from 'env';
import {KEY} from 'input';
import {log} from 'log';

const deadZone = 0.1;
let joystickMap;
let joystickState = {};
let joystickAxes = [];
let joystickIdx;
let joystickTimer = null;
let dpadMode = true;

function onDpadToggle(checked) {
    if (dpadMode === checked) {
        return //error?
    }
    if (dpadMode) {
        dpadMode = false;
        // reset dpad keys pressed before moving to analog stick mode
        checkJoystickAxisState(KEY.LEFT, false);
        checkJoystickAxisState(KEY.RIGHT, false);
        checkJoystickAxisState(KEY.UP, false);
        checkJoystickAxisState(KEY.DOWN, false);
    } else {
        dpadMode = true;
        // reset analog stick axes before moving to dpad mode
        joystickAxes.forEach(function (value, index) {
            checkJoystickAxis(index, 0);
        });
    }
}

// check state for each axis -> dpad
function checkJoystickAxisState(name, state) {
    if (joystickState[name] !== state) {
        joystickState[name] = state;
        pub(state === true ? KEY_PRESSED : KEY_RELEASED, {key: name});
    }
}

function checkJoystickAxis(axis, value) {
    if (-deadZone < value && value < deadZone) value = 0;
    if (joystickAxes[axis] !== value) {
        joystickAxes[axis] = value;
        pub(AXIS_CHANGED, {id: axis, value: value});
    }
}

// loop timer for checking joystick state
function checkJoystickState() {
    let gamepad = navigator.getGamepads()[joystickIdx];
    if (gamepad) {
        if (dpadMode) {
            // axis -> dpad
            let corX = gamepad.axes[0]; // -1 -> 1, left -> right
            let corY = gamepad.axes[1]; // -1 -> 1, up -> down
            checkJoystickAxisState(KEY.LEFT, corX <= -0.5);
            checkJoystickAxisState(KEY.RIGHT, corX >= 0.5);
            checkJoystickAxisState(KEY.UP, corY <= -0.5);
            checkJoystickAxisState(KEY.DOWN, corY >= 0.5);
        } else {
            gamepad.axes.forEach(function (value, index) {
                checkJoystickAxis(index, value);
            });
        }

        // normal button map
        Object.keys(joystickMap).forEach(function (btnIdx) {
            const buttonState = gamepad.buttons[btnIdx];

            const isPressed = navigator.webkitGetGamepads ? buttonState === 1 :
                buttonState.value > 0 || buttonState.pressed === true;

            if (joystickState[btnIdx] !== isPressed) {
                joystickState[btnIdx] = isPressed;
                pub(isPressed === true ? KEY_PRESSED : KEY_RELEASED, {key: joystickMap[btnIdx]});
            }
        });
    }
}

// we only capture the last plugged joystick
const onGamepadConnected = (e) => {
    let gamepad = e.gamepad;
    log.info(`Gamepad connected at index ${gamepad.index}: ${gamepad.id}. ${gamepad.buttons.length} buttons, ${gamepad.axes.length} axes.`);

    joystickIdx = gamepad.index;

    // Ref: https://github.com/giongto35/cloud-game/issues/14
    // get mapping first (default KeyMap2)
    const os = env.getOs;
    const browser = env.getBrowser;

    if (os === platform.android) {
        // default of android is KeyMap1
        joystickMap = {
            2: KEY.A,
            0: KEY.B,
            3: KEY.START,
            4: KEY.SELECT,
            10: KEY.LOAD,
            11: KEY.SAVE,
            8: KEY.HELP,
            9: KEY.QUIT,
            12: KEY.UP,
            13: KEY.DOWN,
            14: KEY.LEFT,
            15: KEY.RIGHT
        };
    } else {
        // default of other OS is KeyMap2
        joystickMap = {
            0: KEY.A,
            1: KEY.B,
            2: KEY.START,
            3: KEY.SELECT,
            8: KEY.LOAD,
            9: KEY.SAVE,
            6: KEY.HELP,
            7: KEY.QUIT,
            12: KEY.UP,
            13: KEY.DOWN,
            14: KEY.LEFT,
            15: KEY.RIGHT
        };
    }

    if (os === platform.android && browser === br.firefox) { //KeyMap2
        joystickMap = {
            0: KEY.A,
            1: KEY.B,
            2: KEY.START,
            3: KEY.SELECT,
            8: KEY.LOAD,
            9: KEY.SAVE,
            6: KEY.HELP,
            7: KEY.QUIT,
            12: KEY.UP,
            13: KEY.DOWN,
            14: KEY.LEFT,
            15: KEY.RIGHT
        };
    }

    if (os === platform.windows && browser === br.firefox) { //KeyMap3
        joystickMap = {
            1: KEY.A,
            2: KEY.B,
            0: KEY.START,
            3: KEY.SELECT,
            8: KEY.LOAD,
            9: KEY.SAVE,
            6: KEY.HELP,
            7: KEY.QUIT
        };
    }

    if (os === platform.macos && browser === br.safari) { //KeyMap4
        joystickMap = {
            1: KEY.A,
            2: KEY.B,
            0: KEY.START,
            3: KEY.SELECT,
            8: KEY.LOAD,
            9: KEY.SAVE,
            6: KEY.HELP,
            7: KEY.QUIT,
            14: KEY.UP,
            15: KEY.DOWN,
            16: KEY.LEFT,
            17: KEY.RIGHT
        };
    }

    if (os === platform.macos && browser === br.firefox) { //KeyMap5
        joystickMap = {
            1: KEY.A,
            2: KEY.B,
            0: KEY.START,
            3: KEY.SELECT,
            8: KEY.LOAD,
            9: KEY.SAVE,
            6: KEY.HELP,
            7: KEY.QUIT,
            14: KEY.UP,
            15: KEY.DOWN,
            16: KEY.LEFT,
            17: KEY.RIGHT
        };
    }

    // https://bugs.chromium.org/p/chromium/issues/detail?id=1076272
    if (gamepad.id.includes('PLAYSTATION(R)3')) {
        if (browser === br.chrome) {
            joystickMap = {
                1: KEY.A,
                0: KEY.B,
                2: KEY.Y,
                3: KEY.X,
                4: KEY.L,
                5: KEY.R,
                8: KEY.SELECT,
                9: KEY.START,
                10: KEY.DTOGGLE,
                11: KEY.R3,
            };
        } else {
            joystickMap = {
                13: KEY.A,
                14: KEY.B,
                12: KEY.X,
                15: KEY.Y,
                3: KEY.START,
                0: KEY.SELECT,
                4: KEY.UP,
                6: KEY.DOWN,
                7: KEY.LEFT,
                5: KEY.RIGHT,
                10: KEY.L,
                11: KEY.R,
                8: KEY.L2,
                9: KEY.R2,
                1: KEY.DTOGGLE,
                2: KEY.R3,
            };
        }
    }

    // reset state
    joystickState = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};
    Object.keys(joystickMap).forEach(function (btnIdx) {
        joystickState[btnIdx] = false;
    });

    joystickAxes = new Array(gamepad.axes.length).fill(0);

    // looper, too intense?
    if (joystickTimer !== null) {
        clearInterval(joystickTimer);
    }

    joystickTimer = setInterval(checkJoystickState, 10); // milliseconds per hit
    pub(GAMEPAD_CONNECTED);
};

sub(DPAD_TOGGLE, (data) => onDpadToggle(data.checked));

/**
 * Joystick controls.
 *
 * cross == a      <--> a
 * circle == b     <--> b
 * square == x     <--> start
 * triangle == y   <--> select
 * share           <--> load
 * option          <--> save
 * L2 == LT        <--> full
 * R2 == RT        <--> quit
 * dpad            <--> up down left right
 * axis 0, 1       <--> second dpad
 *
 * change full to help (temporary)
 *
 * @version 1
 */
export const joystick = {
    init: () => {
        // we only capture the last plugged joystick
        window.addEventListener('gamepadconnected', onGamepadConnected);

        // disconnected event is triggered
        window.addEventListener('gamepaddisconnected', (event) => {
            clearInterval(joystickTimer);
            log.info(`Gamepad disconnected at index ${event.gamepad.index}`);
            pub(GAMEPAD_DISCONNECTED);
        });

        log.info('[input] joystick has been initialized');
    }
}
