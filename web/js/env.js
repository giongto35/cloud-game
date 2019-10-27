const env = (() => {
    const win = $(window);
    const doc = $(document);

    // Screen state
    let isLayoutSwitched = false;

    // Window rerender / rotate screen if needed
    const fixScreenLayout = () => {
        let targetWidth = doc.width() * 0.9;
        let targetHeight = doc.height() * 0.9;

        // mobile == full screen
        if (env.getOs() === 'android') {
            targetWidth = doc.width();
            targetHeight = doc.height();
        }

        // Should have maximum box for desktop?
        // targetWidth = 800; targetHeight = 600; // test on desktop

        fixElementLayout($('#gamebody'), targetWidth, targetHeight);

        const elem = $('#ribbon');
        let st = '';
        if (isLayoutSwitched) {
            st = 'rotate(90deg)';
            elem.css('bottom', 0);
            elem.css('top', '');
        } else {
            elem.css('bottom', '');
            elem.css('top', 0);
        }
        elem.css('transform', st);
        elem.css('-webkit-transform', st);
        elem.css('-moz-transform', st);
    };

    const fixElementLayout = (elem, targetWidth, targetHeight) => {
        let st = 'translate(-50%, -50%) ';

        // rotate portrait layout
        if (isPortrait()) {
            st += `rotate(90deg) `;
            let tmp = targetHeight;
            targetHeight = targetWidth;
            targetWidth = tmp;
            isLayoutSwitched = true;
        } else {
            isLayoutSwitched = false;
        }

        // scale, fit to target size
        st += `scale(${Math.min(targetWidth / elem.width(), targetHeight / elem.height())}) `;

        elem.css('transform', st);
        elem.css('-webkit-transform', st);
        elem.css('-moz-transform', st);
    };

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

    // bindings
    win.on('resize', fixScreenLayout);
    win.on('orientationchange', fixScreenLayout);

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
})($, window, document, log);
