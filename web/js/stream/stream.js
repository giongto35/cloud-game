/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
const stream = (() => {
    const opts = {
        volume: 0.5,
        poster: '/static/img/screen_loading.gif'
    };

    const screen = document.getElementById('game-screen');

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

    const toggle = (show) => screen.toggleAttribute('hidden', !show)

    const toggleFullscreen = () => {
        let h = parseFloat(getComputedStyle(screen, null).height.replace('px', ''))
        env.display().toggleFullscreen(h !== window.innerHeight, screen);
    }

    const getVideoEl = () => screen


    screen.addEventListener('loadstart', () => {
        screen.volume = opts.volume;
        screen.poster = opts.poster;
    });
    screen.addEventListener('canplay', () => {
        screen.poster = '';
    });

    return Object.freeze({
        audio: {
            mute
        },
        video: {
            toggleFullscreen,
            el: getVideoEl
        },
        play: stream,
        toggle,
    })
})(env, log);
