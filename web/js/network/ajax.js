/**
 * AJAX request module.
 * @version 1
 */
const ajax = (() => {
    const timeout_ = 10000;

    return {
        fetch: (url, options, timeout = timeout_) => new Promise((resolve, reject) => {
            const controller = new AbortController();
            const signal = controller.signal;
            const options_ = Object.assign({}, options, signal);

            // fetch(url, {...options, signal})
            fetch(url, options_)
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
        timeout: () => timeout_
    }
})();