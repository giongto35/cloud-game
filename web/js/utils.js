/**
 * Utility module.
 * @version 1
 */
const utils = (() => {
    return {
        /**
         * A decorator that passes the call to function at maximum once per specified milliseconds.
         * @param f The function to call.
         * @param ms The amount of time in milliseconds to ignore the function calls.
         * @returns {Function}
         * @example
         * const showMessage = () => { alert('00001'); }
         * const showOnlyOnceASecond = debounce(showMessage, 1000);
         */
        debounce: (f, ms) => {
            let wait = false;

            return function () {
                if (wait) return;

                f.apply(this, arguments);
                wait = true;
                setTimeout(() => wait = false, ms);
            };
        },

        /**
         * A decorator that blocks and calls the last function until the specified amount of milliseconds.
         * @param f The function to call.
         * @param ms The amount of time in milliseconds to ignore the function calls.
         * @returns {Function}
         */
        throttle: (f, ms) => {
            let lastCall;
            let lastTime;

            return function () {
                // could be a stack
                const lastContext = this;
                const lastArguments = arguments;

                if (!lastTime) {
                    f.apply(lastContext, lastArguments);
                    lastTime = Date.now()
                } else {
                    clearTimeout(lastCall);
                    lastCall = setTimeout(() => {
                        if (Date.now() - lastTime >= ms) {
                            f.apply(lastContext, lastArguments);
                            lastTime = Date.now()
                        }
                    }, ms - (Date.now() - lastTime))
                }
            }
        }
    }
})();
