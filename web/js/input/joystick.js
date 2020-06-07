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
const joystick = (() => {
    let joystickMap;
    let joystickState;
    let joystickAxes;
    let joystickIdx;
    let joystickTimer = null;

    // check state for each axis -> dpad
    function checkJoystickAxisState(name, state) {
        if (joystickState[name] !== state) {
            joystickState[name] = state;
            event.pub(state === true ? KEY_PRESSED : KEY_RELEASED, {key: name});
        }
    }

    // loop timer for checking joystick state
    function checkJoystickState() {
        let gamepad = navigator.getGamepads()[joystickIdx];
        if (gamepad) {
            // Could reuse this logic with a toggle or with key remapping
            // // axis -> dpad
            // let corX = gamepad.axes[0]; // -1 -> 1, left -> right
            // let corY = gamepad.axes[1]; // -1 -> 1, up -> down
            // checkJoystickAxisState(KEY.LEFT, corX <= -0.5);
            // checkJoystickAxisState(KEY.RIGHT, corX >= 0.5);
            // checkJoystickAxisState(KEY.UP, corY <= -0.5);
            // checkJoystickAxisState(KEY.DOWN, corY >= 0.5);
            gamepad.axes.forEach(function (value, index) {
                if (-0.1 < value && value < 0.1) value = 0;
                if (joystickAxes[index] !== value) {
                    joystickAxes[index] = value;
                    event.pub(AXIS_CHANGED, {id: index, value: value});
                }
            });

            // normal button map
            Object.keys(joystickMap).forEach(function (btnIdx) {
                const buttonState = gamepad.buttons[btnIdx];

                const isPressed = navigator.webkitGetGamepads ? buttonState === 1 :
                    buttonState.value > 0 || buttonState.pressed === true;

                if (joystickState[btnIdx] !== isPressed) {
                    joystickState[btnIdx] = isPressed;
                    event.pub(isPressed === true ? KEY_PRESSED : KEY_RELEASED, {key: joystickMap[btnIdx]});
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
        let os = env.getOs();
        let browser = env.getBrowser();

        if (os === 'android') {
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

        if (os === 'android' && (browser === 'firefox' || browser === 'uc')) { //KeyMap2
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

        if (os === 'win' && browser === 'firefox') { //KeyMap3
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

        if (os === 'mac' && browser === 'safari') { //KeyMap4
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

        if (os === 'mac' && browser === 'firefox') { //KeyMap5
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
        if (browser === 'chrome' && gamepad.id.includes('PLAYSTATION(R)3')) {
            joystickMap = {
                0: KEY.A,
                1: KEY.B,
                2: KEY.Y,
                3: KEY.X,
                4: KEY.L,
                5: KEY.R,
                8: KEY.SELECT,
                9: KEY.START,
                10: KEY.L3,
                11: KEY.R3,
            };
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

        joystickTimer = setInterval(checkJoystickState, 10); // miliseconds per hit
        event.pub(GAMEPAD_CONNECTED);
    };

    return {
        init: () => {
            // we only capture the last plugged joystick
            window.addEventListener('gamepadconnected', onGamepadConnected);

            // disconnected event is triggered
            window.addEventListener('gamepaddisconnected', (event) => {
                clearInterval(joystickTimer);
                log.info(`Gamepad disconnected at index ${event.gamepad.index}`);
                event.pub(GAMEPAD_DISCONNECTED);
            });

            log.info('[input] joystick has been initialized');
        }
    }
})(event, env, KEY, navigator, window);
