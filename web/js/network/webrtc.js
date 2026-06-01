import {
    pub,
    WEBRTC_CONNECTION_CLOSED,
    WEBRTC_CONNECTION_READY,
    WEBRTC_ICE_CANDIDATE_FOUND,
    WEBRTC_ICE_CANDIDATES_FLUSH,
    WEBRTC_SDP_ANSWER,
} from "event";
import { log } from "log";

let connection;
let dataChannel;
let keyboardChannel;
let mouseChannel;
let mediaStream;
let candidates = [];
let isAnswered = false;
let isFlushing = false;

let connected = false;
let inputReady = false;

let onData;

const ice = (() => {
    const ICE_TIMEOUT = 3000;
    let timeForIceGathering;

    return {
        onIceCandidate: (data) => {
            if (!data.candidate) return;
            log.debug(`[rtc] [ice] local: ${data.candidate?.candidate}`);
            pub(WEBRTC_ICE_CANDIDATE_FOUND, { candidate: data.candidate });
        },
        onIceCandidateError: (event) => {
            let { address, errorCode, errorText, url } = event;

            if (errorCode === 701) {
                errorText = "couldn't reach the server";
            }

            log.debug(
                `[rtc] [ice] candidate error: ${address || ""} ${errorCode}: ${errorText} / ${url}`,
            );
        },
        onIceStateChange: (event) => {
            const t = event.target;
            log.debug(`[rtc] [ice] state: ${t.iceGatheringState}`);

            switch (event.target.iceGatheringState) {
                case "gathering":
                    timeForIceGathering = setTimeout(() => {
                        log.warn(
                            `[rtc] ice gathering was aborted due to timeout ${ICE_TIMEOUT}ms`,
                        );
                        // sendCandidates();
                    }, ICE_TIMEOUT);
                    break;
                case "complete":
                    if (timeForIceGathering) {
                        clearTimeout(timeForIceGathering);
                    }
            }
        },
        onIceConnectionStateChange: () => {
            log.debug(
                `[rtc] [ice] connection state: ${connection.iceConnectionState}`,
            );
            switch (connection.iceConnectionState) {
                case "connected":
                    connected = true;
                    break;
                case "disconnected":
                    log.info(
                        `[rtc] [ice] disconnected... ` +
                            `connection: ${connection.connectionState}, ice: ${connection.iceConnectionState}, ` +
                            `gathering: ${connection.iceGatheringState}, signalling: ${connection.signalingState}`,
                    );
                    connected = false;
                    pub(WEBRTC_CONNECTION_CLOSED);
                    break;
                case "failed":
                    log.error(
                        "[rtc] [ice] failed establish connection, retry...",
                    );
                    connected = false;
                    connection
                        .createOffer({ iceRestart: true })
                        .then((description) =>
                            connection
                                .setLocalDescription(description)
                                .catch(log.error),
                        )
                        .catch(log.error);
                    break;
            }
        },
    };
})();

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start: (iceServers = []) => {
        log.debug("[rtc] got remote ICE servers", iceServers);
        connection = new RTCPeerConnection({ iceServers: iceServers });
        mediaStream = new MediaStream();

        connection.ondatachannel = (e) => {
            log.debug(`[rtc] [data-ch] push: ${e.channel.label}`);
            e.channel.binaryType = "arraybuffer";

            if (e.channel.label === "keyboard") {
                keyboardChannel = e.channel;
                return;
            }

            if (e.channel.label === "mouse") {
                mouseChannel = e.channel;
                return;
            }

            dataChannel = e.channel;
            dataChannel.onopen = () => {
                log.debug("[rtc] [data-ch] input channel has been opened");
                inputReady = true;
                pub(WEBRTC_CONNECTION_READY);
            };
            if (onData) {
                dataChannel.onmessage = onData;
            }
            dataChannel.onclose = () => {
                inputReady = false;
                log.debug("[rtc] [data-ch] input channel has been closed");
            };
        };
        connection.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        connection.onicegatheringstatechange = ice.onIceStateChange;
        connection.onicecandidate = ice.onIceCandidate;
        connection.onicecandidateerror = ice.onIceCandidateError;
        connection.onconnectionstatechange = (_) => {
            console.debug(
                `[rtc] connection state: ${connection.connectionState}`,
            );
        };
        connection.ontrack = (event) => {
            mediaStream.addTrack(event.track);
        };
    },
    setRemoteDescription: async (sdp, media) => {
        log.debug("[rtc] [sdp] remote offer", sdp);

        try {
            const offer = new RTCSessionDescription(sdp);
            await connection.setRemoteDescription(offer);
        } catch (e) {
            log.error(`[rtc] [sdp] remote offer error: ${e}`);
        }

        log.debug(
            `[rtc] [sdp] remote Trickle ICE support: ${connection.canTrickleIceCandidates}`,
        );

        try {
            const answer = await connection.createAnswer();
            // Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
            // force stereo params for Opus tracks (a=fmtp:111 ...)
            answer.sdp = answer.sdp.replace(/(a=fmtp:111 .*)/g, "$1;stereo=1");
            await connection.setLocalDescription(answer);
            log.debug("[rtc] [sdp] local answer", answer);

            isAnswered = true;
            pub(WEBRTC_ICE_CANDIDATES_FLUSH);
            pub(WEBRTC_SDP_ANSWER, { sdp: answer });
        } catch (e) {
            log.error(`[rtc] [sdp] local answer error: ${e}`);
        }

        media.srcObject = mediaStream;
    },
    addCandidate: (data) => {
        if (data === "") {
            pub(WEBRTC_ICE_CANDIDATES_FLUSH);
        } else {
            candidates.push(data);
        }
    },
    flushCandidates: () => {
        if (isFlushing || !isAnswered) return;
        isFlushing = true;
        if (log.level >= log.DEBUG) {
            log.debug(
                `[rtc] [ice] set local candidates (${candidates.length}): ${candidates.map((c) => c.candidate)}`,
            );
        }
        let data = undefined;
        while (typeof (data = candidates.shift()) !== "undefined") {
            connection.addIceCandidate(new RTCIceCandidate(data)).catch((e) => {
                log.error("[rtc] candidate add failed", e.name);
            });
        }
        isFlushing = false;
    },
    keyboard: (data) => keyboardChannel?.send(data),
    mouse: (data) => mouseChannel?.send(data),
    input: (data) => inputReady && dataChannel.send(data),
    isConnected: () => connected,
    isInputReady: () => inputReady,
    stats: async () => {
        if (!connected) return Promise.resolve();
        return await connection.getStats();
    },
    stop: () => {
        if (mediaStream) {
            mediaStream.getTracks().forEach((t) => {
                t.stop();
                mediaStream.removeTrack(t);
            });
            mediaStream = null;
        }
        if (connection) {
            connection.close();
            connection = null;
        }
        if (dataChannel) {
            dataChannel.close();
            dataChannel = null;
        }
        if (keyboardChannel) {
            keyboardChannel?.close();
            keyboardChannel = null;
        }
        if (mouseChannel) {
            mouseChannel?.close();
            mouseChannel = null;
        }
        candidates = [];
        log.info("[rtc] WebRTC has been closed");
    },
    set onData(fn) {
        onData = fn;
    },
};
