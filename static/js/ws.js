// web socket

function startGame() {
    log("Starting game screen");

    // clear
    endInput();
    document.getElementById('div').innerHTML = "";
    if (!DEBUG) {
        $("#menu-screen").fadeOut(400, function() {
            $("#game-screen").show();
        });    
    }
    // end clear

    conn = new WebSocket(`ws://${location.host}/ws`);

    // Clear old roomID
    conn.onopen = () => {
        log("WebSocket is opened. Send ping");
        conn.send(JSON.stringify({"id": "ping", "data": GAME_LIST[gameIdx].nes, "room_id": roomID.value, "player_index": parseInt(playerIndex.value, 10)}));
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
            roomID.value = ""
            currentRoomID.innerText = d["room_id"]
            break;
        case "save":
            log(`Got save response: ${d["data"]}`);
            break;
        case "load":
            log(`Got load response: ${d["data"]}`);
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
        console.log(event.streams);
        var el = document.createElement(event.track.kind);
        el.srcObject = event.streams[0];
        el.autoplay = true;
        el.width = 800;
        el.height = 600;
        el.poster = new URL("https://orig00.deviantart.net/cdcd/f/2017/276/a/a/october_2nd___gameboy_poltergeist_by_wanyo-dbpdmnd.gif");
        document.getElementById('remoteVideos').appendChild(el)

        // log("New stream, yay!");
        // document.getElementById("game-screen").srcObject = event.streams[0];
        // $("#game-screen").show();
    }


    // candidate packet from STUN
    pc.onicecandidate = event => {
        if (event.candidate === null) {
            // send to ws
            session = btoa(JSON.stringify(pc.localDescription));
            localSessionDescription = session;
            log("Send SDP to remote peer");
            conn.send(JSON.stringify({"id": "sdp", "data": session}));
        } else {
            console.log(JSON.stringify(event.candidate));
        }
    }

    function startWebRTC() {
        // receiver only tracks
        pc.addTransceiver('video', {'direction': 'recvonly'});
        pc.addTransceiver('audio', {'direction': 'recvonly'});

        // create SDP
        pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: true}).then(d => {
            pc.setLocalDescription(d).catch(log);
        })
    }
}
