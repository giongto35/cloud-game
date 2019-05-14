var pc;
var curPacketID = "";
var curSessionID = "";
var gamelist = [];
// web socket

conn = new WebSocket(`ws://${location.host}/ws`);

// Clear old roomID
conn.onopen = () => {
    log("WebSocket is opened. Send ping");
    log("Send ping pong frequently")
    pingpongTimer = setInterval(sendPing, 1000 / PINGPONGPS)

    startWebRTC();
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

    case "gamelist":
        files = JSON.parse(d["data"])
        // parse files list to gamelist
        gamelist = []
        files.forEach(file => {
            var file = file
            var name = file.substr(0, file.indexOf('.'));
            // var image = name + '.png'
            gamelist.push({file: file, name: name})
        })

        // Update Game Options 
        gamelist.forEach(game => {
          ee = document.createElement("option");
          ee.value = game.file;
          ee.innerHTML = game.name;
          gameOp.append(ee);
        });
        log("Received game list", gamelist)
        break;
    case "sdp":
        log("Got remote sdp");
        pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(d["data"]))));
        //conn.send(JSON.stringify({"id": "sdpdon", "packet_id": d["packet_id"]}));
        break;
    case "requestOffer":
        curPacketID = d["packet_id"];
        log("Received request offer ", curPacketID)
        startWebRTC();
        //pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: false}).then(d => {
            //pc.setLocalDescription(d).catch(log);
        //})

    //case "sdpremote":
        //log("Got remote sdp");
        //pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(d["data"]))));
        //conn.send(JSON.stringify({"id": "remotestart", "data": GAME_LIST[gameIdx].nes, "room_id": roomID.value, "player_index": parseInt(playerIndex.value, 10)}));inputTimer
        //break;
    case "heartbeat":
        console.log("Ping: ", Date.now() - d["data"])
        // TODO: Calc time
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

function sendPing() {
    // TODO: format the package with time
    conn.send(JSON.stringify({"id": "heartbeat", "data": Date.now().toString()}));
}

function startWebRTC() {
    // webrtc
    pc = new RTCPeerConnection({iceServers: [{urls: 'stun:stun.l.google.com:19302'}]})

    // input channel, ordered + reliable, id 0
    inputChannel = pc.createDataChannel('a', {
        ordered: true,
        negotiated: true,
        id: 0,
    });
    inputChannel.onopen = () => log('inputChannel has opened');
    inputChannel.onclose = () => log('inputChannel has closed');


    // audio channel, unordered + unreliable, id 1
    var audioCtx = new (window.AudioContext || window.webkitAudioContext)();
    var isInit = false;
    var audioStack = [];
    var nextTime = 0;
    var packetIdx = 0;

    function scheduleBuffers() {
        while (audioStack.length) {
            var buffer = audioStack.shift();
            var source = audioCtx.createBufferSource();
            source.buffer = buffer;
            source.connect(audioCtx.destination);

            // tracking linear time
            if (nextTime == 0)
                nextTime = audioCtx.currentTime + 0.1;  /// add 100ms latency to work well across systems - tune this if you like
            source.start(nextTime);
            nextTime+=source.buffer.duration; // Make the next buffer wait the length of the last buffer before being played

        };
    }

    sampleRate = 16000;
    channels = 1;
    bitDepth = 16;
    decoder = new OpusDecoder(sampleRate, channels);
    function decodeChunk(opusChunk) {
        pcmChunk = decoder.decode_float(opusChunk);
        myBuffer = audioCtx.createBuffer(channels, pcmChunk.length, sampleRate);
        nowBuffering = myBuffer.getChannelData(0, bitDepth, sampleRate);
        nowBuffering.set(pcmChunk);
        return myBuffer;
    }

    audioChannel = pc.createDataChannel('b', {
        ordered: false,
        negotiated: true,
        id: 1,
        maxRetransmits: 0
    })
    audioChannel.onopen = () => log('audioChannel has opened');
    audioChannel.onclose = () => log('audioChannel has closed');
    
    audioChannel.onmessage = (e) => {
        arr = new Uint8Array(e.data);
        idx = arr[arr.length - 1];
        // only accept missing 5 packets
        if (idx < packetIdx && packetIdx - idx < 251) // 256 - 5
            return;
        packetIdx = idx;
        audioStack.push(decodeChunk(e.data));
        if (isInit || (audioStack.length > 10)) { // make sure we put at least 10 chunks in the buffer before starting
            isInit = true;
            scheduleBuffers();
        }
    }


    // 

    pc.oniceconnectionstatechange = e => {
        log(`iceConnectionState: ${pc.iceConnectionState}`);

        if (pc.iceConnectionState === "connected") {
            //conn.send(JSON.stringify({"id": "start", "data": ""}));
        }
        else if (pc.iceConnectionState === "disconnected") {
            stopInputTimer();
        }
    }


    // video channel
    pc.ontrack = function (event) {
        document.getElementById("game-screen").srcObject = event.streams[0];
        $("#game-screen").show();
    }


    // candidate packet from STUN
    pc.onicecandidate = event => {
        if (event.candidate === null) {
            // send to ws
            session = btoa(JSON.stringify(pc.localDescription));
            localSessionDescription = session;
            log("Send SDP to remote peer");
            // TODO: Fix curPacketID
            conn.send(JSON.stringify({"id": "initwebrtc", "data": session, "packet_id": curPacketID}));
        } else {
            //pc.addIceCandidate(event.candidate).catch(e => {
                //log("Failure during addIceCandidate(): " + e.name);});
            conn.send(JSON.stringify({"id": "icecandidate", "data": JSON.stringify(event.candidate)}));
            console.log(JSON.stringify(event.candidate));
        }
    }

    // receiver only tracks
    pc.addTransceiver('video', {'direction': 'recvonly'});

    // create SDP
    pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: false}).then(d => {
        pc.setLocalDescription(d).catch(log);
    })

}

function startGame() {
    log("Starting game screen");
    screenState = "game";

    conn.send(JSON.stringify({"id": "start", "data": gamelist[gameIdx].file, "room_id": roomID.value, "player_index": parseInt(playerIndex.value, 10)}));

    // clear menu screen
    stopInputTimer();
    document.getElementById('div').innerHTML = "";
    if (!DEBUG) {
        $("#menu-screen").fadeOut(400, function() {
            $("#game-screen").show();
        });
    }
    // end clear
    startInputTimer();
}
