import { log } from "log";

let /** @type {RTCPeerConnection} */ pc;
let /** @type {Map<string, RTCDataChannel>} */ channels = new Map();
let /** @type {MediaStream} */ stream;
let /** @type {RTCLocalIceCandidateInit[]} */ candidateBuf = [];
let handleSdpAnswer;
let _initiator = false;

const ice = (() => {
    let handleIceCandidate;

    const onIceCandidate = (/** @type {RTCPeerConnectionIceEvent} */ ev) => {
        if (!ev.candidate) return;
        log.debug(`[rtc] [ice] local`, ev.candidate);
        if (handleIceCandidate) handleIceCandidate(ev.candidate);
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
    };

    const onIceConnectionStateChange = () => {
        log.debug(`[rtc] [ice] connection state: ${pc.iceConnectionState}`);
        switch (pc.iceConnectionState) {
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
        set handleIceCandidate(cb) {
            handleIceCandidate = cb;
        },
    };
})();

const isConnected = () => pc?.connectionState === "connected";
const hasRemoteDescription = () => pc?.remoteDescription !== null;

const addRemoteCandidate = (data) => {
    if (!data) return;
    const candidate = new RTCIceCandidate(data);
    pc.addIceCandidate(candidate).catch((e) => {
        log.error("[rtc] [ice] remote candidate add failed", e.name);
    });
    log.debug(`[rtc] [ice] added remote`, candidate);
};

const flushRemoteCandidates = () => {
    if (!hasRemoteDescription() || candidateBuf.length === 0) return;

    log.debug(`[rtc] [ice] remote candidate buf (${candidateBuf.length})`);
    let data = undefined;
    while (typeof (data = candidateBuf.shift()) !== "undefined") {
        addRemoteCandidate(data);
    }
};

// hacks
// Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
// force stereo params for Opus tracks (a=fmtp:111 ...)
const enableOpusStereo = (sdp) =>
    sdp.replace(/(a=fmtp:111 .*)/g, "$1;stereo=1");

const pushChannel = (chan) => {
    channels.set(chan.label, chan);
    log.debug(`[rtc] [data-ch] push: ${chan.label}`);
};

const stop = () => {
    if (stream) {
        while (stream.getTracks().length > 0) {
            const t = stream.getTracks()[0];
            t.stop();
            stream.removeTrack(t);
        }
        stream = null;
    }
    if (pc) {
        ice.handleIceCandidate = null;
        handleSdpAnswer = null;
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
};

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start: ({
        iceServers = [],
        media,
        initiator = false,
        onNegotiationNeeded,
        onDataChannel,
        onConnect,
        onDisconnect,
        onIceCandidate,
        onSdpAnswer,
    } = {}) => {
        log.debug("[rtc] got remote ICE servers", iceServers);
        pc = new RTCPeerConnection({ iceCandidatePoolSize: 1, iceServers });
        stream = new MediaStream();

        _initiator = initiator;
        log.debug(`[rtc] you will be: ${_initiator ? "caller" : "callee"}`);

        if (media) {
            media.srcObject = stream;
        } else {
            log.warn(
                "[rtc] no media provided, stream will not be attached to any element",
            );
        }

        if (onIceCandidate) ice.handleIceCandidate = onIceCandidate;
        if (onSdpAnswer) handleSdpAnswer = onSdpAnswer;

        pc.addTransceiver("video", { direction: "recvonly" });
        pc.addTransceiver("audio", { direction: "recvonly" });

        pc.ondatachannel = (ev) => {
            let chan = onDataChannel ? onDataChannel(ev.channel) : ev.channel;
            pushChannel(chan);
        };

        pc.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        pc.onicegatheringstatechange = ice.onIceGatheringStateChange;
        pc.onicecandidate = ice.onIceCandidate;
        pc.onicecandidateerror = ice.onIceCandidateError;
        pc.onconnectionstatechange = () => {
            log.debug(`[rtc] connection state: ${pc.connectionState}`);

            switch (pc.connectionState) {
                case "connected":
                    if (onConnect) onConnect();
                    break;
                case "failed":
                case "closed":
                    if (onDisconnect) onDisconnect();
                    stop();
                    break;
            }
        };
        pc.onnegotiationneeded = () => {
            log.debug("[rtc] negotiation needed");
            if (onNegotiationNeeded) onNegotiationNeeded();
        };
        pc.ontrack = (event) => stream.addTrack(event.track);
    },
    offerSdp: async () => {
        if (!pc || !_initiator) return;

        try {
            const offer = await pc.createOffer();
            offer.sdp = enableOpusStereo(offer.sdp);
            await pc.setLocalDescription(offer);
            log.debug("[rtc] [sdp] local offer", offer);
            return offer;
        } catch (e) {
            log.error(`[rtc] [sdp] local offer error: ${e}`);
        }
    },
    setRemoteDescription: async (
        /** @type {RTCSessionDescriptionInit} */ sdp,
    ) => {
        log.debug("[rtc] [sdp] remote SDP", sdp);

        try {
            await pc.setRemoteDescription(new RTCSessionDescription(sdp));
        } catch (e) {
            log.error(`[rtc] [sdp] remote offer error: ${e}`);
        }

        log.debug(`[rtc] [sdp] Trickle ICE: ${pc.canTrickleIceCandidates}`);

        flushRemoteCandidates();

        if (_initiator) return;

        try {
            const answer = await pc.createAnswer();
            answer.sdp = enableOpusStereo(answer.sdp);
            await pc.setLocalDescription(answer);
            log.debug("[rtc] [sdp] local answer", answer);
            handleSdpAnswer(answer);
        } catch (e) {
            log.error(`[rtc] [sdp] local answer error: ${e}`);
        }
    },
    pushChannel,
    addCandidate: (
        /** @type {RTCLocalIceCandidateInit | string} */ candidate,
    ) => {
        if (hasRemoteDescription()) {
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
    createDataChannel: ({ onChannel }) => {
        try {
            let ch = pc.createDataChannel("data", {
                ordered: false,
                maxRetransmits: 0,
            });
            ch = onChannel ? onChannel(ch) : ch;
            if (!ch) throw new Error("null channel");
            channels.set(ch.label, ch);
        } catch (e) {
            log.error("[rtc] failed to create data channel", e);
        }
    },
    stop,
};
