import {pub, SETTINGS_CHANGED} from 'event';
import {log} from 'log';

/**
 * App-wide option names.
 */
export const opts = {
    _VERSION: '_version',
    LOG_LEVEL: 'log.level',
    INPUT_KEYBOARD_MAP: 'input.keyboard.map',
    INPUT_GAMEPAD_MAP: 'input.gamepad.map',
    INPUT_DEADZONE: 'input.deadzone',
    INPUT_DPAD_MODE: 'input.dpad',
    TOUCH_MODE: 'touch.mode',
    TOUCH_OPACITY: 'touch.opacity',
    TOUCH_SCALE: 'touch.scale',
    MIRROR_SCREEN: 'mirror.screen',
    VIDEO_PIXEL: 'video.pixel',
    VOLUME: 'volume',
    FORCE_FULLSCREEN: 'force.fullscreen',
    WEBRTC_INIT_OFFER: 'webrtc.init.offer',
    SHOW_STATS: 'hud.stats',
    LIB_FAVORITES: 'library.favorites',
    LIB_RECENT: 'library.recent',
};

// options that are not synced with the account (device specific)
const localOnly = new Set([opts.LOG_LEVEL, opts.WEBRTC_INIT_OFFER, opts.TOUCH_MODE]);

// internal structure version: a bump resets the stored settings
const revision = 2;
const storageKey = 'belko.settings';

let store = {[opts._VERSION]: revision};
const defaults = {};
let remote = null;
let pushTimer = null;

const clone = (v) => (v === undefined ? v : JSON.parse(JSON.stringify(v)));

const persist = () => {
    try {
        localStorage.setItem(storageKey, JSON.stringify(store));
    } catch (e) {
        log.warn('[settings] cannot persist', e);
    }
};

const syncedSnapshot = () => {
    const out = {[opts._VERSION]: revision};
    for (const k of Object.keys(store)) if (!localOnly.has(k) && k !== opts._VERSION) out[k] = store[k];
    return out;
};

const schedulePush = () => {
    if (!remote) return;
    clearTimeout(pushTimer);
    pushTimer = setTimeout(() => remote.save(syncedSnapshot()).catch((e) => log.warn('[settings] sync fail', e)), 1500);
};

const init = () => {
    try {
        const raw = JSON.parse(localStorage.getItem(storageKey) || 'null');
        if (raw && raw[opts._VERSION] === revision) store = raw;
    } catch {
        // broken storage, start over
    }
    // the previous (v1) format is obsolete
    try {
        localStorage.removeItem('settings');
    } catch {
        // ignore
    }
    persist();
};

/** Returns the value or registers and returns the default. */
const loadOr = (key, value) => {
    defaults[key] = clone(value);
    if (!(key in store)) {
        store[key] = clone(value);
        persist();
    }
    return clone(store[key]);
};

const get = (key) => (key === undefined ? store : clone(store[key] ?? defaults[key]));

const set = (key, value, notify = true) => {
    store[key] = clone(value);
    persist();
    if (!localOnly.has(key)) schedulePush();
    notify && pub(SETTINGS_CHANGED, {key});
};

const reset = (key) => set(key, clone(defaults[key]));

const resetAll = () => {
    store = {[opts._VERSION]: revision};
    for (const k of Object.keys(defaults)) store[k] = clone(defaults[k]);
    persist();
    schedulePush();
    pub(SETTINGS_CHANGED, {});
};

/**
 * Connects the settings to the account storage.
 * The account copy wins over the local one on load.
 */
const attachRemote = async ({load, save}) => {
    remote = {save};
    try {
        const data = await load();
        if (data && data[opts._VERSION] === revision) {
            for (const [k, v] of Object.entries(data)) if (!localOnly.has(k)) store[k] = v;
            persist();
            pub(SETTINGS_CHANGED, {});
        } else {
            schedulePush();
        }
    } catch (e) {
        log.warn('[settings] cannot load account preferences', e);
    }
};

/** Copies a changed option into obj[field]; returns true if it was changed. */
const changed = (key, obj, field) => {
    const v = store[key];
    if (JSON.stringify(v) === JSON.stringify(obj[field])) return false;
    obj[field] = clone(v);
    return true;
};

export const settings = {init, loadOr, get, set, reset, resetAll, attachRemote, changed};
