import {
    pub,
    WEBRTC_CONNECTION_CLOSED,
    WEBRTC_CONNECTION_READY,
    WEBRTC_ICE_CANDIDATE_FOUND,
    WEBRTC_ICE_CANDIDATES_FLUSH,
    WEBRTC_SDP_ANSWER
} from 'event';
import {log} from 'log';

let connection;
let dataChannel
let keyboardChannel
let mouseChannel
let mediaStream;
let candidates = [];
let isAnswered = false;
let isFlushing = false;

let connected = false;
let inputReady = false;

let onData;

const start = (iceservers) => {
    log.info('[rtc] <- ICE servers', iceservers);
    const servers = iceservers || [];
    connection = new RTCPeerConnection({iceServers: servers});
    mediaStream = new MediaStream();

    connection.ondatachannel = e => {
        log.debug('[rtc] ondatachannel', e.channel.label)
        e.channel.binaryType = "arraybuffer";

        if (e.channel.label === 'keyboard') {
            keyboardChannel = e.channel
            return
        }

        if (e.channel.label === 'mouse') {
            mouseChannel = e.channel
            return
        }

        dataChannel = e.channel;
        dataChannel.onopen = () => {
            log.info('[rtc] the input channel has been opened');
            inputReady = true;
            pub(WEBRTC_CONNECTION_READY)
        };
        if (onData) {
            dataChannel.onmessage = onData;
        }
        dataChannel.onclose = () => {
            inputReady = false
            log.info('[rtc] the input channel has been closed')
        }
    }
    connection.oniceconnectionstatechange = ice.onIceConnectionStateChange;
    connection.onicegatheringstatechange = ice.onIceStateChange;
    connection.onicecandidate = ice.onIcecandidate;
    connection.ontrack = event => {
        mediaStream.addTrack(event.track);
    }
};

const stop = () => {
    if (mediaStream) {
        mediaStream.getTracks().forEach(t => {
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
        dataChannel.close()
        dataChannel = null
    }
    if (keyboardChannel) {
        keyboardChannel?.close()
        keyboardChannel = null
    }
    if (mouseChannel) {
        mouseChannel?.close()
        mouseChannel = null
    }
    candidates = [];
    log.info('[rtc] WebRTC has been closed');
}

const ice = (() => {
    const ICE_TIMEOUT = 2000;
    let timeForIceGathering;

    return {
        onIcecandidate: data => {
            if (!data.candidate) return;
            log.info('[rtc] user candidate', data.candidate);
            pub(WEBRTC_ICE_CANDIDATE_FOUND, {candidate: data.candidate})
        },
        onIceStateChange: event => {
            switch (event.target.iceGatheringState) {
                case 'gathering':
                    log.info('[rtc] ice gathering');
                    timeForIceGathering = setTimeout(() => {
                        log.warn(`[rtc] ice gathering was aborted due to timeout ${ICE_TIMEOUT}ms`);
                        // sendCandidates();
                    }, ICE_TIMEOUT);
                    break;
                case 'complete':
                    log.info('[rtc] ice gathering has been completed');
                    if (timeForIceGathering) {
                        clearTimeout(timeForIceGathering);
                    }
            }
        },
        onIceConnectionStateChange: () => {
            log.info('[rtc] <- iceConnectionState', connection.iceConnectionState);
            switch (connection.iceConnectionState) {
                case 'connected':
                    log.info('[rtc] connected...');
                    connected = true;
                    break;
                case 'disconnected':
                    log.info(`[rtc] disconnected... ` +
                        `connection: ${connection.connectionState}, ice: ${connection.iceConnectionState}, ` +
                        `gathering: ${connection.iceGatheringState}, signalling: ${connection.signalingState}`)
                    connected = false;
                    pub(WEBRTC_CONNECTION_CLOSED);
                    break;
                case 'failed':
                    log.error('[rtc] failed establish connection, retry...');
                    connected = false;
                    connection.createOffer({iceRestart: true})
                        .then(description => connection.setLocalDescription(description).catch(log.error))
                        .catch(log.error);
                    break;
            }
        }
    }
})();

/**
 * WebRTC connection module.
 */
export const webrtc = {
    start,
    setRemoteDescription: async (data, media) => {
        log.debug('[rtc] remote SDP', data)
        const offer = new RTCSessionDescription(JSON.parse(atob(data)));
        await connection.setRemoteDescription(offer);

        const answer = await connection.createAnswer();
        // Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
        // force stereo params for Opus tracks (a=fmtp:111 ...)
        answer.sdp = answer.sdp.replace(/(a=fmtp:111 .*)/g, '$1;stereo=1');
        await connection.setLocalDescription(answer);
        log.debug("[rtc] local SDP", answer)

        isAnswered = true;
        pub(WEBRTC_ICE_CANDIDATES_FLUSH);
        pub(WEBRTC_SDP_ANSWER, {sdp: answer});
        media.srcObject = mediaStream;
    },
    addCandidate: (data) => {
        if (data === '') {
            pub(WEBRTC_ICE_CANDIDATES_FLUSH);
        } else {
            candidates.push(data);
        }
    },
    flushCandidates: () => {
        if (isFlushing || !isAnswered) return;
        isFlushing = true;
        log.debug('[rtc] flushing candidates', candidates);
        candidates.forEach(data => {
            const candidate = new RTCIceCandidate(JSON.parse(atob(data)))
            connection.addIceCandidate(candidate).catch(e => {
                log.error('[rtc] candidate add failed', e.name);
            });
        });
        isFlushing = false;
    },
    keyboard: (data) => keyboardChannel?.send(data),
    mouse: (data) => mouseChannel?.send(data),
    input: (data) => inputReady && dataChannel.send(data),
    isConnected: () => connected,
    isInputReady: () => inputReady,
    stats: async () => {
        if (!connected) return Promise.resolve();
        return await connection.getStats()
    },
    stop,
    set onData(fn) {
        onData = fn
    }
}
