const input = (() => {
    let pollIntervalMs = 10;
    let pollIntervalId = 0;
    let controllerChangedIndex = -1;

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
        [KEY.RIGHT]: false,
        // extra
        [KEY.R2]: false,
        [KEY.L2]: false,
        [KEY.R3]: false,
        [KEY.L3]: false
    };

    const controllerEncoded = new Array(5).fill(0);

    const keys = Object.keys(controllerState);

    const poll = () => {
        return {
            setPollInterval: (ms) => pollIntervalMs = ms,
            enable: () => {
                if (pollIntervalId > 0) return;

                log.info(`[input] poll set to ${pollIntervalMs}ms`);
                pollIntervalId = setInterval(sendControllerState, pollIntervalMs)
            },
            disable: () => {
                if (pollIntervalId < 1) return;

                log.info('[input] poll has been disabled');
                clearInterval(pollIntervalId);
                pollIntervalId = 0;
            }
        }
    };

    const sendControllerState = () => {
        if (controllerChangedIndex >= 0) {
            event.pub(CONTROLLER_UPDATED, _encodeState());
            controllerChangedIndex = -1;
        }
    };

    const setKeyState = (name, state) => {
        if (controllerState[name] !== undefined) {
            controllerState[name] = state;
            controllerChangedIndex = Math.max(controllerChangedIndex, 0);
        }
    };

    const setAxisChanged = (index, value) => {
        if (controllerEncoded[index+1] !== undefined) {
            controllerEncoded[index+1] = Math.floor(32767 * value);
            controllerChangedIndex = Math.max(controllerChangedIndex, index+1);
        }
    };

    /**
     * Converts key state into a bitmap and prepends it to the axes state.
     *
     * @returns {Uint16Array} The controller state.
     * First uint16 is the controller state bitmap.
     * The other uint16 are the axes values.
     * Truncated to the last value changed.
     *
     * @private
     */
    const _encodeState = () => {
        controllerEncoded[0] = 0;
        for (let i = 0, len = keys.length; i < len; i++) controllerEncoded[0] += controllerState[keys[i]] ? 1 << i : 0;

        return new Uint16Array(controllerEncoded.slice(0, controllerChangedIndex+1));
    }

    return {
        poll,
        setKeyState,
        setAxisChanged,
    }
})(event, KEY);
