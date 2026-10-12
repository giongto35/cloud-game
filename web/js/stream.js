import {
    sub,
    APP_VIDEO_CHANGED,
    SETTINGS_CHANGED,
    TRANSFORM_CHANGE,
} from "event";
import { log } from "log";
import { opts, settings } from "settings";

const videoEl = document.getElementById("stream");
const mirrorEl = document.getElementById("mirror-stream");
const playEl = document.getElementById("play-stream");

const options = {
    volume: 0.5,
    poster: "",
    mirrorMode: null,
    mirrorUpdateRate: 1 / 60,
};

const state = {
    screen: videoEl,
    timerId: null,
    w: 0,
    h: 0,
    aspect: 4 / 3,
    fit: "contain",
    flip: false,
    ready: false,
    autoplayWait: false,
};

const mute = (mute) => (videoEl.muted = mute);

const applyTransform = () => {
    const rot = state.rot ? `rotate(${-state.rot}deg)` : "";
    const flip = state.flip ? "scaleY(-1)" : "";
    mirrorEl.style.transform = rot;
    videoEl.style.transform = [rot, flip].filter(Boolean).join(" ");
};

const onPlay = () => {
    state.ready = true;
    videoEl.poster = "";
    applyTransform();
    resize(state.w, state.h, state.aspect, state.fit);
    useCustomScreen(options.mirrorMode === "mirror");
};

const play = () =>
    videoEl
        .play()
        ?.then(onPlay)
        .catch((error) => {
            if (error.name === "NotAllowedError") {
                showPlayButton();
            } else {
                log.error("Playback fail", error);
            }
        });

const toggle = (show) =>
    state.screen.toggleAttribute("hidden", show === undefined ? show : !show);

const resize = (w, h, aspect, fit) => {
    if (!state.ready) return;

    state.screen.setAttribute("width", "" + w);
    state.screen.setAttribute("height", "" + h);
    if (aspect !== undefined && aspect > 0) {
        state.screen.style.setProperty("--ar", "" + aspect);
    }
    if (fit !== undefined) {
        state.screen.style["object-fit"] = fit;
    }

    // properly size the element for rotated view
    if (state.rot && Math.abs(state.rot % 180) > 1) {
        const fullscreen = document.fullscreenElement !== null;
        const ch = fullscreen
            ? window.innerHeight
            : state.screen.parentElement.clientHeight;
        if (ch && state.aspect) {
            const availableH = ch - (fullscreen ? 14 : 0);
            const fw = availableH;
            const fh = Math.round(availableH * state.aspect);
            const shift = (availableH - fh) / 2;
            const rot = `rotate(${-state.rot}deg)`;
            const flp =
                state.flip && state.screen === videoEl ? "scaleY(-1)" : "";
            state.screen.style.width = fw + "px";
            state.screen.style.height = fh + "px";
            state.screen.style.objectFit = "fill";
            state.screen.style.position = "relative";
            state.screen.style.top = "0";
            state.screen.style.transform =
                `translateY(${shift}px) ${rot} ${flp}`.trim();
        }
    } else {
        state.screen.style.width = "";
        state.screen.style.height = "";
        state.screen.style.position = "";
        state.screen.style.top = "";
        // hack: cover 1px edge artifact
        state.screen.style.marginLeft = state.flip ? "1px" : "";
        state.screen.style.marginTop = state.flip ? "-1px" : "";
    }
};

const showPlayButton = () => {
    state.autoplayWait = true;
    toggle();
    playEl.removeAttribute("hidden");
};

playEl.addEventListener("click", () => {
    playEl.setAttribute("hidden", "");
    state.autoplayWait = false;
    play();
    toggle();
});

// Track resize even when the underlying media stream changes its video size
videoEl.addEventListener("resize", () => {
    recalculateSize();
    if (state.screen === videoEl) return;
    resize(videoEl.videoWidth, videoEl.videoHeight);
});

videoEl.addEventListener("loadstart", () => {
    videoEl.volume = options.volume / 100;
});

videoEl.onfocus = () => videoEl.blur();
videoEl.onerror = (e) => log.error("Playback error", e);

const onFullscreen = (fullscreen) => {
    if (document.fullscreenElement === videoEl) {
        videoEl.classList.toggle("no-media-controls", !fullscreen);
        videoEl.blur();
    }
    applyTransform();
    resize(state.w, state.h, state.aspect, state.fit);
    recalculateSize();
};

const vs = { w: 1, h: 1 };

const recalculateSize = () => {
    const fullscreen = document.fullscreenElement !== null;
    const { aspect, screen } = state;

    let width, height;
    // the element keeps the video aspect ratio, so its box is the picture
    ({ width, height } = screen.getBoundingClientRect());
    void fullscreen;
    void aspect;

    vs.w = width;
    vs.h = height;
};

const useCustomScreen = (use) => {
    if (use) {
        if (videoEl.paused || videoEl.ended) return;

        if (state.screen === mirrorEl) return;

        toggle(false);
        state.screen = mirrorEl;
        applyTransform();
        resize(
            videoEl.videoWidth,
            videoEl.videoHeight,
            state.aspect,
            state.fit,
        );

        // stretch depending on the video orientation (skip when rotated -
        // resize handles the sizing)
        if (!(state.rot && Math.abs(state.rot % 180) > 1)) {
            const isPortrait = videoEl.videoWidth < videoEl.videoHeight;
            state.screen.style.width = isPortrait ? "auto" : videoEl.videoWidth;
        }

        let surface = state.screen.getContext("2d");
        if (state.ready) {
            toggle(true);
        }
        state.timerId = setInterval(function () {
            if (videoEl.paused || videoEl.ended || !surface) return;
            if (state.flip) {
                surface.setTransform(1, 0, 0, -1, 0, mirrorEl.height);
                surface.drawImage(videoEl, 0, 0);
                surface.setTransform(1, 0, 0, 1, 0, 0);
            } else {
                surface.drawImage(videoEl, 0, 0);
            }
        }, options.mirrorUpdateRate);
    } else {
        clearInterval(state.timerId);
        toggle(false);
        state.screen = videoEl;
        mirrorEl.style.transform = "";
        if (state.ready) {
            toggle(true);
        }
    }
};

const init = () => {
    options.mirrorMode = settings.loadOr(opts.MIRROR_SCREEN, "none");
    options.volume = settings.loadOr(opts.VOLUME, 50);
    sub(SETTINGS_CHANGED, () => {
        if (settings.changed(opts.MIRROR_SCREEN, options, "mirrorMode")) {
            useCustomScreen(options.mirrorMode === "mirror");
        }
        if (settings.changed(opts.VOLUME, options, "volume")) {
            videoEl.volume = options.volume / 100;
        }
    });
};

const reset = () => {
    state.ready = false;
    state.autoplayWait = false;
    playEl.setAttribute("hidden", "");
    useCustomScreen(false);
    videoEl.srcObject = null;
};

sub(APP_VIDEO_CHANGED, (payload) => {
    const { w, h, a, s, flip, rot } = payload;

    if (flip !== undefined) state.flip = flip;
    if (rot !== undefined) state.rot = rot;
    if (state.ready) applyTransform();

    const scale = !s ? 1 : s;
    const ww = w * scale;
    const hh = h * scale;

    state.aspect = a;

    state.h = hh;
    state.w = Math.floor(hh * a);
    resize(
        ww,
        hh,
        state.aspect,
        a > 1 && a.toFixed(6) !== (ww / hh).toFixed(6) ? "fill" : "contain",
    );
    recalculateSize();
});

sub(TRANSFORM_CHANGE, () => {
    // cache stream element size when the interface is transformed
    recalculateSize();
});

/**
 * Game streaming module.
 * Contains HTML5 AV media elements.
 *
 * @version 1
 */
export const stream = {
    audio: { mute },
    video: {
        el: videoEl,
        get size() {
            return vs;
        },
    },
    play,
    showPlayButton,
    toggle,
    hasDisplay: true,
    init,
    onFullscreen,
    reset,
    recalculateSize,
};
