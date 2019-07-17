var curPacketID = "";
var curSessionID = "";
// web socket

conn = new WebSocket(`ws://${location.host}/ws`);

// Clear old roomID
conn.onopen = () => {
    log("WebSocket is opened. Send ping");
    log("Send ping pong frequently")
    pingpongTimer = setInterval(sendPing, 1000 / PINGPONGPS)

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
        // parse files list to gamelist
        files = JSON.parse(d["data"]);
        gameList = [];
        files.forEach(file => {
            var file = file
            var name = file.substr(0, file.indexOf('.'));
            gameList.push({file: file, name: name});
        });

        log("Received game list");

        // change screen to menu
        reloadGameMenu();
        showMenuScreen();

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
        // console.log("Ping: ", Date.now() - d["data"])
        // TODO: Calc time
        break;
    case "start":
        roomID = d["room_id"];    
        log(`Got start with room id: ${roomID}`);
        popup("Started! You can share you game!")
        saveRoomID(roomID);

        $("#btn-join").html("share");

        // TODO: remove
        $("#room-txt").val(d["room_id"]);

        break;
    case "save":
        log(`Got save response: ${d["data"]}`);
        popup("Saved");
        break;
    case "load":
        log(`Got load response: ${d["data"]}`);
        popup("Loaded");
        break;
    case "checkLatency":
        var s = d["data"];
        var latencyList = [];
        curPacketID = d["packet_id"];
        latencyPacketID = curPacketID;
        addrs = s.split(",")

        var latenciesMap = {};
        var cntResp = 0;
        beforeTime = Date.now();
        for (const addr of addrs) {
            var sumLatency = 0

            // TODO: Clean code, use async
            var xmlHttp = new XMLHttpRequest();
            xmlHttp.open( "GET", "http://"+addr+":9000/echo?_=" + beforeTime, true ); // false for synchronous request, add date to not calling cache
            xmlHttp.timeout = 1000
            xmlHttp.ontimeout = () => {
                cntResp++;
                afterTime = Date.now();
                //sumLatency += afterTime - beforeTime
                latenciesMap[addr] = afterTime - beforeTime
                if (cntResp == addrs.length) {
                    log(`Send latency list`)
                    console.log(latenciesMap)

                    conn.send(JSON.stringify({"id": "checkLatency", "data": JSON.stringify(latenciesMap), "packet_id": latencyPacketID}));
                    startWebRTC();
        }
            }
            xmlHttp.onload = () => {
                cntResp++;
                afterTime = Date.now();
                //sumLatency += afterTime - beforeTime
                latenciesMap[addr] = afterTime - beforeTime
                if (cntResp == addrs.length) {
                    log(`Send latency list ${latenciesMap}`)
                    console.log(latenciesMap)

                    //conn.send(JSON.stringify({"id": "checkLatency", "data": latenciesMap, "packet_id": latencyPacketID}));
                    conn.send(JSON.stringify({"id": "checkLatency", "data": JSON.stringify(latenciesMap), "packet_id": latencyPacketID}));
                    startWebRTC();
        }
            }
            xmlHttp.send( null );
        }
    }
}

function updateLatencies(beforeTime, addr, latenciesMap, cntResp, curPacketID) {
        afterTime = Date.now();
        //sumLatency += afterTime - beforeTime
        latenciesMap[addr] = afterTime - beforeTime
        if (cntResp == addrs.length) {
            log(`Send latency list ${latenciesMap}`)
            log(curPacketID)

            conn.send(JSON.stringify({"id": "checkLatency", "data": latenciesMap, "packet_id": curPacketID}));
        }
}

function sendPing() {
    // TODO: format the package with time
    conn.send(JSON.stringify({"id": "heartbeat", "data": Date.now().toString()}));
}

function startWebRTC() {
    // webrtc
    var iceservers = [];
    if (STUNTURN == "") {
        iceservers = defaultICE
    } else {
        iceservers = JSON.parse(STUNTURN);
    }
    pc = new RTCPeerConnection({iceServers: iceservers });

    // input channel, ordered + reliable, id 0
    inputChannel = pc.createDataChannel('a', {
        ordered: true,
        negotiated: true,
        id: 0,
    });
    inputChannel.onopen = () => {
        log('inputChannel has opened');
        inputReady = true;
        // TODO: Event based
        if (roomID != "") {
            startGame()
        }
    }
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

    //sampleRate = 16000;
    sampleRate = 48000;
    //sampleRate = 32768;
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
    audioChannel.onopen = () => {
        log('audioChannel has opened');
        audioReady = true;
        // TODO: Event based
        if (roomID != "") {
            startGame()
        }
    }
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


    pc.oniceconnectionstatechange = e => {
        log(`iceConnectionState: ${pc.iceConnectionState}`);

        if (pc.iceConnectionState === "connected") {
            gameReady = true
            iceSuccess = true
            if (roomID != "") {
                startGame()
            }
        }
        else if (pc.iceConnectionState === "failed") {
            gameReady = false
            iceSuccess = false
            log(`failed. Retry...`)
            pc.createOffer({iceRestart: true }).then(d => {
                pc.setLocalDescription(d).catch(log);
            }).catch(log);
        }
        else if (pc.iceConnectionState === "disconnected") {
            stopInputTimer();
        }
    }


    // video channel
    pc.ontrack = function (event) {
        document.getElementById("game-screen").srcObject = event.streams[0];
        var promise = document.getElementById("game-screen").play();
        if (promise !== undefined) {
            promise.then(_ => {
                console.log("Media can autoplay")
            }).catch(error => {
                // Usually error happens when we autoplay unmuted video, browser requires manual play.
                // We already muted video and use separate audio encoding so it's fine now
                console.log("Media Failed to autoplay")
                console.log(error)
                // TODO: Consider workaround
            });
        }
    }


    // candidate packet from STUN
    pc.onicecandidate = event => {
        if (event.candidate === null) {
            // send to ws
            if (!iceSent) {
                session = btoa(JSON.stringify(pc.localDescription));
                log("Send SDP to remote peer");
                // TODO: Fix curPacketID
                //conn.send(JSON.stringify({"id": "initwebrtc", "data": session, "packet_id": curPacketID}));
                conn.send(JSON.stringify({"id": "initwebrtc", "data": session}));
                iceSent = true
            }
        } else {
            console.log(JSON.stringify(event.candidate));
            // TODO: tidy up, setTimeout multiple time now
            // timeout
            setTimeout(() => {
                if (!iceSent) {
                    log("Ice gathering timeout, send anyway")
                    session = btoa(JSON.stringify(pc.localDescription));
                    conn.send(JSON.stringify({"id": "initwebrtc", "data": session}));
                    iceSent = true;
                }
            }, ICE_TIMEOUT)
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
    if (!iceSuccess) {
        popup("Game cannot load. Please refresh");
        return false;
    }
    // TODO: Add while loop
    if (!gameReady || !inputReady || !audioReady) {
        popup("Game is not ready yet. Please wait");
        return false;
    }
    if (screenState != "menu") {
        return false;
    }
    log("Starting game screen");
    screenState = "game";

    // conn.send(JSON.stringify({"id": "start", "data": gameList[gameIdx].file, "room_id": $("#room-txt").val(), "player_index": parseInt(playerIndex.value, 10)}));
    conn.send(JSON.stringify({"id": "start", "data": gameList[gameIdx].file, "room_id": roomID != null ? roomID : '', "player_index": 1}));

    // clear menu screen
    stopGameInputTimer();
    //$("#menu-screen").fadeOut(DEBUG ? 0 : 400, function() {
        //$("#game-screen").show();
    //});
    $("#menu-screen").hide()
    $("#game-screen").show();
    $("#btn-save").show();
    $("#btn-load").show();
    // end clear
    startGameInputTimer();

    return true
}
