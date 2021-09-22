/**
 * WebRTC connection module.
 * @version 1
 *
 * Events:
 *   @link WEBRTC_CONNECTION_CLOSED
 *   @link WEBRTC_CONNECTION_READY
 *   @link WEBRTC_ICE_CANDIDATE_FOUND
 *   @link WEBRTC_ICE_CANDIDATES_FLUSH
 *   @link WEBRTC_SDP_ANSWER
 *
 */
const webrtc = (() => {
    let connection;
    let inputChannel;
    let mediaStream;
    let candidates = Array();
    let isAnswered = false;
    let isFlushing = false;

    let connected = false;
    let inputReady = false;

    let onMessage;

    const start = (iceservers) => {
        log.info('[rtc] <- ICE servers', iceservers);

        connection = new RTCPeerConnection({iceServers: iceservers});
        mediaStream = new MediaStream();

        connection.ondatachannel = e => {
            log.debug('[rtc] ondatachannel', e.channel.label)
            inputChannel = e.channel;
            inputChannel.onopen = () => {
                log.info('[rtc] the input channel has been opened');
                inputReady = true;
                event.pub(WEBRTC_CONNECTION_READY)
            };
            if (onMessage) {
                inputChannel.onmessage = onMessage;
        }
            inputChannel.onclose = () => log.info('[rtp] the input channel has been closed');
        }
        connection.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        connection.onicegatheringstatechange = ice.onIceStateChange;
        connection.onicecandidate = ice.onIcecandidate;
        connection.ontrack = event => {
            mediaStream.addTrack(event.track);
        }
    };

    const ice = (() => {
        const ICE_TIMEOUT = 2000;
        let timeForIceGathering;

        return {
            onIcecandidate: data => {
                if (!data.candidate) return;
                log.info('[rtc] user candidate', data.candidate);
                event.pub(WEBRTC_ICE_CANDIDATE_FOUND, {candidate: data.candidate})
            },
            onIceStateChange: event => {
                switch (event.target.iceGatheringState) {
                    case 'gathering':
                        log.info('[rtc] ice gathering');
                        timeForIceGathering = setTimeout(() => {
                            log.warning(`[rtc] ice gathering was aborted due to timeout ${ICE_TIMEOUT}ms`);
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
                    case 'connected': {
                        log.info('[rtc] connected...');
                        connected = true;
                        break;
                    }
                    case 'disconnected': {
                        log.info('[rtc] disconnected...');
                        connected = false;
                        event.pub(WEBRTC_CONNECTION_CLOSED);
                        break;
                    }
                    case 'failed': {
                        log.error('[rtc] failed establish connection, retry...');
                        connected = false;
                        connection.createOffer({iceRestart: true})
                            .then(description => connection.setLocalDescription(description).catch(log.error))
                            .catch(log.error);
                        break;
                    }
                }
            }
        }
    })();

    return {
        start: start,
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
            event.pub(WEBRTC_ICE_CANDIDATES_FLUSH);
            event.pub(WEBRTC_SDP_ANSWER, {sdp: answer});
            media.srcObject = mediaStream;
        },
        setMessageHandler: (handler) => onMessage = handler,
        addCandidate: (data) => {
            if (data === '') {
                event.pub(WEBRTC_ICE_CANDIDATES_FLUSH);
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
                    console.error('[rtc] candidate add failed', e.name);
                });
            });
            isFlushing = false;
        },
        message: (mess = '') => inputChannel.send(mess),
        input: (data) => inputChannel.send(data),
        isConnected: () => connected,
        isInputReady: () => inputReady,
        getConnection: () => connection,
    }
})(event, log);
