/**
 * RTCP connection module.
 * @version 1
 */
const rtcp = (() => {
    let connection;
    let inputChannel;
    let mediaStream;
    let candidates = Array();
    let isAnswered = false;
    let isFlushing = false;

    let connected = false;
    let inputReady = false;

    const start = (iceservers) => {
        log.info(`[rtcp] <- received stunturn from worker ${iceservers}`);

        connection = new RTCPeerConnection({
            iceServers: JSON.parse(iceservers)
        });

        mediaStream = new MediaStream();

        // input channel, ordered + reliable, id 0
        // inputChannel = connection.createDataChannel('a', {ordered: true, negotiated: true, id: 0,});
        // recv dataChannel from worker
        connection.ondatachannel = e => {
            log.debug(`[rtcp] ondatachannel: ${e.channel.label}`)
            inputChannel = e.channel;
            inputChannel.onopen = () => {
                log.debug('[rtcp] the input channel has opened');
                inputReady = true;
                event.pub(CONNECTION_READY)
            };
            inputChannel.onclose = () => log.debug('[rtcp] the input channel has closed');
        }

        // addVoiceStream(connection)

        connection.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        connection.onicegatheringstatechange = ice.onIceStateChange;
        connection.onicecandidate = ice.onIcecandidate;
        connection.ontrack = event => {
            mediaStream.addTrack(event.track);
        }

        socket.send({'id': 'init_webrtc'});
    };

    async function addVoiceStream(connection) {
        let stream = null;

        try {
            stream = await navigator.mediaDevices.getUserMedia({video: false, audio: true});

            stream.getTracks().forEach(function (track) {
                log.info("Added voice track")
                connection.addTrack(track);
            });

        } catch (e) {
            log.info("Error getting audio stream from getUserMedia")
            log.info(e)

        } finally {
            socket.send({'id': 'init_webrtc'});
        }
    }

    const ice = (() => {
        const ICE_TIMEOUT = 2000;
        let timeForIceGathering;

        return {
            onIcecandidate: event => {
                if (!event.candidate) return;
                // send ICE candidate to the worker
                const candidate = JSON.stringify(event.candidate);
                log.info(`[rtcp] user candidate: ${candidate}`);
                socket.send({'id': 'ice_candidate', 'data': btoa(candidate)})
            },
            onIceStateChange: event => {
                switch (event.target.iceGatheringState) {
                    case 'gathering':
                        log.info('[rtcp] ice gathering');
                        timeForIceGathering = setTimeout(() => {
                            log.info(`[rtcp] ice gathering was aborted due to timeout ${ICE_TIMEOUT}ms`);
                            // sendCandidates();
                        }, ICE_TIMEOUT);
                        break;
                    case 'complete':
                        log.info('[rtcp] ice gathering completed');
                        if (timeForIceGathering) {
                            clearTimeout(timeForIceGathering);
                        }
                }
            },
            onIceConnectionStateChange: () => {
                log.info(`[rtcp] <- iceConnectionState: ${connection.iceConnectionState}`);
                switch (connection.iceConnectionState) {
                    case 'connected': {
                        log.info('[rtcp] connected...');
                        connected = true;
                        break;
                    }
                    case 'disconnected': {
                        log.info('[rtcp] disconnected...');
                        connected = false;
                        event.pub(CONNECTION_CLOSED);
                        break;
                    }
                    case 'failed': {
                        log.error('[rtcp] connection failed, retry...');
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
            const offer = new RTCSessionDescription(JSON.parse(atob(data)));
            await connection.setRemoteDescription(offer);

            const answer = await connection.createAnswer();
            // Chrome bug https://bugs.chromium.org/p/chromium/issues/detail?id=818180 workaround
            // force stereo params for Opus tracks (a=fmtp:111 ...)
            answer.sdp = answer.sdp.replace(/(a=fmtp:111 .*)/g, '$1;stereo=1');
            await connection.setLocalDescription(answer);
            log.debug("Local SDP: ", answer)

            isAnswered = true;
            event.pub(MEDIA_STREAM_CANDIDATE_FLUSH);

            socket.send({'id': 'answer', 'data': btoa(JSON.stringify(answer))});

            media.srcObject = mediaStream;
        },
        addCandidate: (data) => {
            if (data === '') {
                event.pub(MEDIA_STREAM_CANDIDATE_FLUSH);
            } else {
                candidates.push(data);
            }
        },
        flushCandidate: () => {
            if (isFlushing || !isAnswered) return;
            isFlushing = true;
            candidates.forEach(data => {
                d = atob(data);
                candidate = new RTCIceCandidate(JSON.parse(d));
                log.debug('[rtcp] add candidate: ' + d);
                connection.addIceCandidate(candidate);
            });
            isFlushing = false;
        },
        input: (data) => inputChannel.send(data),
        isConnected: () => connected,
        isInputReady: () => inputReady,
        getConnection: () => connection,
    }
})(event, socket, env, log);
