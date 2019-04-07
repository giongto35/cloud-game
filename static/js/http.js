// http signal server

function startGame() {
    log("Starting game screen")

    // clear
    endInput();
    document.getElementById('div').innerHTML = "";
    $("#loading-screen").show();
    $("#menu-screen").fadeOut();
    // end clear
    

    // Register with server the session description
    function postSession(session) {
        if (session == "") {
            return;
        }
        var xhttp = new XMLHttpRequest();
        xhttp.onreadystatechange = function () {
            if (this.readyState == 4 && this.status == 200) {
                remoteSessionDescription = this.responseText;
                // document.getElementById('playGame').disabled = false;
                // by original design, we would click to start.
                startSession();
            }
        };
        xhttp.open("POST", "/session", true);
        xhttp.setRequestHeader("Content-type", "application/json");
        xhttp.send(JSON.stringify({ "game": GAME_LIST[gameIdx].nes, "sdp": session }));

    }
    let pc = new RTCPeerConnection({
        iceServers: [
            {
                urls: 'stun:stun.l.google.com:19302'
            }
        ]
    })


    inputChannel = pc.createDataChannel('foo')
    inputChannel.onclose = () => log('inputChannel has closed')
    inputChannel.onopen = () => log('inputChannel has opened')
    inputChannel.onmessage = e => log(`Message from DataChannel '${inputChannel.label}' payload '${e.data}'`)

    pc.ontrack = function (event) {
        log("New stream, yay!");
        document.getElementById("loading-screen").srcObject = event.streams[0];

        // var el = document.createElement(event.track.kind)
        // el.srcObject = event.streams[0]
        // el.autoplay = true
        // el.width = 800;
        // el.height = 600;
        // el.poster = new URL("https://orig00.deviantart.net/cdcd/f/2017/276/a/a/october_2nd___gameboy_poltergeist_by_wanyo-dbpdmnd.gif");

        // document.getElementById('remoteVideos').appendChild(el)
    }

    pc.onicecandidate = event => {
        if (event.candidate === null) {
            var session = btoa(JSON.stringify(pc.localDescription));
            localSessionDescription = session;
            postSession(session)
        }
    }
    pc.createOffer({ offerToReceiveVideo: true, offerToReceiveAudio: true }).then(d => pc.setLocalDescription(d)).catch(log)

    window.startSession = () => {
        let sd = remoteSessionDescription
        if (sd === '') {
            return alert('Session Description must not be empty')
        }
        try {
            pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(sd))));
        } catch (e) {
            alert(e);
        }
    }

    pc.oniceconnectionstatechange = e => {
        log(`iceConnectionState: ${pc.iceConnectionState}`);

        if (pc.iceConnectionState === "connected") {
            startInput();
            screenState = "game";
        }
        else if (pc.iceConnectionState === "disconnected") {
            endInput();
        }

    }
}
