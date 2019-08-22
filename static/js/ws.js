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

    case "init":
        // TODO: Read from struct
        // init package has 2 part [stunturn, gamelist]
        // The first element is stunturn address
        // The rest are list of game
        data = JSON.parse(d["data"]);
        stunturn = data[0]
        startWebRTC(stunturn);
        data.shift()
        gameList = [];

        data.forEach(file => {
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
        break;

    case "requestOffer":
        curPacketID = d["packet_id"];
        log("Received request offer ", curPacketID)
        startWebRTC();

    case "heartbeat":
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
                    //startWebRTC();
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
                    //startWebRTC();
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

function startWebRTC(iceservers) {
    log(`received stunturn from worker ${iceservers}`)
    // webrtc
    iceservers = JSON.parse(iceservers);
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


    var stream = new MediaStream();
    document.getElementById("game-screen").srcObject = stream;

    // video channel
    pc.ontrack = function (event) {
        stream.addTrack(event.track);
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
    pc.addTransceiver('audio', {'direction': 'recvonly'});

    // create SDP
    pc.createOffer({offerToReceiveVideo: true, offerToReceiveAudio: true}).then(d => {
        log(d.sdp)
        pc.setLocalDescription(d).catch(log);
    })

}

function startGame() {
    if (!iceSuccess) {
        popup("Game cannot load. Please refresh");
        return false;
    }
    // TODO: Add while loop
    if (!gameReady || !inputReady) {
        popup("Game is not ready yet. Please wait");
        return false;
    }

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

    if (screenState != "menu") {
        return false;
    }
    log("Starting game screen");
    screenState = "game";

    // conn.send(JSON.stringify({"id": "start", "data": gameList[gameIdx].file, "room_id": $("#room-txt").val(), "player_index": parseInt(playerIndex.value, 10)}));
    conn.send(JSON.stringify({"id": "start", "data": gameList[gameIdx].file, "room_id": roomID != null ? roomID : '', "player_index": 1}));

    // clear menu screen
    stopGameInputTimer();
    $("#menu-screen").hide()
    $("#game-screen").show();
    $("#btn-save").show();
    $("#btn-load").show();
    // end clear
    startGameInputTimer();

    return true
}
