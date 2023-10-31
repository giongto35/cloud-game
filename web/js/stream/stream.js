/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
const stream = (() => {
        const screen = document.getElementById('stream');

        let options = {
                volume: 0.5,
                poster: '/img/screen_loading.gif',
                mirrorMode: null,
                mirrorUpdateRate: 1 / 60,
            },
            state = {
                screen: screen,
                timerId: null,
                w: 0,
                h: 0,
                aspect: 4/3
            };

        const mute = (mute) => screen.muted = mute

        const stream = () => {
            screen.play()
                .then(() => log.info('Media can autoplay'))
                .catch(error => {
                    // Usually error happens when we autoplay unmuted video, browser requires manual play.
                    // We already muted video and use separate audio encoding so it's fine now
                    log.error('Media Failed to autoplay');
                    log.error(error)
                    // TODO: Consider workaround
                });
        }

        const toggle = (show) => {
            state.screen.toggleAttribute('hidden', !show)
        }

        const toggleFullscreen = () => {
            let h = parseFloat(getComputedStyle(state.screen, null)
                .height
                .replace('px', '')
            )
            env.display().toggleFullscreen(h !== window.innerHeight, state.screen);
        }

        const getVideoEl = () => screen

        screen.onerror = (e) => {
            // video playback failed - show a message saying why
            switch (e.target.error.code) {
                case e.target.error.MEDIA_ERR_ABORTED:
                    log.error('You aborted the video playback.');
                    break;
                case e.target.error.MEDIA_ERR_NETWORK:
                    log.error('A network error caused the video download to fail part-way.');
                    break;
                case e.target.error.MEDIA_ERR_DECODE:
                    log.error('The video playback was aborted due to a corruption problem or because the video used features your browser did not support.');
                    break;
                case e.target.error.MEDIA_ERR_SRC_NOT_SUPPORTED:
                    log.error('The video could not be loaded, either because the server or network failed or because the format is not supported.');
                    break;
                default:
                    log.error('An unknown video error occurred.');
                    break;
            }
        };

        screen.addEventListener('loadedmetadata', () => {
            if (state.screen !== screen) {
                state.screen.setAttribute('width', screen.videoWidth);
                state.screen.setAttribute('height', screen.videoHeight);
            }
        }, false);
        screen.addEventListener('loadstart', () => {
            screen.volume = options.volume;
            screen.poster = options.poster;
        }, false);
        screen.addEventListener('canplay', () => {
            screen.poster = '';
            useCustomScreen(options.mirrorMode === 'mirror');
        }, false);

        screen.addEventListener('fullscreenchange', () => {
            const fullscreen = document.fullscreenElement


            screen.style.padding = '0'
            if (fullscreen) {
                const dw = (window.innerWidth - fullscreen.clientHeight * state.aspect) / 2
                screen.style.padding = `0 ${dw}px`
                // chrome bug
                setTimeout(() => {
                    const dw = (window.innerHeight - fullscreen.clientHeight * state.aspect) / 2
                    screen.style.padding = `0 ${dw}px`
                }, 1)
            }

            // !to flipped

        })

        const useCustomScreen = (use) => {
            if (use) {
                if (screen.paused || screen.ended) return;

                let id = state.screen.getAttribute('id');
                if (id === 'canvas-mirror') return;

                const canvas = gui.create('canvas');
                canvas.setAttribute('id', 'canvas-mirror');
                canvas.setAttribute('hidden', '');
                canvas.setAttribute('width', screen.videoWidth);
                canvas.setAttribute('height', screen.videoHeight);
                canvas.style['image-rendering'] = 'pixelated';
                canvas.style.width = '100%'
                canvas.style.height = '100%'
                canvas.classList.add('game-screen');

                // stretch depending on the video orientation
                // portrait -- vertically, landscape -- horizontally
                const isPortrait = screen.videoWidth < screen.videoHeight;
                canvas.style.width = isPortrait ? 'auto' : canvas.style.width;
                // canvas.style.height = isPortrait ? canvas.style.height : 'auto';

                let surface = canvas.getContext('2d');
                screen.parentNode.insertBefore(canvas, screen.nextSibling);
                toggle(false)
                state.screen = canvas
                toggle(true)
                state.timerId = setInterval(function () {
                    if (screen.paused || screen.ended || !surface) return;
                    surface.drawImage(screen, 0, 0);
                }, options.mirrorUpdateRate);
            } else {
                clearInterval(state.timerId);
                let mirror = state.screen;
                state.screen = screen;
                toggle(true);
                if (mirror !== screen) {
                    mirror.parentNode.removeChild(mirror);
                }
            }
        }

        const init = () => {
            options.mirrorMode = settings.loadOr(opts.MIRROR_SCREEN, 'none');
        }

        event.sub(SETTINGS_CHANGED, () => {
            const newValue = settings.get()[opts.MIRROR_SCREEN];
            if (newValue !== options.mirrorMode) {
                useCustomScreen(newValue === 'mirror');
                options.mirrorMode = newValue;
            }
        });


        const fit = 'contain'

        event.sub(APP_VIDEO_CHANGED, (payload) => {
            const {w, h, a} = payload

            state.aspect = a

            const a2 = w / h

            const attr = a.toFixed(6) !== a2.toFixed(6) ? 'fill' : fit
            state.screen.style['object-fit'] = attr

            state.h = payload.h
            state.w = Math.floor(payload.h * payload.a)
            // payload.a > 0 && (state.aspect = payload.a)
            state.screen.setAttribute('width', payload.w)
            state.screen.setAttribute('height', payload.h)
            state.screen.style.aspectRatio = state.aspect
        })

        return {
            audio: {mute},
            video: {toggleFullscreen, el: getVideoEl},
            play: stream,
            toggle,
            useCustomScreen,
            init
        }
    }
)(env, event, gui, log, opts, settings);
