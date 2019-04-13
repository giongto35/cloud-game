
// menu screen
function showMenuScreen() {
    log("Clean up connection / frame");
    // clean up before / after menu
    //try {
        //inputChannel.gameboyIndeoclose();
    //} catch (err) {
        //log(`> [Warning] peer connection: ${err}`);
    //}

    //try {
        //conn.close();
    //} catch (err) {
        //log(`> [Warning] Websocket connection: ${err}`);
    //}

    $("#game-screen").hide();
    if (!DEBUG) {
        $("#menu-screen").hide();
        // show
        $("#game-screen").show().delay(1000).fadeOut(400, () => {
            log("Loading menu screen");
            $("#menu-screen").fadeIn(400, () => {
                chooseGame(gameIdx, true);
                screenState = "menu";
            });
        });

    } else {
        screenState = "debug";
    }
}


// game menu
function chooseGame(idx, force=false) {
    if (idx < 0 || (idx == gameIdx && !force) || idx >= GAME_LIST.length) return false;

    $("#menu-screen #box-art").fadeOut(400, function () {
        $(this).attr("src", `/static/img/boxarts/${GAME_LIST[idx].art}`);
        $(this).fadeIn(400, function () {
            $("#menu-screen #title p").html(GAME_LIST[idx].name);
        });
    });

    if (idx == 0) {
        $("#menu-screen .left").hide();
    } else {
        $("#menu-screen .left").show();
    }

    if (idx == GAME_LIST.length - 1) {
        $("#menu-screen .right").hide();
    } else {
        $("#menu-screen .right").show();
    }

    gameIdx = idx;
    log(`> [Pick] game ${gameIdx + 1}/${GAME_LIST.length} - ${GAME_LIST[gameIdx].name}`);
}

function setState(e, bo) {
    if (e.keyCode in KEY_MAP) {
        keyState[KEY_MAP[e.keyCode]] = bo;
        stateUnchange = false;
        unchangePacket = INPUT_STATE_PACKET;
    }
}

document.body.onkeyup = function (e) {
    if (screenState === "menu") {
        switch (KEY_MAP[e.keyCode]) {
        case "left":
            chooseGame(gameIdx - 1);
            break;

        case "right":
            chooseGame(gameIdx + 1);
            break;

        case "select":
            startGame();
        }
    } else if (screenState === "game") {
        setState(e, false);

        switch (KEY_MAP[e.keyCode]) {
        case "save":
            conn.send(JSON.stringify({"id": "save", "data": ""}));
            break;
        case "load":
            conn.send(JSON.stringify({"id": "load", "data": ""}));
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
    if (KEY_MAP[e.keyCode] === "quit") {
        endInput();
        showMenuScreen();
    }

}

function openFullscreen(elem) {
  if (elem.requestFullscreen) {
    elem.requestFullscreen();
  } else if (elem.mozRequestFullScreen) { /* Firefox */
    elem.mozRequestFullScreen();
  } else if (elem.webkitRequestFullscreen) { /* Chrome, Safari and Opera */
    elem.webkitRequestFullscreen();
  } else if (elem.msRequestFullscreen) { /* IE/Edge */
    elem.msRequestFullscreen();
  }
}

/* Close fullscreen */
function closeFullscreen() {
  if (document.exitFullscreen) {
    document.exitFullscreen();
  } else if (document.mozCancelFullScreen) { /* Firefox */
    document.mozCancelFullScreen();
  } else if (document.webkitExitFullscreen) { /* Chrome, Safari and Opera */
    document.webkitExitFullscreen();
  } else if (document.msExitFullscreen) { /* IE/Edge */
    document.msExitFullscreen();
  }
}

document.body.onkeydown = function (e) {
  if (screenState === "game") {
    // game keys
    setState(e, true);
  }
};


function sendInput() {
    // prepare key
    if (stateUnchange || unchangePacket > 0) {
        st = "";
        KEY_BIT.slice().reverse().forEach(elem => {
            st += keyState[elem] ? 1 : 0;
        });
        ss = parseInt(st, 2);
        console.log(`Key state string: ${st} ==> ${ss}`);

        // send
        a = new Uint8Array(1);
        a[0] = ss;
        inputChannel.send(a);

        stateUnchange = false;
        unchangePacket--;
    }
}


function startInput() {
    if (inputTimer == null) {
        inputTimer = setInterval(sendInput, 1000 / INPUT_FPS)
    }
}

function endInput() {
    clearInterval(inputTimer);
    inputTimer = null;
}
