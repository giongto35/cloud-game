var pc;
var curPacketID = "";
var curSessionID = "";
// web socket

conn = new WebSocket(`ws://${location.host}/ws`);

// Clear old roomID
conn.onopen = () => {
    log("WebSocket is opened. Send ping");
    log("Send ping pong frequently")
    // pingpongTimer = setInterval(sendPing, 1000 / PINGPONGPS)

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

    // input channel, ordered + reliable
    inputChannel = pc.createDataChannel('a', {
        ordered: true,
    });
    inputChannel.onclose = () => log('inputChannel has closed');

    window.AudioContext = window.AudioContext || window.webkitAudioContext;
    var context = new AudioContext();
    var delayTime = 0;
    var init = 0;
    var audioStack = [];
    var nextTime = 0;

    var isRun = false;

    function scheduleBuffers() {
        if (isRun) return;
        console.log("daaaa");
        while ( audioStack.length) {
            isRun = true;
            var buffer = audioStack.shift();
            var source    = context.createBufferSource();
            source.buffer = buffer;
            source.connect(context.destination);
            if (nextTime == 0)
                nextTime = context.currentTime + 0.05;  /// add 50ms latency to work well across systems - tune this if you like
            source.start(nextTime);
            nextTime+=source.buffer.duration; // Make the next buffer wait the length of the last buffer before being played
        };
        console.log("diii");
        isRun = false;
    }

    sampleRate = 16000;
    channels = 1;
    bitDepth = 16;
    decoder = new OpusDecoder(sampleRate, channels);
    function damn(opusChunk) {
        pcmChunk = decoder.decode_float(opusChunk);
        myBuffer = context.createBuffer(channels, pcmChunk.length, sampleRate);
        nowBuffering = myBuffer.getChannelData(0, bitDepth, sampleRate);
        nowBuffering.set(pcmChunk);
        return myBuffer;
    }

    pc.ondatachannel = function (ev) {
        log(`New data channel '${ev.channel.label}'`);
        ev.channel.onopen = () => log('channelX has opened');
        ev.channel.onclose = () => log('channelX has closed');

        ev.channel.onmessage = (e) => {
            // source = context.createBufferSource();
            // source.buffer = buf;
            // source.connect(context.destination);
            // source.start(0);

            audioStack.push(damn(e.data));
            if ((init!=0) || (audioStack.length > 10)) { // make sure we put at least 10 chunks in the buffer before starting
                init++;
                scheduleBuffers();
            }
        }
    }

    pc.oniceconnectionstatechange = e => {
        log(`iceConnectionState: ${pc.iceConnectionState}`);

        if (pc.iceConnectionState === "connected") {
            //conn.send(JSON.stringify({"id": "start", "data": ""}));
        }
        else if (pc.iceConnectionState === "disconnected") {
            endInput();
        }
    }

    window.stream = new MediaStream();
    document.getElementById("game-screen2").srcObject = stream;

    // stream channel
    pc.ontrack = function (event) {
        console.log(event);
        stream.addTrack(event.track);
        // var el = document.createElement(event.track.kind);
        // el.srcObject = event.streams[0];
        // el.autoplay = true;
        // el.width = 800;
        // el.height = 600;
        // el.poster = new URL("https://orig00.deviantart.net/cdcd/f/2017/276/a/a/october_2nd___gameboy_poltergeist_by_wanyo-dbpdmnd.gif");
        // document.getElementById('remoteVideos').appendChild(el)

        log("New stream, yay!");
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
            // TODO: Fix curPacketID
            conn.send(JSON.stringify({"id": "initwebrtc", "data": session, "packet_id": curPacketID}));
        } else {
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

    conn.send(JSON.stringify({"id": "start", "data": GAME_LIST[gameIdx].nes, "room_id": roomID.value, "player_index": parseInt(playerIndex.value, 10)}));

    // clear menu screen
    endInput();
    document.getElementById('div').innerHTML = "";
    if (!DEBUG) {
        $("#menu-screen").fadeOut(400, function() {
            $("#game-screen").show();
        });
    }
    // end clear

    // startInput();
}
