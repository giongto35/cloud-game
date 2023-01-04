const input = (() => {
    const pollingIntervalMs = 4;
    let controllerChangedIndex = -1;

    // Libretro config
    let controllerState = {
        [KEY.B]: false,
        [KEY.Y]: false,
        [KEY.SELECT]: false,
        [KEY.START]: false,
        [KEY.UP]: false,
        [KEY.DOWN]: false,
        [KEY.LEFT]: false,
        [KEY.RIGHT]: false,
        [KEY.A]: false,
        [KEY.X]: false,
        // extra
        [KEY.L]: false,
        [KEY.R]: false,
        [KEY.L2]: false,
        [KEY.R2]: false,
        [KEY.L3]: false,
        [KEY.R3]: false
    };

    const poll = (intervalMs, callback) => {
        let _ticker = 0;
        return {
            enable: () => {
                if (_ticker > 0) return;
                log.debug(`[input] poll set to ${intervalMs}ms`);
                _ticker = setInterval(callback, intervalMs)
            },
            disable: () => {
                if (_ticker < 1) return;
                log.debug('[input] poll has been disabled');
                clearInterval(_ticker);
                _ticker = 0;
            }
        }
    };

    const controllerEncoded = new Array(5).fill(0);
    const keys = Object.keys(controllerState);

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
        if (controllerEncoded[index + 1] !== undefined) {
            controllerEncoded[index + 1] = Math.floor(32767 * value);
            controllerChangedIndex = Math.max(controllerChangedIndex, index + 1);
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
        return new Uint16Array(controllerEncoded.slice(0, controllerChangedIndex + 1));
    }

    return {
        poll: poll(pollingIntervalMs, sendControllerState),
        setKeyState,
        setAxisChanged,
    }
})(event, KEY, log);
