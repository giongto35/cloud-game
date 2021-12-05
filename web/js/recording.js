const RECORDING_ON = 1;
const RECORDING_OFF = 0;
const RECORDING_REC = 2;

/**
 * Recording module.
 * @version 1
 */
const recording = (() => {
    const userName = document.getElementById('user-name'),
        recButton = document.getElementById('btn-rec');

    if (!userName || !recButton) {
        return {
            isActive: () => false,
            getUser: () => '',
        }
    }

    let state = {
        userName: '',
        state: RECORDING_OFF,
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

    const setRec = (val) => {
        recButton.classList.toggle('record', val);
    }
    const setIndicator = (val) => {
        recButton.classList.toggle('blink', val);
    };

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
    setIndicator(false);
    setRec(state.state === RECORDING_ON)

    // text
    userName.addEventListener('focus', () => event.pub(KEYBOARD_TOGGLE_FILTER_MODE))
    userName.addEventListener('blur', () => event.pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true}))
    userName.addEventListener('keyup', ev => {
        ev.stopPropagation();
        saveUserName()
    })

    // button
    recButton.addEventListener('click', () => {
        state.state = (state.state + 1) % 2
        const active = state.state === RECORDING_ON
        setRec(active)
        saveLastState()
        event.pub(RECORDING_TOGGLED, {userName: state.userName, recording: active})
    })
    return {
        isActive: () => state.state > 0,
        getUser: () => state.userName,
        setIndicator: setIndicator,
    }
})(document, event, localStorage, utils);
