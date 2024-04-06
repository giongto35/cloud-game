import {gui} from 'gui';
import {
    sub,
    MENU_HANDLER_ATTACHED,
} from 'event';

const rootEl = document.getElementById('menu-screen');

// touch stuff
sub(MENU_HANDLER_ATTACHED, (data) => {
    rootEl.addEventListener(data.event, data.handler, {passive: true});
});

export const menu = {
    toggle: (show) => show === undefined ? gui.toggle(rootEl) : gui.toggle(rootEl, show),
    noFullscreen: true,
}
