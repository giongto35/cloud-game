const defaultTimeout = 10000;
/**
 * AJAX request module.
 * @version 1
 */
export const ajax = {
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
