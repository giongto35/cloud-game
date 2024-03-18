// Pointer (aka mouse) stuff
import {
    MOUSE_PRESSED,
    MOUSE_MOVED,
    pub
} from 'event';
import {browser, env} from 'env';

const hasRawPointer = 'onpointerrawupdate' in window

const p = {dx: 0, dy: 0}

const move = (e, cb, single = false) => {
    // !to fix ff https://github.com/w3c/pointerlock/issues/42
    if (single) {
        p.dx = e.movementX
        p.dy = e.movementY
        cb(p)
    } else {
        const _events = e.getCoalescedEvents?.()
        if (_events && (hasRawPointer || _events.length > 1)) {
            for (let i = 0; i < _events.length; i++) {
                p.dx = _events[i].movementX
                p.dy = _events[i].movementY
                cb(p)
            }
        }
    }
}

const _track = (el, cb, single) => {
    const _move = (e) => {
        move(e, cb, single)
    }
    el.addEventListener(hasRawPointer ? 'pointerrawupdate' : 'pointermove', _move)
    return () => {
        el.removeEventListener(hasRawPointer ? 'pointerrawupdate' : 'pointermove', _move)
    }
}

const dpiScaler = () => {
    let ex = 0
    let ey = 0
    let scaled = {dx: 0, dy: 0}
    return {
        scale(x, y, src_w, src_h, dst_w, dst_h) {
            scaled.dx = x / (src_w / dst_w) + ex
            scaled.dy = y / (src_h / dst_h) + ey

            ex = scaled.dx % 1
            ey = scaled.dy % 1

            scaled.dx -= ex
            scaled.dy -= ey

            return scaled
        }
    }
}

const dpi = dpiScaler()

const handlePointerMove = (el, cb) => {
    let w, h = 0
    let s = false
    const dw = 640, dh = 480
    return (p) => {
        ({w, h, s} = cb())
        pub(MOUSE_MOVED, s ? dpi.scale(p.dx, p.dy, w, h, dw, dh) : p)
    }
}

const trackPointer = (el, cb) => {
    let mpu, mpd
    let noTrack

    // disable coalesced mouse move events
    const single = true

    // coalesced event are broken since FF 120
    const isFF = env.getBrowser === browser.firefox

    const pm = handlePointerMove(el, cb)

    return (enabled) => {
        if (enabled) {
            !noTrack && (noTrack = _track(el, pm, isFF || single))
            mpu = pointer.handle.up(el)
            mpd = pointer.handle.down(el)
            return
        }

        mpu?.()
        mpd?.()
        noTrack?.()
        noTrack = null
    }
}

const handleDown = ((b = {b: null, p: true}) => (e) => {
    b.b = e.button
    pub(MOUSE_PRESSED, b)
})()

const handleUp = ((b = {b: null, p: false}) => (e) => {
    b.b = e.button
    pub(MOUSE_PRESSED, b)
})()

const autoHide = (el, time = 3000) => {
    let tm
    let move
    const cl = el.classList

    const hide = (force = false) => {
        cl.add('no-pointer')
        !force && el.addEventListener('pointermove', move)
    }

    move = () => {
        cl.remove('no-pointer')
        clearTimeout(tm)
        tm = setTimeout(hide, time)
    }

    const show = () => {
        clearTimeout(tm)
        el.removeEventListener('pointermove', move)
        cl.remove('no-pointer')
    }

    return {
        autoHide: (on) => on ? show() : hide()
    }
}

export const pointer = {
    autoHide,
    lock: async (el) => {
        await el.requestPointerLock(/*{ unadjustedMovement: true}*/)
    },
    track: trackPointer,
    handle: {
        down: (el) => {
            el.onpointerdown = handleDown
            return () => (el.onpointerdown = null)
        },
        up: (el) => {
            el.onpointerup = handleUp
            return () => (el.onpointerup = null)
        }
    }
}
