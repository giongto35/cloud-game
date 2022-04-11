/**
 * Logging module.
 *
 * @version 2
 */
const log = (() => {
    const noop = () => ({})

    const _log = {
        ASSERT: 1,
        ERROR: 2,
        WARN: 3,
        INFO: 4,
        DEBUG: 5,
        TRACE: 6,

        DEFAULT: 5,

        set level(level) {
            this.assert = level >= this.ASSERT ? console.assert.bind(window.console) : noop;
            this.error = level >= this.ERROR ? console.error.bind(window.console) : noop;
            this.warn = level >= this.WARN ? console.warn.bind(window.console) : noop;
            this.info = level >= this.INFO ? console.info.bind(window.console) : noop;
            this.debug = level >= this.DEBUG ? console.debug.bind(window.console) : noop;
            this.trace = level >= this.TRACE ? console.log.bind(window.console) : noop;
            this._level = level;
        },
        get level() {
            return this._level;
        }
    }
    _log.level = _log.DEFAULT;

    return _log
})(console, window);
