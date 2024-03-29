// UI
const page = document.getElementsByTagName('html')[0];
const gameBoy = document.getElementById('gamebody');
const sourceLink = document.getElementsByClassName('source')[0];

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

const getOS = () => {
    // linux? ios?
    let OSName = 'unknown';
    if (navigator.appVersion.indexOf('Win') !== -1) OSName = 'win';
    else if (navigator.appVersion.indexOf('Mac') !== -1) OSName = 'mac';
    else if (navigator.userAgent.indexOf('Linux') !== -1) OSName = 'linux';
    else if (navigator.userAgent.indexOf('Android') !== -1) OSName = 'android';
    return OSName;
};

const getBrowser = () => {
    let browserName = 'unknown';
    if (navigator.userAgent.indexOf('Firefox') !== -1) browserName = 'firefox';
    if (navigator.userAgent.indexOf('Chrome') !== -1) browserName = 'chrome';
    if (navigator.userAgent.indexOf('Edge') !== -1) browserName = 'edge';
    if (navigator.userAgent.indexOf('Version/') !== -1) browserName = 'safari';
    if (navigator.userAgent.indexOf('UCBrowser') !== -1) browserName = 'uc';
    return browserName;
};

const isPortrait = () => getWidth(page) < getHeight(page);

const toggleFullscreen = (enable, element) => {
    const el = enable ? element : document;

    if (enable) {
        if (el.requestFullscreen) {
            el.requestFullscreen();
        } else if (el.mozRequestFullScreen) { /* Firefox */
            el.mozRequestFullScreen();
        } else if (el.webkitRequestFullscreen) { /* Chrome, Safari and Opera */
            el.webkitRequestFullscreen();
        } else if (el.msRequestFullscreen) { /* IE/Edge */
            el.msRequestFullscreen();
        }
    } else {
        if (el.exitFullscreen) {
            el.exitFullscreen();
        } else if (el.mozCancelFullScreen) { /* Firefox */
            el.mozCancelFullScreen();
        } else if (el.webkitExitFullscreen) { /* Chrome, Safari and Opera */
            el.webkitExitFullscreen();
        } else if (el.msExitFullscreen) { /* IE/Edge */
            el.msExitFullscreen();
        }
    }
};

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
    getOs: getOS,
    getBrowser: getBrowser,
    isMobileDevice: () => /Mobi|Android|iPhone/i.test(navigator.userAgent),
    display: () => ({
        isPortrait: isPortrait,
        toggleFullscreen: toggleFullscreen,
        fixScreenLayout: fixScreenLayout,
        isLayoutSwitched: isLayoutSwitched
    })
}
