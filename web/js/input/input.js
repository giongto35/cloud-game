const input = (() => {
    let pollIntervalMs = 10;
    let pollIntervalId = 0;
    let isStateChanged = false;

    let controllerState = {
        // control
        [KEY.A]: false,
        [KEY.B]: false,
        [KEY.X]: false,
        [KEY.Y]: false,
        [KEY.L]: false,
        [KEY.R]: false,
        [KEY.SELECT]: false,
        [KEY.START]: false,
        // dpad
        [KEY.UP]: false,
        [KEY.DOWN]: false,
        [KEY.LEFT]: false,
        [KEY.RIGHT]: false
    };

    const keys = Object.keys(controllerState);

    const poll = () => {
        return {
            setPollInterval: (ms) => pollIntervalMs = ms,
            enable: () => {
                if (pollIntervalId > 0) return;

                log.info(`[input] poll set to ${pollIntervalMs}ms`);
                pollIntervalId = setInterval(sendKeyState, pollIntervalMs)
            },
            disable: () => {
                if (pollIntervalId < 1) return;

                log.info('[input] poll has been disabled');
                clearInterval(pollIntervalId);
                pollIntervalId = 0;
            }
        }
    };

    const sendKeyState = () => {
        if (isStateChanged) {
            event.pub(KEY_STATE_UPDATED, _encodeState());
            isStateChanged = false;
        }
    };

    const setKeyState = (name, state) => {
        if (controllerState[name] !== undefined) {
            controllerState[name] = state;
            isStateChanged = true;
        }
    };

    /**
     * Converts controller state into a binary number.
     *
     * @returns {Uint8Array} The controller state.
     * First byte is controller state.
     * Second byte is d-pad state converted (shifted) into a byte.
     * So the whole state is just splitted by 8 bits.
     *
     * @private
     */
    const _encodeState = () => {
        let result = 0;
        for (let i = 0, len = keys.length; i < len; i++) result += controllerState[keys[i]] ? 1 << i : 0;

        return new Uint8Array([result & ((1 << 8) - 1), result >> 8]);
    }

    return {
        poll,
        setKeyState,
    }
})(event, KEY);
