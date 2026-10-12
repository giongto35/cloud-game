import {esc, hue} from './dom.js?v=4';
import {gameTitle, systemShort} from '../i18n.js?v=4';

/** A short display glyph for a title: initials of the first words. */
const glyph = (title) => {
    const words = title.replace(/[^\p{L}\p{N}\s]/gu, ' ').split(/\s+/).filter(Boolean);
    if (!words.length) return '?';
    if (words.length === 1) return words[0].slice(0, 3).toUpperCase();
    return words.slice(0, 3).map((w) => w[0]).join('').toUpperCase();
};

/**
 * Generated cover art (unique gradient per game + system badge).
 * @param {string} title raw game title
 * @param {string} system emulator system id
 * @param {string} extra additional markup placed at the top
 */
export const cover = (title, system, extra = '') => {
    const {title: clean} = gameTitle(title);
    const h1 = hue(title);
    const h2 = (h1 + 40 + (hue(system) % 80)) % 360;
    const src = coverUrl(title, system);
    const art = failed.has(src) ? '' :
        `<img class="art-bg" src="${esc(src)}" alt="" aria-hidden="true" loading="lazy" decoding="async">
        <img class="art" src="${esc(src)}" alt="" loading="lazy" decoding="async">`;
    return `<div class="cover ${loaded.has(src) ? 'has-art' : ''}" style="--h1:${h1};--h2:${h2}">
        ${art}
        ${extra || `<span class="sys">${esc(systemShort(system))}</span>`}
        <span class="glyph">${esc(glyph(clean))}</span>
    </div>`;
};

/** Server URL of the stored cover art (fetched once from libretro-thumbnails). */
export const coverUrl = (title, system) =>
    `/api/covers?system=${encodeURIComponent(system || '')}&game=${encodeURIComponent(title)}`;

// remember results so re-rendered cards don't flash or retry missing pictures
const loaded = new Set();
const failed = new Set();

document.addEventListener('load', (e) => {
    const img = e.target;
    if (!(img instanceof HTMLImageElement) || !img.classList.contains('art')) return;
    loaded.add(img.getAttribute('src'));
    img.closest('.cover')?.classList.add('has-art');
}, true);

document.addEventListener('error', (e) => {
    const img = e.target;
    if (!(img instanceof HTMLImageElement) || !img.closest('.cover')) return;
    failed.add(img.getAttribute('src'));
    const c = img.closest('.cover');
    c.querySelectorAll('img').forEach((i) => i.remove());
    c.classList.remove('has-art');
}, true);
