import {env} from 'env';
import {
    pub,
    sub,
    APP_VIDEO_CHANGED,
    FULLSCREEN_CHANGE,
    KB_MOUSE_FLAG,
    MOUSE_MOVED,
    MOUSE_PRESSED,
    POINTER_LOCK_CHANGE,
    SETTINGS_CHANGED
} from 'event' ;
import {gui} from 'gui';
import {log} from 'log';
import {opts, settings} from 'settings';

const screen = document.getElementById('stream');

let options = {
        volume: 0.5,
        poster: '/img/screen_loading.gif',
        mirrorMode: null,
        mirrorUpdateRate: 1 / 60,
        forceFullscreen: true,
    },
    state = {
        screen: screen,
        fullscreen: false,
        kbmLock: false,
        timerId: null,
        w: 0,
        h: 0,
        aspect: 4 / 3
    };

const mute = (mute) => screen.muted = mute

const _stream = () => {
    screen.play()
        .then(() => log.info('Media can autoplay'))
        .catch(error => {
            log.error('Media failed to play', error);
        });
}

const toggle = (show) => {
    state.screen.toggleAttribute('hidden', !show)
}

const toggleFullscreen = () => {
    let h = parseFloat(getComputedStyle(state.screen, null)
        .height
        .replace('px', '')
    )
    env.display().toggleFullscreen(h !== window.innerHeight, state.screen);
}

const getVideoEl = () => screen

const getActualVideoSize = () => {
    if (state.fullscreen) {
        // we can't get real <video> size without black bars, so we're trying to
        // derive its dimensions from the known width or height
        // and by calculating unknown dimension from the aspect ratio
        const horizontal = screen.videoWidth > screen.videoHeight;
        return {
            w: horizontal ? screen.offsetHeight * state.aspect : screen.offsetWidth,
            h: horizontal ? screen.offsetHeight : screen.offsetWidth * state.aspect
        }
    }

    const size = screen.getBoundingClientRect()

    return {w: size.width, h: size.height}
}

screen.onerror = (e) => {
    // video playback failed - show a message saying why
    switch (e.target.error.code) {
        case e.target.error.MEDIA_ERR_ABORTED:
            log.error('You aborted the video playback.');
            break;
        case e.target.error.MEDIA_ERR_NETWORK:
            log.error('A network error caused the video download to fail part-way.');
            break;
        case e.target.error.MEDIA_ERR_DECODE:
            log.error('The video playback was aborted due to a corruption problem or because the video used features your browser did not support.');
            break;
        case e.target.error.MEDIA_ERR_SRC_NOT_SUPPORTED:
            log.error('The video could not be loaded, either because the server or network failed or because the format is not supported.');
            break;
        default:
            log.error('An unknown video error occurred.');
            break;
    }
};

screen.addEventListener('loadedmetadata', () => {
    if (state.screen !== screen) {
        state.screen.setAttribute('width', screen.videoWidth);
        state.screen.setAttribute('height', screen.videoHeight);
    }
}, false);
screen.addEventListener('loadstart', () => {
    screen.volume = options.volume;
    screen.poster = options.poster;
}, false);
screen.addEventListener('canplay', () => {
    screen.poster = '';
    useCustomScreen(options.mirrorMode === 'mirror');
}, false);

screen.addEventListener('focus', () => {
    screen.blur();
})

const handlePointerDown = (e) => pub(MOUSE_PRESSED, {b: e.button, p: true});
const handlePointerUp = (e) => pub(MOUSE_PRESSED, {b: e.button, p: false});

const makeFullscreen = (make = false) => {
    screen.classList.toggle('no-media-controls', make)
}

const forceFullscreenMaybe = () => {
    const touchMode = env.isMobileDevice();
    log.debug('touch check', touchMode)
    !touchMode && options.forceFullscreen && toggleFullscreen();
}

const useCustomScreen = (use) => {
    if (use) {
        if (screen.paused || screen.ended) return;

        let id = state.screen.getAttribute('id');
        if (id === 'canvas-mirror') return;

        const canvas = gui.create('canvas');
        canvas.setAttribute('id', 'canvas-mirror');
        canvas.setAttribute('hidden', '');
        canvas.setAttribute('width', screen.videoWidth);
        canvas.setAttribute('height', screen.videoHeight);
        canvas.style['image-rendering'] = 'pixelated';
        canvas.style.width = '100%'
        canvas.style.height = '100%'
        canvas.classList.add('game-screen');

        // stretch depending on the video orientation
        // portrait -- vertically, landscape -- horizontally
        const isPortrait = screen.videoWidth < screen.videoHeight;
        canvas.style.width = isPortrait ? 'auto' : canvas.style.width;
        // canvas.style.height = isPortrait ? canvas.style.height : 'auto';

        let surface = canvas.getContext('2d');
        screen.parentNode.insertBefore(canvas, screen.nextSibling);
        toggle(false)
        state.screen = canvas
        toggle(true)
        state.timerId = setInterval(function () {
            if (screen.paused || screen.ended || !surface) return;
            surface.drawImage(screen, 0, 0);
        }, options.mirrorUpdateRate);
    } else {
        clearInterval(state.timerId);
        let mirror = state.screen;
        state.screen = screen;
        toggle(true);
        if (mirror !== screen) {
            mirror.parentNode.removeChild(mirror);
        }
    }
}

const init = () => {
    options.mirrorMode = settings.loadOr(opts.MIRROR_SCREEN, 'none');
    options.volume = settings.loadOr(opts.VOLUME, 50) / 100;
    options.forceFullscreen = settings.loadOr(opts.FORCE_FULLSCREEN, false);
}

sub(SETTINGS_CHANGED, () => {
    const s = settings.get();
    const newValue = s[opts.MIRROR_SCREEN];
    if (newValue !== options.mirrorMode) {
        useCustomScreen(newValue === 'mirror');
        options.mirrorMode = newValue;
    }
    const newValue2 = s[opts.FORCE_FULLSCREEN];
    if (newValue2 !== options.forceFullscreen) {
        options.forceFullscreen = newValue2;
    }
});

let pointerLocked = false;

const screenToAspect = (el) => {
    const w = window.screen.width ?? window.innerWidth;
    const {w: ww} = getActualVideoSize();
    const dw = (w - ww) / 2;
    screen.style.padding = `0 ${dw}px`
}

sub(FULLSCREEN_CHANGE, async (fullscreenEl) => {
    state.fullscreen = !!fullscreenEl;

    if (!state.fullscreen) {
        screen.style.padding = '0'
    } else {
        screenToAspect(fullscreenEl);
        // chrome bug
        setTimeout(() => {
            screenToAspect(fullscreenEl);
        }, 1)
    }
    makeFullscreen(state.fullscreen);

    screen.blur();

    if (!state.kbmLock) return;

    if (state.fullscreen && !pointerLocked) {
        // event.pub(POINTER_LOCK_CHANGE, screen);
        await screen.requestPointerLock(
            // { unadjustedMovement: true,}
        );
    }

    screen.onpointerdown = state.fullscreen ? handlePointerDown : null;
    screen.onpointerup = state.fullscreen ? handlePointerUp : null;

    // !to flipped
})

let ex = 0, ey = 0;
const scaleCursorPos = (x, y) => {
    const {w, h} = getActualVideoSize();

    const sw = w / screen.videoWidth;
    const sh = h / screen.videoHeight;

    const rez = {
        dx: x / sw + ex,
        dy: y / sh + ey
    }

    ex = rez.dx % 1;
    ey = rez.dy % 1;

    rez.dx -= ex;
    rez.dy -= ey;

    return rez;
}

const handlePointerMove = (e) => {
    // !to fix ff https://github.com/w3c/pointerlock/issues/42
    pub(MOUSE_MOVED, scaleCursorPos(e.movementX, e.movementY));
}

const fit = 'contain'

sub(APP_VIDEO_CHANGED, (payload) => {
    const {w, h, a, s} = payload

    const scale = !s ? 1 : s;
    const ww = w * scale;
    const hh = h * scale;

    state.aspect = a

    const a2 = (ww / hh).toFixed(6)

    state.screen.style['object-fit'] = a > 1 && a.toFixed(6) !== a2 ? 'fill' : fit
    state.h = hh
    state.w = Math.floor(hh * a)
    state.screen.setAttribute('width', '' + ww)
    state.screen.setAttribute('height', '' + hh)
    state.screen.style.aspectRatio = '' + state.aspect
})

sub(KB_MOUSE_FLAG, () => {
    log.info('Keyboard and mouse will be locked in fullscreen');
    state.kbmLock = true;
    sub(POINTER_LOCK_CHANGE, (lockedEl) => {
        pointerLocked = lockedEl === screen;
        screen.onpointermove = pointerLocked ? handlePointerMove : null;
        log.debug(`Pointer lock: ${pointerLocked}`);
    });
})

/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
export const stream = {
    audio: {mute},
    video: {toggleFullscreen, el: getVideoEl},
    play: _stream,
    toggle,
    useCustomScreen,
    forceFullscreenMaybe,
    init
}
