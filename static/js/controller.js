/*
    Menu Controller
*/

function reloadGameMenu() {
    log("Load game menu");

    // sort gameList first
    gameList.sort(function (a, b) {
        return a.name > b.name ? 1 : -1;
    });

    // generate html
    var listbox = $("#menu-container");
    listbox.html('');
    gameList.forEach(function (game) {
        listbox.append(`<div class="menu-item unselectable" unselectable="on"><div><span>${game.name}</span></div></div>`);
    });
}

function showMenuScreen() {
    // clear scenes
    $("#game-screen").hide();
    $("#menu-screen").hide();
    $("#btn-save").hide();
    $("#btn-load").hide();
    $("#btn-join").html("play");

    // show menu scene
    $("#game-screen").show().delay(DEBUG ? 0 : 0).fadeOut(DEBUG ? 0 : 0, function () {
        log("Loading menu screen");
        $("#menu-screen").fadeIn(DEBUG ? 0 : 0, function () {
            pickGame(gameIdx);
            screenState = "menu";
        });
    });
}


function pickGame(idx) {
    // check boundaries
    // cycle
    if (idx < 0) idx = gameList.length - 1;
    if (idx >= gameList.length) idx = 0;

    // transition menu box

    var listbox = $("#menu-container");
    listbox.css("transition", "top 0.2s");
    listbox.css("-moz-transition", "top 0.2s");
    listbox.css("-webkit-transition", "top 0.2s");

    menuTop = MENU_TOP_POSITION - idx * 36;
    listbox.css("top", `${menuTop}px`);

    // overflow marquee
    $(".menu-item .pick").removeClass("pick");
    $(`.menu-item:eq(${idx}) span`).addClass("pick");

    gameIdx = idx;
    log(`> [Pick] game ${gameIdx + 1}/${gameList.length} - ${gameList[gameIdx].name}`);
}


function startGamePickerTimer(direction) {
    if (gamePickerTimer === null) {
        pickGame(gameIdx + (direction === "up" ? -1 : 1));

        log("Start game picker timer");
        // velocity?
        gamePickerTimer = setInterval(function () {
            pickGame(gameIdx + (direction === "up" ? -1 : 1));
        }, 200);
    }
}

function stopGamePickerTimer() {
    if (gamePickerTimer !== null) {
        log("Stop game picker timer");
        clearInterval(gamePickerTimer);
        gamePickerTimer = null;
    }
}


/*
    Game controller
*/

function sendKeyState() {
    // check if state is changed
    if (unchangePacket > 0) {
        // pack keystate
        var bits = "";
        KEY_BIT.slice().reverse().forEach(elem => {
            bits += keyState[elem] ? 1 : 0;
        });
        var data = parseInt(bits, 2);

        console.log(`Key state string: ${bits} ==> ${data}`);

        // send packed keystate
        var arrBuf = new Uint8Array(1);
        arrBuf[0] = data;
        inputChannel.send(arrBuf);

        unchangePacket--;
    }
}


function startGameInputTimer() {
    if (gameInputTimer === null) {
        log("Start game input timer");
        gameInputTimer = setInterval(sendKeyState, 1000 / INPUT_FPS)
    }
}


function stopGameInputTimer() {
    if (gameInputTimer !== null) {
        log("Stop game input timer");
        clearInterval(gameInputTimer);
        gameInputTimer = null;
    }
}


function setKeyState(name, state) {
    if (name in keyState) {
        keyState[name] = state;
        unchangePacket = INPUT_STATE_PACKET;
    }
}

function doButtonDown(name) {
    $(`#btn-${name}`).addClass("pressed");

    if (screenState === "menu") {
        if (name === "up" || name === "down") {
            startGamePickerTimer(name);
        }
    } else if (screenState === "game") {
        setKeyState(name, true);
    }
}


function doButtonUp(name) {
    $(`#btn-${name}`).removeClass("pressed");

    if (screenState === "menu") {
        if (name === "up" || name === "down") {
            stopGamePickerTimer();
        } else if (name === "join" || name === "a" || name === "b" || name === "start" || name === "select") {
            startGame();
            //log("select game");
        }
    } else if (screenState === "game") {
        setKeyState(name, false);

        switch (name) {
            case "join":
                copyToClipboard(window.location.href.split('?')[0] + `?id=${roomID}`)
                popup("Copy link to clipboard!")
                break;

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
            case "quit":
                stopGameInputTimer();
                showMenuScreen();
                
                // TODO: Stop game
                conn.send(JSON.stringify({ "id": "quit", "data": "", "room_id": roomID }));

                $("#room-txt").val("");
                popup("Quit!");
                break;
        }
    }
}

