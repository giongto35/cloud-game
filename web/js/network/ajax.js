/**
 * AJAX request module.
 * @version 1
 */
const ajax = (() => {
    const defaultTimeout = 10000;

    return {
        fetch: (url, options, timeout = defaultTimeout) => new Promise((resolve, reject) => {
            const controller = new AbortController();
            const signal = controller.signal;
            const allOptions = Object.assign({}, options, signal);

            // fetch(url, {...options, signal})
            fetch(url, allOptions)
                .then(resolve, () => {
                    controller.abort();
                    return reject
                });

            // auto abort when a timeout reached
            setTimeout(() => {
                controller.abort();
                reject();
            }, timeout);
        }),
        defaultTimeoutMs: () => defaultTimeout
    }
})();