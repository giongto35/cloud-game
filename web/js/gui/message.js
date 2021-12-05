/**
 * App UI message module.
 *
 * @version 1
 */
const message = (() => {
    const popupBox = document.getElementById('noti-box');

    // fifo queue
    let queue = [];
    const queueMaxSize = 5;

    let isScreenFree = true;

    const _popup = () => {
        // recursion edge case:
        // no messages in the queue or one on the screen
        if (!(queue.length > 0 && isScreenFree)) {
            return;
        }

        isScreenFree = false;
        popupBox.innerText = queue.shift();
        gui.anim.fadeInOut(popupBox, 1000, .05).finally(() => {
            isScreenFree = true;
            _popup();
        })
    }

    const _storeMessage = (text) => {
        if (queue.length <= queueMaxSize) {
            queue.push(text);
        }
    }

    const _proceed = (text) => {
        _storeMessage(text);
        _popup();
    }

    const show = (text) => {
        _proceed(text)
    }

    return Object.freeze({
        show: show
    })
})(document, gui, utils);
