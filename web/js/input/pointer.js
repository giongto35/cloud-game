import { pub, MOUSE_PRESSED, MOUSE_MOVED } from 'event';
import { browser, env } from 'env';

const hasRawPointer = 'onpointerrawupdate' in window;
const isFirefox = env.getBrowser === browser.firefox;
const moveEvent = hasRawPointer ? 'pointerrawupdate' : 'pointermove';

// Reusable event data objects to avoid allocations
const move = { dx: 0, dy: 0 };
const btn = { b: null, p: false };

// Game resolution for DPI scaling
const gameW = 640;
const gameH = 480;

// Accumulates fractional pixels to prevent drift when scaling
let errX = 0;
let errY = 0;

const scaleDpi = (dx, dy, srcW, srcH) => {
    move.dx = dx / (srcW / gameW) + errX;
    move.dy = dy / (srcH / gameH) + errY;
    errX = move.dx % 1;
    errY = move.dy % 1;
    move.dx = Math.trunc(move.dx);
    move.dy = Math.trunc(move.dy);
    return move;
};

const onDown = (e) => { btn.b = e.button; btn.p = true; pub(MOUSE_PRESSED, btn); };
const onUp = (e) => { btn.b = e.button; btn.p = false; pub(MOUSE_PRESSED, btn); };

/*
 * Tracks pointer movement and publishes events.
 * Uses raw pointer events when available for better accuracy.
 * Coalesced events are broken in Firefox 120+, so we skip them there.
 */
const track = (el, getDisplay) => {
    let off = null;

    const handle = (e) => {
        // Firefox workaround: skip coalesced events
        const events = isFirefox ? [e] : (e.getCoalescedEvents?.() || [e]);
        const { w, h, s } = getDisplay();

        for (const ev of events) {
            move.dx = ev.movementX;
            move.dy = ev.movementY;
            pub(MOUSE_MOVED, s ? scaleDpi(move.dx, move.dy, w, h) : move);
        }
    };

    return (on) => {
        if (on && !off) {
            el.addEventListener(moveEvent, handle);
            el.onpointerdown = onDown;
            el.onpointerup = onUp;
            off = () => {
                el.removeEventListener(moveEvent, handle);
                el.onpointerdown = null;
                el.onpointerup = null;
            };
        } else if (!on && off) {
            off();
            off = null;
        }
    };
};

/*
 * Auto-hides cursor after inactivity. Movement shows it again.
 */
const autoHide = (el, timeout = 3000) => {
    let timer;
    const cl = el.classList;
    const reset = () => {
        cl.remove('no-pointer');
        clearTimeout(timer);
        timer = setTimeout(() => cl.add('no-pointer'), timeout);
    };

    return (on) => {
        clearTimeout(timer);
        el.removeEventListener('pointermove', reset);
        cl.remove('no-pointer');
        on || (el.addEventListener('pointermove', reset), reset());
    };
};

export const pointer = {
    lock: (el) => el.requestPointerLock(),
    track,
    autoHide,
};