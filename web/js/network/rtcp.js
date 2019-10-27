/**
 * RTCP connection module.
 * @version 1
 */
const rtcp = (() => {
    let connection;
    let inputChannel;
    let mediaStream;

    let connected = false;
    let inputReady = false;

    const start = (iceservers) => {
        log.info(`[rtcp] <- received stunturn from worker ${iceservers}`);

        iceservers = JSON.parse(iceservers);

        connection = new RTCPeerConnection({
            iceServers: iceservers
        });

        mediaStream = new MediaStream();

        // input channel, ordered + reliable, id 0
        inputChannel = connection.createDataChannel('a', {ordered: true, negotiated: true, id: 0,});
        inputChannel.onopen = () => {
            log.debug('[rtcp] the input channel has opened');
            inputReady = true;
            event.pub(CONNECTION_READY)
        };
        inputChannel.onclose = () => log.debug('[rtcp] the input channel has closed');

        connection.addTransceiver('video', {'direction': 'recvonly'});
        connection.addTransceiver('audio', {'direction': 'recvonly'});

        connection.oniceconnectionstatechange = ice.onIceConnectionStateChange;
        connection.onicegatheringstatechange = ice.onIceStateChange;
        connection.onicecandidate = ice.onIcecandidate;
        // reserved for the future
        // connection.onicecandidateerror = log.error;

        connection.ontrack = event => mediaStream.addTrack(event.track);

        connection.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: true})
            .then(offer => {
                log.info(offer.sdp);
                connection.setLocalDescription(offer).catch(log.error);
            });
    };

    const ice = (() => {
        let isGatheringDone = false;
        let timeForIceGathering;

        const ICE_TIMEOUT = 2000;

        const sendCandidates = () => {
            if (isGatheringDone) return;
            const session = btoa(JSON.stringify(connection.localDescription));
            const data = JSON.stringify({"sdp": session, "is_mobile": env.isMobileDevice()});
            socket.send({"id": "initwebrtc", "data": data});
            isGatheringDone = true;
        };

        return {
            onIcecandidate: event => {
                if (event.candidate && !isGatheringDone) {
                    log.info(JSON.stringify(event.candidate));
                } else {
                    sendCandidates()
                }
                // TODO: Fix curPacketID
            },
            onIceStateChange: event => {
                switch (event.target.iceGatheringState) {
                    case 'gathering':
                        log.info('[rtcp] ice gathering');
                        timeForIceGathering = setTimeout(() => {
                            log.info(`[rtcp] ice gathering was aborted due to timeout ${ICE_TIMEOUT}ms`);
                            sendCandidates();
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
        setRemoteDescription: (data, media) => {
            connection.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(data))))
            // set media object stream
                .then(() => {
                    media.srcObject = mediaStream;
                })
        },
        input: (data) => inputChannel.send(data),
        isConnected: () => connected,
        isInputReady: () => inputReady
    }
})(event, socket, env, log);
