import {
    pub,
    KEYBOARD_TOGGLE_FILTER_MODE,
    RECORDING_TOGGLED
} from 'event';
import {throttle} from 'utils';

export const RECORDING_ON = 1;
export const RECORDING_OFF = 0;
export const RECORDING_REC = 2;

const userName = document.getElementById('user-name'),
    recButton = document.getElementById('btn-rec');

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
const saveUserName = throttle(() => {
    state.userName = userName.value;
    saveLastState();
}, 500)

let _recording = {
    isActive: () => false,
    getUser: () => '',
    setIndicator: () => ({}),
}

if (userName && recButton) {
    restoreLastState();
    setIndicator(false);
    setRec(state.state === RECORDING_ON)

    // text
    userName.addEventListener('focus', () => pub(KEYBOARD_TOGGLE_FILTER_MODE))
    userName.addEventListener('blur', () => pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true}))
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
        pub(RECORDING_TOGGLED, {userName: state.userName, recording: active})
    })

    _recording = {
        isActive: () => state.state > 0,
        getUser: () => state.userName,
        setIndicator,
    }
}

/**
 * Recording module.
 */
export const recording = _recording
