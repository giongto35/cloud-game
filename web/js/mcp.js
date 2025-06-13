import {pub, KEYBOARD_TOGGLE_FILTER_MODE} from 'event';
import {webrtc} from 'network';
import {log} from 'log';

const inputEl = document.getElementById('mcp-cmd');

export const mcp = {
    init: () => {
        if (!inputEl) return;
        inputEl.addEventListener('focus', () => pub(KEYBOARD_TOGGLE_FILTER_MODE));
        inputEl.addEventListener('blur', () => pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true}));
        inputEl.addEventListener('keydown', e => {
            if (e.key === 'Enter') {
                e.preventDefault();
                webrtc.mcp(inputEl.value);
                inputEl.value = '';
            }
        });
        log.info('[mcp] initialized');
    }
};
