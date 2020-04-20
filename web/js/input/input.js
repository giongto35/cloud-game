const input = (() => {
    const INPUT_HZ = 100;
    const INPUT_STATE_PACKET = 1;
    const KEY_BITS = [KEY.A, KEY.B, KEY.X, KEY.Y, KEY.L, KEY.R, KEY.SELECT, KEY.START, KEY.UP, KEY.DOWN, KEY.LEFT, KEY.RIGHT];

    let gameInputTimer = null;
    let unchangePacket = 0;

    // Game controller state
    let keyState = {
        // control
        [KEY.A]: false,
        [KEY.B]: false,
        [KEY.X]: false,
        [KEY.Y]: false,
        [KEY.L]: false,
        [KEY.R]: false,
        [KEY.START]: false,
        [KEY.SELECT]: false,
        // dpad
        [KEY.UP]: false,
        [KEY.DOWN]: false,
        [KEY.LEFT]: false,
        [KEY.RIGHT]: false
    };

    const poll = () => {
        return {
            enable: () => {
                if (gameInputTimer !== null) return;

                const inputPollInterval = 1000 / INPUT_HZ;
                log.info(`[input] setting input polling interval to ${inputPollInterval}ms`);
                gameInputTimer = setInterval(sendKeyState, inputPollInterval)
            },
            disable: () => {
                if (gameInputTimer === null) return;

                log.info('[input] stop game input timer');
                clearInterval(gameInputTimer);
                gameInputTimer = null;
            }
        }
    };

    // relatively slow method
    const sendKeyState = () => {
        // check if state is changed
        if (unchangePacket > 0) {
            // pack keys state
            let bits = '';
            KEY_BITS.slice().reverse().forEach(elem => {
                bits += keyState[elem] ? 1 : 0;
            });
            let data = parseInt(bits, 2);

            let arrBuf = new Uint8Array(2);
            arrBuf[0] = data & ((1 << 8) - 1);
            arrBuf[1] = data >> 8;
            event.pub(KEY_STATE_UPDATED, arrBuf);

            unchangePacket--;
        }
    };

    const setKeyState = (name, state) => {
        if (name in keyState) {
            keyState[name] = state;
            unchangePacket = INPUT_STATE_PACKET;
        }
    };

    return {
        poll: poll,
        setKeyState: setKeyState
    }
})(event, KEY);
