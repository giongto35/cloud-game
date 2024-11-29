import {
    sub,
    APP_VIDEO_CHANGED,
    SETTINGS_CHANGED,
    TRANSFORM_CHANGE
} from 'event';
import {log} from 'log';
import {opts, settings} from 'settings';

const videoEl = document.getElementById('stream')
const mirrorEl = document.getElementById('mirror-stream')
const playEl = document.getElementById('play-stream')

const options = {
    volume: 0.5,
    poster: '/img/screen_loading.gif',
    mirrorMode: null,
    mirrorUpdateRate: 1 / 60,
}

const state = {
    screen: videoEl,
    timerId: null,
    w: 0,
    h: 0,
    aspect: 4 / 3,
    fit: 'contain',
    ready: false,
    autoplayWait: false
}

const mute = (mute) => (videoEl.muted = mute)

const onPlay = () => {
    state.ready = true
    videoEl.poster = ''
    resize(state.w, state.h, state.aspect, state.fit)
    useCustomScreen(options.mirrorMode === 'mirror')
}

const play = () => {
    const promise = videoEl.play()

    if (promise === undefined) {
        log.error('oh no, the video is not a promise!')
        return
    }

    promise
        .then(onPlay)
        .catch(error => {
            if (error.name === 'NotAllowedError') {
                showPlayButton()
            } else {
                log.error('Playback fail', error)
            }
        })
}

const toggle = (show) => state.screen.toggleAttribute('hidden', show === undefined ? show : !show)

const resize = (w, h, aspect, fit) => {
    if (!state.ready) return;

    state.screen.setAttribute('width', '' + w)
    state.screen.setAttribute('height', '' + h)
    aspect !== undefined && (state.screen.style.aspectRatio = '' + aspect)
    fit !== undefined && (state.screen.style['object-fit'] = fit)
}

const showPlayButton = () => {
        state.autoplayWait = true
        toggle()
        playEl.removeAttribute('hidden')
}

playEl.addEventListener('click', () => {
    playEl.setAttribute('hidden', "")
    state.autoplayWait = false
    play()
    toggle()
})

// Track resize even when the underlying media stream changes its video size
videoEl.addEventListener('resize', () => {
    recalculateSize()
    if (state.screen === videoEl) return
    resize(videoEl.videoWidth, videoEl.videoHeight)
})

videoEl.addEventListener('loadstart', () => {
    videoEl.volume = options.volume / 100
    videoEl.poster = options.poster
})

videoEl.onfocus = () => videoEl.blur()
videoEl.onerror = (e) => log.error('Playback error', e)

const onFullscreen = (fullscreen) => {
    const el = document.fullscreenElement

    if (fullscreen) {
        // timeout is due to a chrome bug
        setTimeout(() => {
            // aspect ratio calc
            const w = window.screen.width ?? window.innerWidth
            const hh = el.innerHeight || el.clientHeight || 0
            const dw = (w - hh * state.aspect) / 2
            state.screen.style.padding = `0 ${dw}px`
            state.screen.classList.toggle('with-footer')
        }, 1)
    } else {
        state.screen.style.padding = '0'
        state.screen.classList.toggle('with-footer')
    }

    if (el === videoEl) {
        videoEl.classList.toggle('no-media-controls', !fullscreen)
        videoEl.blur()
    }
}

const vs = {w: 1, h: 1}

const recalculateSize = () => {
    const fullscreen = document.fullscreenElement !== null
    const {aspect, screen} = state

    let width, height
    if (fullscreen) {
        // we can't get the real <video> size on the screen without the black bars,
        // so we're using one side (without the bar) for the length calc of another
        const horizontal = videoEl.videoWidth > videoEl.videoHeight
        width = horizontal ? aspect * screen.offsetHeight : screen.offsetWidth
        height = horizontal ? screen.offsetHeight : aspect * screen.offsetWidth
    } else {
        ({width, height} = screen.getBoundingClientRect())
    }

    vs.w = width
    vs.h = height
}

const useCustomScreen = (use) => {
    if (use) {
        if (videoEl.paused || videoEl.ended) return

        if (state.screen === mirrorEl) return

        toggle(false)
        state.screen = mirrorEl
        resize(videoEl.videoWidth, videoEl.videoHeight)

        // stretch depending on the video orientation
        const isPortrait = videoEl.videoWidth < videoEl.videoHeight
        state.screen.style.width = isPortrait ? 'auto' : videoEl.videoWidth

        let surface = state.screen.getContext('2d')
        state.ready && toggle(true)
        state.timerId = setInterval(function () {
            if (videoEl.paused || videoEl.ended || !surface) return
            surface.drawImage(videoEl, 0, 0)
        }, options.mirrorUpdateRate)
    } else {
        clearInterval(state.timerId)
        toggle(false)
        state.screen = videoEl
        state.ready && toggle(true)
    }
}

const init = () => {
    options.mirrorMode = settings.loadOr(opts.MIRROR_SCREEN, 'none')
    options.volume = settings.loadOr(opts.VOLUME, 50)
    sub(SETTINGS_CHANGED, () => {
        if (settings.changed(opts.MIRROR_SCREEN, options, 'mirrorMode')) {
            useCustomScreen(options.mirrorMode === 'mirror')
        }
        if (settings.changed(opts.VOLUME, options, 'volume')) {
            videoEl.volume = options.volume / 100
        }
    })
}

sub(APP_VIDEO_CHANGED, (payload) => {
    const {w, h, a, s} = payload

    const scale = !s ? 1 : s
    const ww = w * scale
    const hh = h * scale

    state.aspect = a

    const a2 = (ww / hh).toFixed(6)

    state.h = hh
    state.w = Math.floor(hh * a)
    resize(ww, hh, state.aspect, a > 1 && a.toFixed(6) !== a2 ? 'fill' : 'contain')
    recalculateSize()
})

sub(TRANSFORM_CHANGE, () => {
    // cache stream element size when the interface is transformed
    recalculateSize()
})

/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
export const stream = {
    audio: {mute},
    video: {
        el: videoEl,
        get size() {
            return vs
        },
    },
    play,
    showPlayButton,
    toggle,
    hasDisplay: true,
    init,
    onFullscreen,
}
