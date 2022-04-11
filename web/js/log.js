const log = (() => {
    const levels = {'trace': 0, 'debug': 1, 'error': 2, 'warning': 3, 'info': 4};
    let level = -1;

    const atLeast = lv => (lv || -1) >= level;
    const noop = () => ({});

    const
        info = atLeast(levels.info) ? console.info.bind(window.console) : noop,
        debug = atLeast(levels.debug) ? console.debug.bind(window.console) : noop,
        error = atLeast(levels.error) ? console.error.bind(window.console) : noop,
        warning = atLeast(levels.warning) ? console.warn.bind(window.console) : noop;

    return {
        level: levels,
        info,
        debug,
        error,
        warning,
        setLevel: (level_) => level = levels[level_] || -1,
        is: (level_) => level === level_
    }
})(console, window);
