import {
    sub,
    SETTINGS_CHANGED,
    REFRESH_INPUT,
} from 'event';
import {env} from 'env';
import {input, pointer, keyboard} from 'input';
import {opts, settings} from 'settings';
import {gui} from 'gui';

const rootEl = document.getElementById('screen')
const footerEl = document.getElementsByClassName('screen__footer')[0]

const state = {
    components: [],
    current: undefined,
    forceFullscreen: false,
}

const toggle = async (component, force) => {
    component && (state.current = component) // keep the last component
    state.components.forEach(c => c.toggle(false))
    state.current?.toggle(force)
    state.forceFullscreen && fullscreen(true)
}

const init = () => {
    state.forceFullscreen = settings.loadOr(opts.FORCE_FULLSCREEN, false)
    sub(SETTINGS_CHANGED, () => {
        state.forceFullscreen = settings.get()[opts.FORCE_FULLSCREEN]
    })
}

const cursor = pointer.autoHide(rootEl, 2000)

const trackPointer = pointer.track(rootEl, () => {
    const display = state.current;
    return {...display.video.size, s: !!display?.hasDisplay}
})

const fullscreen = () => {
    if (state.current?.noFullscreen) return

    let h = parseFloat(getComputedStyle(rootEl, null).height.replace('px', ''))
    env.display().toggleFullscreen(h !== window.innerHeight, rootEl)
}

const controls = async (locked = false) => {
    if (!state.current?.hasDisplay) return
    if (env.isMobileDevice) return
    if (!input.kbm) return

    if (locked) {
        await pointer.lock(rootEl)
    }

    // oof, remove hover:hover when the pointer is forcibly locked,
    // leaving the element in the hovered state
    locked ? footerEl.classList.remove('hover') : footerEl.classList.add('hover')

    trackPointer(locked)
    await keyboard.lock(locked)
    input.retropad.toggle(!locked)
}

rootEl.addEventListener('fullscreenchange', async () => {
    const fs = document.fullscreenElement !== null

    cursor(!fs)
    gui.toggle(footerEl, fs)
    await controls(fs)
    state.current?.onFullscreen?.(fs)
})

sub(REFRESH_INPUT, async () => {
    await controls(document.fullscreenElement !== null)
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
