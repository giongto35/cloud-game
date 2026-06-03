import {
    pub,
    WEBRTC_CONNECTION_READY,
    WEBRTC_CONNECTION_CLOSED,
    WEBRTC_ICE_CANDIDATE_FOUND,
    WEBRTC_SDP_ANSWER,
} from "event";
import { log } from "log";

let /** @type {RTCPeerConnection} */ pc;
let /** @type {Map<string, RTCDataChannel>} */ channels = new Map();
let /** @type {MediaStream} */ stream;
let /** @type {RTCLocalIceCandidateInit[]} */ candidateBuf = [];
let /** @type {(channel: RTCDataChannel) => RTCDataChannel} */ modDataChannel;

const ice = ((timeout = 3000) => {
    let timeoutId;

    const onIceCandidate = (/** @type {RTCPeerConnectionIceEvent} */ ev) => {
        if (!ev.candidate) return;
        log.debug(`[rtc] [ice] local: ${ev.candidate.candidate}`);
        pub(WEBRTC_ICE_CANDIDATE_FOUND, ev.candidate);
    };

    const onIceCandidateError = (
        /** @type {RTCPeerConnectionIceErrorEvent} */ ev,
    ) => {
        let { address, errorCode, errorText, url } = ev;

        if (errorCode === 701) {
            errorText = "couldn't reach the server";
        }

        log.debug(
            `[rtc] [ice] candidate error: ${address || ""} ${errorCode}: ${errorText} / ${url}`,
        );
    };

    const onIceGatheringStateChange = (event) => {
        const /** @type {RTCPeerConnection} */ t = event.target;
        log.debug(`[rtc] [ice] state: ${t.iceGatheringState}`);

        switch (t.iceGatheringState) {
            case "gathering":
                timeoutId = setTimeout(() => {
                    log.warn(`[rtc] [ice] stopped due to timeout ${timeout}ms`);
                }, timeout);
                break;
            case "complete":
                clearTimeout(timeoutId);
                break;
        }
    };

    const onIceConnectionStateChange = () => {
        log.debug(`[rtc] [ice] connection state: ${pc.iceConnectionState}`);
        switch (pc.iceConnectionState) {
            case "disconnected":
                log.debug(
                    `[rtc] [ice] disconnected... ` +
                        `connection: ${pc.connectionState}, ice: ${pc.iceConnectionState}, ` +
                        `gathering: ${pc.iceGatheringState}, signalling: ${pc.signalingState}`,
                );
                pub(WEBRTC_CONNECTION_CLOSED);
                break;
            case "failed":
                log.error("[rtc] [ice] failed establish connection, retry...");
                pc.restartIce();
                break;
        }
    };

    return {
        onIceCandidate,
        onIceCandidateError,
        onIceGatheringStateChange,
        onIceConnectionStateChange,
    };
})();

const addRemoteCandidate = (data) => {
    if (!data) return;
    pc.addIceCandidate(new RTCIceCandidate(data)).catch((e) => {
        log.error("[rtc] [ice] remote candidate add failed", e.name);
    });
    log.debug(`[rtc] [ice] added remote: ${data.candidate}`);
};

const flushRemoteCandidates = () => {
    // this will work only when the remote description is set
    if (!pc.remoteDescription || candidateBuf.length === 0) return;

    log.debug(`[rtc] [ice] remote candidate buf (${candidateBuf.length})`);
    let data = undefined;
    while (typeof (data = candidateBuf.shift()) !== "undefined") {
        addRemoteCandidate(data);
    }
};

const isConnected = () => pc?.connectionState === "connected";

// hacks
// Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
// force stereo params for Opus tracks (a=fmtp:111 ...)
const enableOpusStereo = (sdp) =>
    sdp.replace(/(a=fmtp:111 .*)/g, "$1;stereo=1");

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start: ({ iceServers = [], media } = {}) => {
        log.debug("[rtc] got remote ICE servers", iceServers);
        pc = new RTCPeerConnection(...(iceServers && [{ iceServers }]));
        stream = new MediaStream();

        if (media) {
            media.srcObject = stream;
        } else {
            log.warn(
                "[rtc] no media provided, stream will not be attached to any element",
            );
        }

        pc.addTransceiver("video", { direction: "recvonly" });
        pc.addTransceiver("audio", { direction: "recvonly" });

        pc.ondatachannel = (ev) => {
            let chan = modDataChannel ? modDataChannel(ev.channel) : ev.channel;
            channels.set(chan.label, chan);
            log.debug(`[rtc] [data-ch] push: ${chan.label}`);
        };
        pc.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        pc.onicegatheringstatechange = ice.onIceGatheringStateChange;
        pc.onicecandidate = ice.onIceCandidate;
        pc.onicecandidateerror = ice.onIceCandidateError;
        pc.onconnectionstatechange = () => {
            log.debug(`[rtc] connection state: ${pc.connectionState}`);
            if (pc.connectionState === "connected")
                pub(WEBRTC_CONNECTION_READY);
        };
        pc.onnegotiationneeded = () => {
            log.debug("[rtc] negotiation needed");
            // todo implement
            // pc.createOffer()
            //     .then((description) =>
            //         pc.setLocalDescription(description).catch(log.error),
            //     )
            //     .catch(log.error);
        };
        pc.ontrack = (event) => {
            stream.addTrack(event.track);
        };
    },
    setRemoteDescription: async (
        /** @type {RTCSessionDescriptionInit} */ sdp,
    ) => {
        log.debug("[rtc] [sdp] remote offer", sdp);

        try {
            const offer = new RTCSessionDescription(sdp);
            await pc.setRemoteDescription(offer);
        } catch (e) {
            log.error(`[rtc] [sdp] remote offer error: ${e}`);
        }

        log.debug(`[rtc] [sdp] Trickle ICE: ${pc.canTrickleIceCandidates}`);

        flushRemoteCandidates();

        try {
            const answer = await pc.createAnswer();
            answer.sdp = enableOpusStereo(answer.sdp);
            await pc.setLocalDescription(answer);
            log.debug("[rtc] [sdp] local answer", answer);
            pub(WEBRTC_SDP_ANSWER, answer);
        } catch (e) {
            log.error(`[rtc] [sdp] local answer error: ${e}`);
        }
    },
    addCandidate: (
        /** @type {RTCLocalIceCandidateInit | string} */ candidate,
    ) => {
        const allowed = pc.remoteDescription !== null;

        if (allowed) {
            addRemoteCandidate(candidate);
        } else {
            candidateBuf.push(candidate);
        }

        if (candidate === "") {
            flushRemoteCandidates();
        }
    },
    send: (chan, data) => {
        const ch = channels.get(chan);
        if (ch?.readyState === "open") ch.send(data);
    },
    isConnected,
    stats: async () => {
        if (!isConnected()) return Promise.resolve();
        return await pc.getStats();
    },
    stop: () => {
        if (stream) {
            while (stream.getTracks().length > 0) {
                const t = stream.getTracks()[0];
                t.stop();
                stream.removeTrack(t);
            }
            stream = null;
        }
        if (pc) {
            pc.oniceconnectionstatechange = null;
            pc.onicegatheringstatechange = null;
            pc.onicecandidate = null;
            pc.onicecandidateerror = null;
            pc.onconnectionstatechange = null;
            pc.ondatachannel = null;
            pc.ontrack = null;
            pc.close();
            pc = null;
        }

        for (const [, channel] of channels) {
            channel.close();
        }
        channels.clear();
        candidateBuf = [];
        log.debug("[rtc] WebRTC has been closed");
    },
    set modDataChannel(fn) {
        modDataChannel = fn;
    },
};
