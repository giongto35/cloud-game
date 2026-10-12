import {icon} from './icons.js?v=4';
import {esc} from './dom.js?v=4';

const icons = {info: 'info', success: 'check', error: 'alert', warning: 'alert'};
let root;
const recent = new Map();

const ensureRoot = () => {
    if (root) return root;
    root = document.createElement('div');
    root.className = 'toasts';
    root.setAttribute('role', 'status');
    root.setAttribute('aria-live', 'polite');
    document.body.append(root);
    return root;
};

/**
 * Shows a short notification.
 * @param {string} text
 * @param {'info'|'success'|'error'|'warning'} kind
 * @param {number} ms
 */
export const toast = (text, kind = 'info', ms = 3200) => {
    // drop duplicates shown in the last second
    const now = Date.now();
    if (now - (recent.get(text) || 0) < 1000) return;
    recent.set(text, now);

    const el = document.createElement('div');
    el.className = `toast ${kind}`;
    el.innerHTML = `${icon(icons[kind] || 'info')}<span>${esc(text)}</span>`;
    const r = ensureRoot();
    r.append(el);
    while (r.children.length > 4) r.firstElementChild.remove();
    setTimeout(() => {
        el.classList.add('leaving');
        el.addEventListener('animationend', () => el.remove(), {once: true});
    }, ms);
};

toast.success = (t, ms) => toast(t, 'success', ms);
toast.error = (t, ms) => toast(t, 'error', ms ?? 4500);
toast.warning = (t, ms) => toast(t, 'warning', ms ?? 4000);
