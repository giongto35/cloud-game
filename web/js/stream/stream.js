/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
const stream = (() => {
        const screen = document.getElementById('stream');

        let opts = {
                volume: 0.5,
                poster: '/static/img/screen_loading.gif',
                mirrorUpdateRate: 1 / 60,
            },
            state = {
                screen: screen,
                timerId: null,
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

        screen.addEventListener('loadedmetadata', () => {
            if (state.screen !== screen) {
                state.screen.setAttribute('width', screen.videoWidth);
                state.screen.setAttribute('height', screen.videoHeight);
            }
        }, false);
        screen.addEventListener('loadstart', () => {
            screen.volume = opts.volume;
            screen.poster = opts.poster;
        }, false);
        screen.addEventListener('canplay', () => {
            screen.poster = '';
        }, false);

        const useCustomScreen = (use) => {
            if (use) {
                let id = state.screen.getAttribute('id');
                if (id === 'canvas-mirror') return;

                const canvas = gui.create('canvas');
                canvas.setAttribute('id', 'canvas-mirror');
                canvas.setAttribute('hidden', '');
                canvas.setAttribute('width', screen.videoWidth);
                canvas.setAttribute('height', screen.videoHeight);
                canvas.style['image-rendering'] = 'pixelated';
                canvas.classList.add('game-screen');

                screen.parentNode.insertBefore(canvas, screen.nextSibling);
                toggle(false)
                state.screen = canvas
                toggle(true)
                let surface = canvas.getContext('2d');
                state.timerId = setInterval(function () {
                    if (screen.paused || screen.ended || !surface) return;
                    surface.drawImage(screen, 0, 0);
                }, opts.mirrorUpdateRate);
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

        return {
            audio: {mute},
            video: {toggleFullscreen, el: getVideoEl},
            play: stream,
            toggle,
            useCustomScreen,
        }
    }
)(env, gui, log);
