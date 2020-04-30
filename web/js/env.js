const env = (() => {
    // UI
    const doc = $(document);
    const gameBoy = $('#gamebody');
    const ghRibbon = $('#ribbon');

    let isLayoutSwitched = false;

    // Window rerender / rotate screen if needed
    const fixScreenLayout = () => {
        let targetWidth = Math.round(doc.width() * 0.9 / 2) * 2,
            targetHeight = Math.round(doc.height() * 0.9 / 2) * 2;

        // mobile == full screen
        if (env.getOs() === 'android') {
            targetWidth = doc.width();
            targetHeight = doc.height();
        }

        // Should have maximum box for desktop?
        // targetWidth = 800; targetHeight = 600; // test on desktop

        rescaleGameBoy(targetWidth, targetHeight);

        let st = isLayoutSwitched ? 'rotate(90deg)' : '';
        ghRibbon.css({
            'bottom': isLayoutSwitched ? 0 : '',
            'top': isLayoutSwitched ? '' : 0,
            'transform': st,
            '-webkit-transform': st,
            '-moz-transform': st
        })
    };

    const rescaleGameBoy = (targetWidth, targetHeight) => {
        const transformations = ['translate(-50%, -50%)'];

        // rotate portrait layout
        isLayoutSwitched = isPortrait();
        if (isLayoutSwitched) {
            transformations.push('rotate(90deg)');
            [targetWidth, targetHeight] = [targetHeight, targetWidth]
        }

        // scale, fit to target size
        const scale = Math.min(targetWidth / gameBoy.width(), targetHeight / gameBoy.height());
        transformations.push(`scale(${scale})`);

        const transform = transformations.join(' ');
        gameBoy.css({
            'transform': transform,
            '-webkit-transform': transform,
            '-moz-transform': transform
        });
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

    // !to use more sophisticated approach / lib
    const isPortrait = () => {
        // ios / mobile app
        switch (window.orientation) {
            case 0:
            case 180:
                return true;
        }

        // desktop
        const orientation = screen.msOrientation || screen.mozOrientation || (screen.orientation || {}).type;
        return orientation === 'portrait-primary';
    };

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
})($, document, log, navigator, screen, window);
