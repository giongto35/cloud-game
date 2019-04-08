// web socket

function startGame() {
    log("Starting game screen");

    // clear
    endInput();
    document.getElementById('div').innerHTML = "";
    $("#loading-screen").show();
    $("#menu-screen").fadeOut();
    // end clear

    conn = new WebSocket(`ws://${location.host}/ws`);

    conn.onopen = () => {
        log("WebSocket is opened. Send ping");
        roomID = roomID.value
        conn.send(JSON.stringify({"id": "ping", "data": GAME_LIST[gameIdx].nes, "room_id": roomID, "player_index": parseInt(playerIndex.value, 10)}));
    }

    conn.onerror = error => {
        log(`Websocket error: ${error}`);
    }

    conn.onclose = () => {
        log("Websocket closed");
    }

    conn.onmessage = e => {
        d = JSON.parse(e.data);
        switch (d["id"]) {
        case "pong":
            log("Recv pong. Start webrtc");
            startWebRTC();
            break;
        case "sdp":
            log("Got remote sdp");
            pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(d["data"]))));
            break;
        case "start":
            log("Got start");
            currentRoomID.value = d["room_id"]
            break;
        }
    }

    // webrtc
    pc = new RTCPeerConnection({iceServers: [{urls: 'stun:stun.l.google.com:19302'}]})
    // input channel
    inputChannel = pc.createDataChannel('foo')
    inputChannel.onclose = () => {
        log('inputChannel has closed');
    }

    inputChannel.onopen = () => {
        log('inputChannel has opened');
    }

    inputChannel.onmessage = e => {
        log(`Message from DataChannel '${inputChannel.label}' payload '${e.data}'`);
    }


    pc.oniceconnectionstatechange = e => {
        log(`iceConnectionState: ${pc.iceConnectionState}`);

        if (pc.iceConnectionState === "connected") {
            conn.send(JSON.stringify({"id": "start", "data": ""}));
            startInput();
            screenState = "game";
        }
        else if (pc.iceConnectionState === "disconnected") {
            endInput();
        }

    }


    // stream channel
    pc.ontrack = function (event) {
        log("New stream, yay!");
        document.getElementById("loading-screen").srcObject = event.streams[0];
        // document.getElementById("loading-screen").width = 270;
        // document.getElementById("loading-screen").height = 240;
    }


    // candidate packet from STUN
    pc.onicecandidate = event => {
        if (event.candidate === null) {

        } else {
            console.log(JSON.stringify(event.candidate));
            // conn.send(JSON.stringify({"id": "candidate", "data": JSON.stringify(event.candidate)}));
        }
    }

    function startWebRTC() {
        // receiver only tracks
        pc.addTransceiver('video', {'direction': 'recvonly'});

        // create SDP
        pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: false}).then(d => {
            pc.setLocalDescription(d, () => {
                // send to ws
                session = btoa(JSON.stringify(pc.localDescription));
                localSessionDescription = session;
                log("Send SDP to remote peer");
                conn.send(JSON.stringify({"id": "sdp", "data": session}));
            });
        }).catch(log);
    }
}
