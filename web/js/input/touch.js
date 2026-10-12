import {pub, sub, AXIS_CHANGED, DPAD_TOGGLE, KEY_PRESSED, KEY_RELEASED} from 'event';
import {KEY} from 'input';
import {log} from 'log';

/**
 * On-screen (touch) gamepad.
 * Multi-touch through pointer events; the stick works as a d-pad or as an analog stick.
 */

let root = null;
let dpadMode = true;
let enabled = true;
const deadZone = 0.12;

const held = new Set();
const press = (key, down) => {
    if (!enabled && down) return;
    if (down === held.has(key)) return;
    down ? held.add(key) : held.delete(key);
    pub(down ? KEY_PRESSED : KEY_RELEASED, {key, touch: true});
};

const stickDirs = {[KEY.LEFT]: false, [KEY.RIGHT]: false, [KEY.UP]: false, [KEY.DOWN]: false};
let axes = [0, 0];

const setDir = (key, on) => {
    if (stickDirs[key] === on) return;
    stickDirs[key] = on;
    press(key, on);
};

const setAxis = (id, v) => {
    if (Math.abs(v) < deadZone) v = 0;
    if (axes[id] === v) return;
    axes[id] = v;
    pub(AXIS_CHANGED, {id, value: v});
};

const resetStick = () => {
    Object.keys(stickDirs).forEach((k) => setDir(k, false));
    setAxis(0, 0);
    setAxis(1, 0);
};

const markup = `
    <div class="tp-stick" data-stick><div class="knob"></div></div>
    <div class="tp-shoulder left"><div class="tp-pill" data-key="l2">L2</div><div class="tp-pill" data-key="l">L</div></div>
    <div class="tp-shoulder right"><div class="tp-pill" data-key="r">R</div><div class="tp-pill" data-key="r2">R2</div></div>
    <div class="tp-buttons">
        <div class="tp-btn" data-key="x">X</div>
        <div class="tp-btn" data-key="y">Y</div>
        <div class="tp-btn" data-key="a">A</div>
        <div class="tp-btn" data-key="b">B</div>
    </div>
    <div class="tp-center"><div class="tp-pill" data-key="select">SELECT</div><div class="tp-pill" data-key="start">START</div></div>`;

const bindButton = (el) => {
    const key = el.dataset.key;
    const down = (e) => {
        e.preventDefault();
        el.setPointerCapture?.(e.pointerId);
        el.classList.add('pressed');
        press(key, true);
        navigator.vibrate?.(8);
    };
    const up = () => {
        el.classList.remove('pressed');
        press(key, false);
    };
    el.addEventListener('pointerdown', down);
    el.addEventListener('pointerup', up);
    el.addEventListener('pointercancel', up);
    el.addEventListener('lostpointercapture', up);
};

const bindStick = (el) => {
    const knob = el.querySelector('.knob');
    let pid = null;

    const update = (e) => {
        const r = el.getBoundingClientRect();
        const max = r.width * 0.32;
        let dx = e.clientX - (r.left + r.width / 2);
        let dy = e.clientY - (r.top + r.height / 2);
        const dist = Math.hypot(dx, dy);
        if (dist > max) {
            dx = dx / dist * max;
            dy = dy / dist * max;
        }
        knob.style.transform = `translate(${dx}px, ${dy}px)`;
        const x = dx / max, y = dy / max;
        if (dpadMode) {
            setDir(KEY.LEFT, x <= -0.45);
            setDir(KEY.RIGHT, x >= 0.45);
            setDir(KEY.UP, y <= -0.45);
            setDir(KEY.DOWN, y >= 0.45);
        } else {
            setAxis(0, x);
            setAxis(1, y);
        }
    };

    el.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        pid = e.pointerId;
        el.setPointerCapture?.(pid);
        knob.style.transition = 'none';
        update(e);
    });
    el.addEventListener('pointermove', (e) => e.pointerId === pid && update(e));
    const end = (e) => {
        if (e.pointerId !== pid) return;
        pid = null;
        knob.style.transition = 'transform .15s';
        knob.style.transform = '';
        resetStick();
    };
    el.addEventListener('pointerup', end);
    el.addEventListener('pointercancel', end);
};

sub(DPAD_TOGGLE, (data) => {
    if (dpadMode === data.checked) return;
    resetStick();
    dpadMode = data.checked;
});

export const isTouchDevice = () => matchMedia('(pointer: coarse)').matches || navigator.maxTouchPoints > 0;

export const touch = {
    /** Builds the on-screen gamepad inside the container. */
    init: (container) => {
        if (!container || root) return;
        root = container;
        root.innerHTML = markup;
        root.querySelectorAll('[data-key]').forEach(bindButton);
        bindStick(root.querySelector('[data-stick]'));
        root.addEventListener('contextmenu', (e) => e.preventDefault());
        log.info('[input] touch input has been initialized');
    },
    show: (v) => root && (root.hidden = !v),
    style: ({opacity, scale}) => {
        if (!root) return;
        opacity !== undefined && root.style.setProperty('--tp-opacity', opacity);
        scale !== undefined && root.style.setProperty('--tp-scale', scale);
    },
    toggle: (v) => {
        enabled = v === undefined ? !enabled : v;
        if (!enabled) {
            [...held].forEach((k) => press(k, false));
            resetStick();
        }
    },
};
