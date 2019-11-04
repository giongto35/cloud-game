const log = (() => {
    let level = 'info';
    const levels = {'trace': 0, 'debug': 1, 'error': 2, 'info': 3};

    const atLeast = (lv) => (levels[lv] || -1) >= levels[level];

    return {
        info: function () {
            atLeast('info') && console.info.apply(null, arguments)
        },
        debug: function () {
            atLeast('debug') && console.debug.apply(null, arguments)
        },
        error: function () {
            atLeast('error') && console.error.apply(null, arguments)
        },
        setLevel: (level_) => {
            level = level_
        }
    }
})();
