// UI
const page = document.getElementsByTagName('html')[0];
const gameBoy = document.getElementById('gamebody');
const sourceLink = document.getElementsByClassName('source')[0];

export const browser = {unknown: 0, firefox: 1, chrome: 2, edge: 3, safari: 4}
export const platform = {unknown: 0, windows: 1, linux: 2, macos: 3, android: 4,}

let isLayoutSwitched = false;

// Window rerender / rotate screen if needed
const fixScreenLayout = () => {
    let pw = getWidth(page),
        ph = getHeight(page),
        targetWidth = Math.round(pw * 0.9 / 2) * 2,
        targetHeight = Math.round(ph * 0.9 / 2) * 2;

    // save page rotation
    isLayoutSwitched = isPortrait();

    rescaleGameBoy(targetWidth, targetHeight);

    sourceLink.style['bottom'] = isLayoutSwitched ? 0 : '';
    if (isLayoutSwitched) {
        sourceLink.style.removeProperty('right');
        sourceLink.style['left'] = 5;
    } else {
        sourceLink.style.removeProperty('left');
        sourceLink.style['right'] = 5;
    }
    sourceLink.style['transform'] = isLayoutSwitched ? 'rotate(-90deg)' : '';
    sourceLink.style['transform-origin'] = isLayoutSwitched ? 'left top' : '';
};

const rescaleGameBoy = (targetWidth, targetHeight) => {
    const transformations = ['translate(-50%, -50%)'];

    if (isLayoutSwitched) {
        transformations.push('rotate(90deg)');
        [targetWidth, targetHeight] = [targetHeight, targetWidth]
    }

    // scale, fit to target size
    const scale = Math.min(targetWidth / getWidth(gameBoy), targetHeight / getHeight(gameBoy));
    transformations.push(`scale(${scale})`);

    gameBoy.style['transform'] = transformations.join(' ');
}

const os = () => {
    const ua = window.navigator.userAgent;
    // noinspection JSUnresolvedReference,JSDeprecatedSymbols
    const plt = window.navigator?.userAgentData?.platform || window.navigator.platform;
    const macs = ["Macintosh", "MacIntel"];
    const wins = ["Win32", "Win64", "Windows"];
    if (wins.indexOf(plt) !== -1) return platform.windows;
    if (macs.indexOf(plt) !== -1) return platform.macos;
    if (/Linux/.test(plt)) return platform.linux;
    if (/Android/.test(ua)) return platform.android;
    return platform.unknown
}

const _browser = () => {
    if (navigator.userAgent.indexOf('Firefox') !== -1) return browser.firefox;
    if (navigator.userAgent.indexOf('Chrome') !== -1) return browser.chrome;
    if (navigator.userAgent.indexOf('Edge') !== -1) return browser.edge;
    if (navigator.userAgent.indexOf('Version/') !== -1) return browser.safari;
    return browser.unknown;
}

const isMobile = () => /Mobi|Android|iPhone/i.test(navigator.userAgent);

const isPortrait = () => getWidth(page) < getHeight(page);

const toggleFullscreen = (enable, element) => {
    const el = enable ? element : document;
    if (enable) {
        el.requestFullscreen?.().then().catch();
        return
    }
    el.exitFullscreen?.().then().catch();
}

function getHeight(el) {
    return parseFloat(getComputedStyle(el, null).height.replace("px", ""));
}

function getWidth(el) {
    return parseFloat(getComputedStyle(el, null).width.replace("px", ""));
}

window.addEventListener('resize', fixScreenLayout);
window.addEventListener('orientationchange', fixScreenLayout);
document.addEventListener('DOMContentLoaded', () => fixScreenLayout(), false);

export const env = {
    getOs: os(),
    getBrowser: _browser(),
    isMobileDevice: isMobile(),
    display: () => ({
        isPortrait,
        toggleFullscreen,
        fixScreenLayout,
        isLayoutSwitched: isLayoutSwitched
    })
}
