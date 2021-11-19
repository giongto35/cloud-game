/**
 * Recording module.
 * @version 1
 */
const recording = (() => {
    const userName = document.getElementById('user-name'),
        recButton = document.getElementById('btn-rec');

    if (userName === undefined || recButton === undefined) {
        return {}
    }

    let state = {
        userName: '',
        _recording: false
    };

    const restoreLastState = () => {
        const lastState = localStorage.getItem('recording');
        if (lastState) {
            const _last = JSON.parse(lastState);
            if (_last) {
                state = _last;
            }
        }
        userName.value = state.userName
    }

    // persistence
    const saveLastState = () => {
        const _state = Object.keys(state)
            .filter(key => !key.startsWith('_'))
            .reduce((obj, key) => ({...obj, [key]: state[key]}), {});
        localStorage.setItem('recording', JSON.stringify(_state));
    }
    const saveUserName = utils.throttle(() => {
        state.userName = userName.value;
        saveLastState();
    }, 500)

    restoreLastState();

    // text
    userName.addEventListener('focus', () => event.pub(KEYBOARD_TOGGLE_FILTER_MODE))
    userName.addEventListener('blur', () => event.pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true}))
    userName.addEventListener('keyup', ev => {
        ev.stopPropagation();
        saveUserName()
    })

    // button
    recButton.addEventListener('click', ev => {
        state._recording = !state._recording
        recButton.classList.toggle('record', state._recording);
        event.pub(RECORDING_TOGGLED, {
            userName: state.userName,
            recording: state._recording,
        })
    })
    return {
        isActive: () => state._recording
    }
})(document, event, localStorage, utils);
