// menu screen
function showMenuScreen() {
    log("Clean up connection / frame");

    $("#game-screen").hide();
    $("#menu-screen").hide();
    // show
    $("#game-screen").show().delay(DEBUG?0:1000).fadeOut(400, () => {
        log("Loading menu screen");
        $("#menu-screen").fadeIn(400, () => {
            chooseGame(gameIdx, true);
            screenState = "menu";
        });
    });
}


// game menu
function chooseGame(idx, force = false) {
    if (idx < 0 || (idx == gameIdx && !force) || idx >= gameList.length) return false;

    $("#menu-screen #box-art").fadeOut(DEBUG?0:400, function () {
        $(this).attr("src", `/static/img/boxarts/${gameList[idx].name}.png`);
        $(this).fadeIn(400, function () {
            $("#menu-screen #title p").html(gameList[idx].name);
        });
    });

    if (idx == 0) {
        $("#menu-screen .left").hide();
    } else {
        $("#menu-screen .left").show();
    }

    if (idx == gameList.length - 1) {
        $("#menu-screen .right").hide();
    } else {
        $("#menu-screen .right").show();
    }

    gameIdx = idx;
    log(`> [Pick] game ${gameIdx + 1}/${gameList.length} - ${gameList[gameIdx].name}`);
}


// global func


function sendInputData() {
    // prepare key
    if (unchangePacket > 0) {
        bits = "";
        KEY_BIT.slice().reverse().forEach(elem => {
            bits += keyState[elem] ? 1 : 0;
        });
        data = parseInt(bits, 2);
        console.log(`Key state string: ${bits} ==> ${data}`);

        // send
        arrBuf = new Uint8Array(1);
        arrBuf[0] = data;
        inputChannel.send(arrBuf);

        unchangePacket--;
    }
}


function startInputTimer() {
    if (inputTimer == null) {
        inputTimer = setInterval(sendInputData, 1000 / INPUT_FPS)
    }
}

function stopInputTimer() {
    clearInterval(inputTimer);
    inputTimer = null;
}


function setState(name, bo) {
    if (name in keyState) {
        keyState[name] = bo;
        unchangePacket = INPUT_STATE_PACKET;
    }
}

function doButtonDown(name) {
    $(`#btn-${name}`).addClass("pressed");

    if (screenState === "game") {
        // game keys
        setState(name, true);
    }
}


function doButtonUp(name) {
    $(`#btn-${name}`).removeClass("pressed");

    if (screenState === "menu") {
        switch (name) {
            case "left":
                chooseGame(gameIdx - 1);
                break;

            case "right":
                chooseGame(gameIdx + 1);
                break;

            case "join":
                startGame();
                // log("select game");
        }
    } else if (screenState === "game") {
        setState(name, false);

        switch (name) {
            case "save":
                conn.send(JSON.stringify({ "id": "save", "data": "" }));
                break;
            case "load":
                conn.send(JSON.stringify({ "id": "load", "data": "" }));
                break;
            case "full":
                // Fullscreen
                screen = document.getElementById("game-screen");

                console.log(screen.height, window.innerHeight);
                if (screen.height === window.innerHeight) {
                    closeFullscreen();
                } else {
                    openFullscreen(screen);
                }
                break;
        }
    }

    // global reset
    if (name === "quit") {
        stopInputTimer();
        showMenuScreen();
        // TODO: Stop game
        screen = document.getElementById("game-screen");
        room_id = $("#room-txt").val()
        conn.send(JSON.stringify({ "id": "quit", "data": "", "room_id": room_id}));
        $("#room-txt").val("");
    }
}

