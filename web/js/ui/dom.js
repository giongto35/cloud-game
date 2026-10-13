/** Small DOM helpers. */

const ESC = {'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'};

/** Escapes text for safe interpolation into HTML. */
export const esc = (v) => String(v ?? '').replace(/[&<>"']/g, (c) => ESC[c]);

/** Creates an element from an HTML string (first element). */
export const html = (markup) => {
    const t = document.createElement('template');
    t.innerHTML = markup.trim();
    return t.content.firstElementChild;
};

export const $ = (sel, root = document) => root.querySelector(sel);
export const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

/** Event delegation: on(root, 'click', '[data-action]', (e, el) => ...) */
export const on = (root, type, selector, fn, opts) =>
    root.addEventListener(type, (e) => {
        const el = e.target.closest(selector);
        if (el && root.contains(el)) fn(e, el);
    }, opts);

/** Stable hue (0-359) for a string. */
export const hue = (str = '') => {
    let h = 0;
    for (let i = 0; i < str.length; i++) h = (h * 31 + str.charCodeAt(i)) >>> 0;
    return h % 360;
};

export const initials = (name = '') => {
    const parts = name.trim().split(/\s+/).filter(Boolean);
    if (!parts.length) return '?';
    return (parts[0][0] + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase();
};

// profile pictures live on the server (/api/avatars/<account id>); users without one keep their initials
const avatarVersions = new Map();
export const avatarUrl = (id) => `/api/avatars/${id}${avatarVersions.has(id) ? `?v=${avatarVersions.get(id)}` : ''}`;
/** Forces the browser to fetch a new picture of an account (after an upload). */
export const bumpAvatar = (id) => avatarVersions.set(id, Date.now());

export const avatar = (name, cls = '', id = null) =>
    `<span class="avatar ${cls}" style="--hue:${hue(name)}" title="${esc(name)}">${esc(initials(name))}${id ? `<img src="${esc(avatarUrl(id))}" alt="" loading="lazy" decoding="async">` : ''}</span>`;

// no picture (404): fall back to the initials
document.addEventListener('error', (e) => {
    const img = e.target;
    if (img instanceof HTMLImageElement && img.parentElement?.classList.contains('avatar')) img.remove();
}, true);

/** Relative time in pt-BR ("há 5 min"). */
export const ago = (date) => {
    const t = typeof date === 'number' ? date : new Date(date).getTime();
    if (!t) return '—';
    const s = Math.max(0, (Date.now() - t) / 1000);
    if (s < 45) return 'agora mesmo';
    if (s < 3600) return `há ${Math.round(s / 60)} min`;
    if (s < 86400) return `há ${Math.round(s / 3600)} h`;
    if (s < 86400 * 30) return `há ${Math.round(s / 86400)} d`;
    return new Date(t).toLocaleDateString('pt-BR');
};

export const dateTime = (date) => date ? new Date(date).toLocaleString('pt-BR', {
    day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit'
}) : '—';

/** True when the event target is a text field (typing should not trigger shortcuts). */
export const isTyping = (target = document.activeElement) => {
    if (!target) return false;
    const tag = target.tagName;
    return target.isContentEditable || tag === 'TEXTAREA' || tag === 'SELECT' ||
        (tag === 'INPUT' && !['checkbox', 'radio', 'range', 'button', 'submit'].includes(target.type));
};

export const copyText = async (text) => {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        const el = document.createElement('textarea');
        el.value = text;
        el.style.position = 'fixed';
        el.style.opacity = '0';
        document.body.append(el);
        el.select();
        const ok = document.execCommand('copy');
        el.remove();
        return ok;
    }
};
