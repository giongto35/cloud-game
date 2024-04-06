import {opts, settings} from 'settings';
import {SETTINGS_CHANGED, sub} from "event";
import {env} from "env";

const rootEl = document.getElementById('screen');

const state = {
    components: [],
    current: undefined,
    forceFullscreen: false,
}

const toggle = (component, force) => {
    component && (state.current = component); // keep the last component
    state.components.forEach(c => c.toggle(false));
    state.current?.toggle(force);
    component && !env.isMobileDevice && !state.current?.noFullscreen && state.forceFullscreen && fullscreen();
}

const init = () => {
    state.forceFullscreen = settings.loadOr(opts.FORCE_FULLSCREEN, false);
    sub(SETTINGS_CHANGED, () => {
        state.forceFullscreen = settings.get()[opts.FORCE_FULLSCREEN];
    });
}

const fullscreen = () => {
    let h = parseFloat(getComputedStyle(rootEl, null)
        .height
        .replace('px', '')
    )
    env.display().toggleFullscreen(h !== window.innerHeight, rootEl);
}

rootEl.addEventListener('fullscreenchange', () => {
    state.current?.onFullscreen?.(document.fullscreenElement !== null)
})

export const screen = {
    fullscreen,
    toggle,
    /**
     * Adds a component. It should have toggle(bool) method and
     * an optional noFullscreen (bool) property.
     */
    add: (...o) => state.components.push(...o),
    init,
}
