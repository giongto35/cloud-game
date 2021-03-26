const env = (() => {
    // UI
    const page = document.getElementsByTagName('html')[0];
    const gameBoy = document.getElementById('gamebody');
    const ghRibbon = document.getElementById('ribbon');

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

        ghRibbon.style['bottom'] = isLayoutSwitched ? 0 : '';
        ghRibbon.style['top'] = isLayoutSwitched ? '' : 0;
        ghRibbon.style['transform'] = isLayoutSwitched ? 'rotate(90deg)' : '';
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

    return {
        getOs: getOS,
        getBrowser: getBrowser,
        // Check mobile type because different mobile can accept different video encoder.
        isMobileDevice: () => (typeof window.orientation !== 'undefined') || (navigator.userAgent.indexOf('IEMobile') !== -1),
        display: () => ({
            isPortrait: isPortrait,
            toggleFullscreen: toggleFullscreen,
            fixScreenLayout: fixScreenLayout,
            isLayoutSwitched: isLayoutSwitched
        })
    }
})(document, log, navigator, screen, window);
