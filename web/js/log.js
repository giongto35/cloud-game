const log = (() => {
    const levels = {'trace': 0, 'debug': 1, 'error': 2, 'warning': 3, 'info': 4};
    let level = 'info';

    const atLeast = (lv) => (levels[lv] || -1) >= levels[level];

    return {
        info: function () {
            atLeast('info') && console.info.apply(null, arguments)
        },
        debug: function () {
            atLeast('debug') && console.debug.apply(null, arguments)
        },
        error: function () {
            console.error.apply(null, arguments)
        },
        warning: function() {
            atLeast('warning') && console.warn.apply(null, arguments)
        },
        setLevel: (level_) => {
            level = level_
        }
    }
})(console);
