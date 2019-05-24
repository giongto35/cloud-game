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

    // show menu scene
    $("#game-screen").show().delay(DEBUG ? 0 : 1000).fadeOut(400, () => {
        log("Loading menu screen");
        $("#menu-screen").fadeIn(400, () => {
            pickGame(gameIdx, true);
            screenState = "menu";
        });
    });
}


function pickGame(idx) {
    // check boundaries
    if (idx < 0) idx = 0;
    if (idx >= gameList.length) idx = gameList.length - 1;

    // transition menu box
    menuTranslateY = -idx * 36;
    $("#menu-container").css("transition", `transform 0.5s`);
    $("#menu-container").css("transform", `translateY(${menuTranslateY}px)`);

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
        } else if (name === "join") {
            startGame();
            //log("select game");
        }
    } else if (screenState === "game") {
        setKeyState(name, false);

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
        stopGameInputTimer();
        showMenuScreen();
        
        // TODO: Stop game
        screen = document.getElementById("game-screen");
        room_id = $("#room-txt").val()
        conn.send(JSON.stringify({ "id": "quit", "data": "", "room_id": room_id }));
        $("#room-txt").val("");
    }
}

