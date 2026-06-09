import { log } from "log";

let /** @type {RTCPeerConnection} */ pc;
let /** @type {Map<string, RTCDataChannel>} */ channels = new Map();
let /** @type {MediaStream} */ stream;
let caller = false;
let signal;

const ice = ((signaller) => {
    // Buffer is used to store ICE candidates while
    // the remote description is not available.
    // Then it is flushed as soon as the remote description is set.
    let /** @type {RTCIceCandidateInit[]} */ buf = [];

    const END_OF_CANDIDATES = null;

    const onCandidate = (/** @type {RTCPeerConnectionIceEvent} */ ev) => {
        if (!ev.candidate) return;
        log.debug(`[rtc] [ice] local`, ev.candidate);
        signaller()?.sendIceCandidate(ev.candidate);
    };

    const onCandidateError = (
        /** @type {RTCPeerConnectionIceErrorEvent} */ ev,
    ) => {
        let { address, errorCode, errorText, url } = ev;
        if (errorCode === 701) errorText = "couldn't reach the server";
        log.debug(
            `[rtc] [ice] candidate error: ${address || ""} ${errorCode}: ${errorText} / ${url}`,
        );
    };

    const onGatheringStateChange = (event) => {
        const /** @type {RTCPeerConnection} */ t = event.target;
        log.debug(`[rtc] [ice] state: ${t.iceGatheringState}`);
    };

    const onConnectionStateChange = (pc) => {
        log.debug(`[rtc] [ice] connection state: ${pc.iceConnectionState}`);
        switch (pc.iceConnectionState) {
            case "failed":
                log.error("[rtc] [ice] failed establish connection, retry...");
                pc.restartIce();
                break;
        }
    };

    // add adds or buffers ICE candidates
    // if wait is true
    const add = (pc, candidate, wait = false) => {
        if (wait) {
            buf.push(candidate);
            return;
        }

        const c = candidate
            ? new RTCIceCandidate(candidate)
            : END_OF_CANDIDATES;
        pc.addIceCandidate(c).catch((e) => {
            log.error("[rtc] [ice] add", e.name);
        });
    };

    const flush = (pc) => {
        if (buf.length === 0) return;
        log.debug(`[rtc] [ice] buf (${buf.length}) flush`);
        while (buf.length) {
            add(pc, buf.shift());
        }
    };

    return {
        onCandidate,
        onCandidateError,
        onGatheringStateChange,
        onConnectionStateChange,
        add,
        flush,
        close: () => (buf = []),
    };
})(() => signal);

const isConnected = () => pc?.connectionState === "connected";

// SDP needs some munging
const mung = (sdp) =>
    // Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
    // force stereo params for Opus tracks (a=fmtp:111 ...)
    sdp.replace(/(a=fmtp:111 .*)/g, "$1;stereo=1");

const stub = () => {};

const offer = async () => {
    if (!pc || !caller) return;

    try {
        const offer = await pc.createOffer();
        offer.sdp = mung(offer.sdp);
        await pc.setLocalDescription(offer);
        log.debug("[rtc] [sdp] local:", offer);
        return offer;
    } catch (e) {
        log.error("[rtc] [sdp] local:", e);
    }
};

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start: ({
        iceServers = [],
        media,
        initiator = false,
        onDataChannel = stub,
        onConnect = stub,
        onDisconnect = stub,
        signalling,
    } = {}) => {
        let connectionTime;

        iceServers = iceServers || [];
        log.debug("[rtc] [config] ICE:", iceServers);
        pc = new RTCPeerConnection({ iceServers });

        // push datachannel
        try {
            let ch = pc.createDataChannel("data", {
                negotiated: true,
                id: 0,
                ordered: false,
                maxRetransmits: 0,
            });
            ch = onDataChannel ? onDataChannel(ch) : ch;
            if (!ch) throw new Error("null channel");
            channels.set(ch.label, ch);
        } catch (e) {
            log.error("[rtc] failed to create data channel", e);
            return;
        }

        caller = initiator;
        log.debug(`[rtc] ${caller ? "caller" : "callee"}`);

        if (!signalling) {
            log.error("[rtc] no signalling provided");
            return;
        }
        signal = signalling;

        stream = new MediaStream();
        if (media) {
            media.srcObject = stream;
        } else {
            log.warn("[rtc] no media provided");
        }

        pc.addTransceiver("video", { direction: "recvonly" });
        pc.addTransceiver("audio", { direction: "recvonly" });

        pc.ondatachannel = (/** @type RTCDataChannelEvent */ ev) => {
            const chan = onDataChannel?.(ev.channel) ?? ev.channel;
            channels.set(chan.label, chan);
            log.debug(`[rtc] [chan] add: [${chan.label}]`);
        };
        pc.oniceconnectionstatechange = () => ice.onConnectionStateChange(pc);
        pc.onicegatheringstatechange = ice.onGatheringStateChange;
        pc.onicecandidate = ice.onCandidate;
        pc.onicecandidateerror = ice.onCandidateError;
        pc.onconnectionstatechange = () => {
            log.debug(`[rtc] connection state: ${pc.connectionState}`);
            switch (pc.connectionState) {
                case "connected":
                    onConnect();
                    log.debug(
                        `[rtc] connection time: ${performance.now() - connectionTime}ms`,
                    );
                    break;
                case "failed":
                case "closed":
                    onDisconnect();
                    break;
            }
        };
        pc.onnegotiationneeded = () => {
            log.debug("[rtc] negotiation");
        };
        pc.ontrack = (event) => stream.addTrack(event.track);
        pc.onsignalingstatechange = () => {
            log.debug(`[rtc] [sig] state: ${pc.signalingState}`);

            if (pc.signalingState === "stable") {
                ice.flush(pc);
            }
        };

        connectionTime = performance.now();
        if (initiator) {
            offer().then((offer) => {
                if (!offer) return;
                signalling.init({ initiator, sdpOffer: offer });
            });
        } else {
            signalling.init();
        }
    },
    offer,
    answer: async (/** @type {RTCSessionDescriptionInit} */ sdp) => {
        log.debug("[rtc] [sdp] remote:", sdp);

        if (!pc) return;

        try {
            await pc.setRemoteDescription(new RTCSessionDescription(sdp));
            ice.flush(pc);
        } catch (e) {
            log.error("[rtc] [sdp] remote:", e);
            return;
        }

        if (caller) return;

        try {
            const answer = await pc.createAnswer();
            answer.sdp = mung(answer.sdp);
            await pc.setLocalDescription(answer);
            log.debug("[rtc] [sdp] local:", answer);
            signal?.sendSdp(answer);
        } catch (e) {
            log.error("[rtc] [sdp] local:", e);
        }
    },
    candidate: (/** @type {RTCIceCandidateInit | string} */ candidate) => {
        log.debug(`[rtc] [ice] remote`, candidate);
        if (pc) {
            const buffered =
                !pc.remoteDescription || pc.signalingState !== "stable";
            ice.add(pc, candidate, buffered);
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
            signal = null;
            ice.close();
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
        log.debug("[rtc] closed");
    },
};
