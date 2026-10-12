/**
 * Spatial navigation: lets a gamepad (or the arrow keys) move the focus
 * between interactive elements by screen position.
 */

const FOCUSABLE = 'button:not([disabled]), a[href], input:not([disabled]):not([type="hidden"]), select, textarea, [tabindex]:not([tabindex="-1"]), [data-nav]';

const scopes = [];

const currentScope = () => scopes[scopes.length - 1] || document.body;

const visible = (el) => {
    if (el.closest('[hidden], .hidden')) return false;
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
};

const candidates = () => [...currentScope().querySelectorAll(FOCUSABLE)].filter(visible);

let marked = null;
const mark = (el) => {
    marked?.classList.remove('nav-focus');
    marked = el;
    el?.classList.add('nav-focus');
};
document.addEventListener('pointerdown', () => mark(null), true);
document.addEventListener('focusout', (e) => e.target === marked && mark(null));

const focus = (el, visual = true) => {
    if (!el) return;
    el.focus({preventScroll: true});
    el.scrollIntoView({block: 'nearest', inline: 'nearest', behavior: 'smooth'});
    if (visual) mark(el);
};

/** Moves the focus in the given direction: 'up' | 'down' | 'left' | 'right'. */
const move = (dir, visual = true) => {
    const list = candidates();
    if (!list.length) return;
    const active = document.activeElement;
    if (!active || active === document.body || !list.includes(active)) {
        focus(list[0], visual);
        return;
    }
    const a = active.getBoundingClientRect();
    const ax = a.left + a.width / 2, ay = a.top + a.height / 2;
    let best = null, bestScore = Infinity;
    for (const el of list) {
        if (el === active) continue;
        const r = el.getBoundingClientRect();
        const x = r.left + r.width / 2, y = r.top + r.height / 2;
        const dx = x - ax, dy = y - ay;
        let main, cross;
        switch (dir) {
            case 'up': main = -dy; cross = Math.abs(dx); break;
            case 'down': main = dy; cross = Math.abs(dx); break;
            case 'left': main = -dx; cross = Math.abs(dy); break;
            case 'right': main = dx; cross = Math.abs(dy); break;
        }
        if (main <= 1) continue;
        const score = main + cross * 2.2;
        if (score < bestScore) {
            bestScore = score;
            best = el;
        }
    }
    if (best) focus(best, visual);
};

/** Clicks the focused element. */
const activate = () => {
    const el = document.activeElement;
    if (!el || el === document.body) {
        move('down');
        return;
    }
    if (el.matches('input[type="checkbox"], input[type="radio"]')) {
        el.click();
        return;
    }
    if (el.matches('input, textarea')) {
        el.focus();
        return;
    }
    el.click();
};

export const nav = {
    move,
    activate,
    focus,
    /** Restricts navigation to an element (e.g. an open modal). */
    pushScope: (el) => scopes.push(el),
    popScope: (el) => {
        const i = scopes.lastIndexOf(el);
        if (i >= 0) scopes.splice(i, 1);
    },
};
