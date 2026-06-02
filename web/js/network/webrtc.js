import {
    pub,
    WEBRTC_CONNECTION_CLOSED,
    WEBRTC_ICE_CANDIDATE_FOUND,
    WEBRTC_SDP_ANSWER,
} from "event";
import { log } from "log";

let /** @type {RTCPeerConnection} */ pc;
let /** @type {Map<string, DataChannelWrapper>} */ channels = new Map();
let /** @type {MediaStream} */ stream;
let /** @type {RTCLocalIceCandidateInit[]} */ candidateBuf = [];

/**
 * @typedef {Object} DataChannelWrapper
 * @property {(data: string) => void} send - sends data if channel is open.
 * @property {() => void} close - closes the underlying channel.
 */

let connected = false;

let /** @type {(channel: RTCDataChannel) => RTCDataChannel} */ modDataChannel;

const ice = ((timeout = 3000) => {
    let timeoutId;

    const onIceCandidate = (/** @type {RTCPeerConnectionIceEvent} */ ev) => {
        if (!ev.candidate) return;
        log.debug(`[rtc] [ice] local: ${ev.candidate.candidate}`);
        pub(WEBRTC_ICE_CANDIDATE_FOUND, { candidate: ev.candidate });
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
            case "connected":
                connected = true;
                break;
            case "disconnected":
                log.info(
                    `[rtc] [ice] disconnected... ` +
                        `connection: ${pc.connectionState}, ice: ${pc.iceConnectionState}, ` +
                        `gathering: ${pc.iceGatheringState}, signalling: ${pc.signalingState}`,
                );
                connected = false;
                pub(WEBRTC_CONNECTION_CLOSED);
                break;
            case "failed":
                log.error("[rtc] [ice] failed establish connection, retry...");
                connected = false;
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

// readyChan - wraps an RTCDataChannel to ensure it is ready before sending
const readyChan = (/** @type {RTCDataChannel} */ dc) => ({
    send: (/** @type {string} */ data) =>
        dc.readyState === "open" && dc.send(data),
    close: dc.close,
});

const flushRemoteCandidates = () => {
    // this will work only when the remote description is set
    if (!pc.remoteDescription) return;

    if (log.level >= log.DEBUG) {
        log.debug(
            `[rtc] [ice] remote candidates (${candidateBuf.length}): ${candidateBuf.map((c) => c.candidate)}`,
        );
    }
    let data = undefined;
    while (typeof (data = candidateBuf.shift()) !== "undefined") {
        pc.addIceCandidate(new RTCIceCandidate(data)).catch((e) => {
            log.error("[rtc] remote candidate add failed", e.name);
        });
    }
};

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start: (iceServers = [], media) => {
        log.debug("[rtc] got remote ICE servers", iceServers);
        pc = new RTCPeerConnection({ iceServers });
        stream = new MediaStream();
        media.srcObject = stream;

        pc.addTransceiver("video", { direction: "recvonly" });
        pc.addTransceiver("audio", { direction: "recvonly" });

        pc.ondatachannel = (ev) => {
            let chan = modDataChannel ? modDataChannel(ev.channel) : ev.channel;
            channels.set(chan.label, readyChan(chan));
            log.debug(`[rtc] [data-ch] push: ${chan.label}`);
        };
        pc.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        pc.onicegatheringstatechange = ice.onIceGatheringStateChange;
        pc.onicecandidate = ice.onIceCandidate;
        pc.onicecandidateerror = ice.onIceCandidateError;
        pc.onconnectionstatechange = () => {
            log.debug(`[rtc] connection state: ${pc.connectionState}`);
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
            // Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
            // force stereo params for Opus tracks (a=fmtp:111 ...)
            answer.sdp = answer.sdp.replace(/(a=fmtp:111 .*)/g, "$1;stereo=1");
            await pc.setLocalDescription(answer);
            log.debug("[rtc] [sdp] local answer", answer);
            pub(WEBRTC_SDP_ANSWER, { sdp: answer });
        } catch (e) {
            log.error(`[rtc] [sdp] local answer error: ${e}`);
        }
    },
    addCandidate: (/** @type {RTCLocalIceCandidateInit} */ candidate) => {
        if (candidate === "") {
            flushRemoteCandidates();
            return;
        }
        candidateBuf.push(candidate);
    },
    send: (chan, data) => channels.get(chan)?.send(data),
    isConnected: () => connected,
    stats: async () => {
        if (!connected) return Promise.resolve();
        return await pc.getStats();
    },
    stop: () => {
        if (stream) {
            stream.getTracks().forEach((t) => {
                t.stop();
                stream.removeTrack(t);
            });
            stream = null;
        }
        if (pc) {
            pc.close();
            pc = null;
        }

        for (const [, channel] of channels) {
            channel.close();
        }
        channels.clear();
        candidateBuf = [];
        log.info("[rtc] WebRTC has been closed");
    },
    set modDataChannel(fn) {
        modDataChannel = fn;
    },
};
