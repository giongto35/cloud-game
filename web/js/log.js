const log = (() => {
    const levels = {'trace': 0, 'debug': 1, 'error': 2, 'warning': 3, 'info': 4};
    let level = -1;

    const atLeast = lv => (lv || -1) >= level;

    return {
        level: levels,
        info: function () {
            atLeast(levels.info) && console.info.apply(null, arguments)
        },
        debug: function () {
            atLeast(levels.debug) && console.debug.apply(null, arguments)
        },
        error: function () {
            console.error.apply(null, arguments)
        },
        warning: function () {
            atLeast(levels.warning) && console.warn.apply(null, arguments)
        },
        setLevel: (level_) => {
            level = levels[level_] || -1;
        },
        is: (level_) => level === level_
    }
})(console);
