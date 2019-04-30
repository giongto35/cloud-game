// menu screen
function showMenuScreen() {
    log("Clean up connection / frame");

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
function chooseGame(idx, force = false) {
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


// global func

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

function doButton(bo, name) {
    if (bo == true) {
        doButtonDown(name);
    } else if (bo == false) {
        doButtonUp(name);
    }
}

function doButtonDown(name) {
    if (screenState === "game") {
        // game keys
        setState(name, true);
    }
}


function doButtonUp(name) {
    if (screenState === "menu") {
        switch (name) {
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
    }
}

// KEYBOARD

KEYBOARD_MAP = {
    37: "left",
    38: "up",
    39: "right",
    40: "down",

    90: "a", // z
    88: "b", // x
    67: "start", // c
    86: "select", // v

    // non-game
    81: "quit", // q
    83: "save", // s
    76: "load", // l
    70: "full", // f
}

document.body.onkeyup = function (e) {
    if (e.keyCode in KEYBOARD_MAP) {
        doButtonUp(KEYBOARD_MAP[e.keyCode]);
    }
}

document.body.onkeydown = function (e) {
    if (e.keyCode in KEYBOARD_MAP) {
        doButtonDown(KEYBOARD_MAP[e.keyCode]);
    }
};


// JOYSTICK

/*
    cross == a      <--> a
    circle == b     <--> b
    square == x     <--> start
    triangle == y   <--> select
    share           <--> load
    option          <--> save
    L2 == LT        <--> full
    R2 == RT        <--> quit
    dpad            <--> up down left right
    axis 0, 1       <--> second dpad
*/
var padState, gamepadTimer;

function getOS() {
    // linux? ios?
    var OSName = "unknown";
    if (navigator.appVersion.indexOf("Win") != -1) OSName = "win";
    else if (navigator.appVersion.indexOf("Mac") != -1) OSName = "mac";
    else if (navigator.userAgent.indexOf("Android") != -1) OSName = "android";
    return OSName;
}

function getBrowser() {
    var browserName = "unknown";
    if (navigator.userAgent.indexOf("Firefox") != -1) browserName = "firefox";
    if (navigator.userAgent.indexOf("Chrome") != -1) browserName = "chrome";
    if (navigator.userAgent.indexOf("Edge") != -1) browserName = "edge";
    if (navigator.userAgent.indexOf("Version/") != -1) browserName = "safari";
    if (navigator.userAgent.indexOf("UCBrowser") != -1) browserName = "uc";
    return browserName;
}


// only capture the last plugged joystick
window.addEventListener("gamepadconnected", (e) => {
    gamepad = e.gamepad;
    log(`Gamepad connected at index ${gamepad.index}: ${gamepad.id}. ${gamepad.buttons.length} buttons, ${gamepad.axes.length} axes.`);

    padIdx = gamepad.index;

    // Ref: https://github.com/giongto35/cloud-game/issues/14
    // get mapping first (default KeyMap2)
    os = getOS();
    browser = getBrowser();

    console.log(os);
    console.log(browser);

    if (os == "android") {
        // default of android is KeyMap1
        padMap = { 2: "a", 0: "b", 3: "start", 4: "select", 10: "load", 11: "save", 8: "full", 9: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    } else {
        // default of other OS is KeyMap2
        padMap = { 0: "a", 1: "b", 2: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    }

    if (os == "android" && (browser == "firefox" || browser == "uc")) { //KeyMap2
        padMap = { 0: "a", 1: "b", 2: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    }

    if (os == "win" && browser == "firefox") { //KeyMap3
        padMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit" };
    }

    if (os == "mac" && browser == "safari") { //KeyMap4
        padMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 14: "up", 15: "down", 16: "left", 17: "right" };
    }

    if (os == "mac" && browser == "firefox") { //KeyMap5
        padMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 14: "up", 15: "down", 16: "left", 17: "right" };
    }

    // reset state
    padState = {
        left: false,
        right: false,
        up: false,
        down: false,
    };
    Object.keys(padMap).forEach(k => {
        padState[k] = false;
    });


    // looper, too intense?
    if (gamepadTimer) {
        clearInterval(gamepadTimer);
    }

    function checkAxis(bo, axis) {
        if (bo != padState[axis]) {
            padState[axis] = bo;
            doButton(bo, axis);
        }
    }

    gamepadTimer = setInterval(function () {
        gamepad = navigator.getGamepads()[padIdx];
        if (gamepad) {
            // axis pad
            corX = gamepad.axes[0]; // -1 -> 1, left -> right
            corY = gamepad.axes[1]; // -1 -> 1, up -> down
            checkAxis(corX <= -0.5, "left");
            checkAxis(corX >= 0.5, "right");
            checkAxis(corY <= -0.5, "up");
            checkAxis(corY >= 0.5, "down");

            // normal button
            Object.keys(padMap).forEach(k => {
                if (navigator.webkitGetGamepads) {
                    curPressed = (gamepad.buttons[k] == 1);
                } else {
                    curPressed = (gamepad.buttons[k].value > 0 || gamepad.buttons[k].pressed == true);
                }

                if (padState[k] != curPressed) {
                    padState[k] = curPressed;
                    doButton(curPressed, padMap[k]);
                }
            });
        }

    }, 10); // miliseconds per hit

});

window.addEventListener("gamepaddisconnected", (event) => {
    clearInterval(gamepadTimer);
    log(`Gamepad disconnected at index ${e.gamepad.index}`);
});



// VIRTUAL JOYSTICK
// Ref: https://jsfiddle.net/aa0et7tr/5/
createJoystick($('.dpad'));
var dpadState = {};
var touchIdx = null;

function resetDPad() {
    dpadState = {
        up: false,
        down: false,
        left: false,
        right: false,
    };

    $(".dpad .face").removeClass("pressed");
}

function checkDPadAxis(bo, axis) {
    if (bo != dpadState[axis]) {
        dpadState[axis] = bo;

        if (dpadState[axis]) {
            $(`.dpad .${axis}`).addClass("pressed");
        } else {
            $(`.dpad .${axis}`).removeClass("pressed");
        }
    }
}

function createJoystick(parent) {
    const maxDiff = 50;

    stick = document.createElement('div');
    stick.classList.add('joystick');

    // TODO: REMOVE MOUSE
    parent.on('mousedown', handleMouseDown);
    $(document).on('mousemove', handleMouseMove);
    $(document).on('mouseup', handleMouseUp);

    parent.on('touchstart', handleMouseDown);
    parent.on('touchmove', handleMouseMove);
    parent.on('touchend', handleMouseUp);

    let dragStart = null;
    let currentPos = { x: 0, y: 0 };

    function handleMouseDown(event) {
        event.preventDefault();
        stick.style.transition = '0s';
        if (event.changedTouches) {
            touchIdx = event.changedTouches[0].identifier;
            dragStart = {
                x: event.changedTouches[0].clientX,
                y: event.changedTouches[0].clientY,
            };
            resetDPad();
            return;
        }
        dragStart = {
            x: event.clientX,
            y: event.clientY,
        };

    }

    function handleMouseMove(event) {
        event.preventDefault();
        if (dragStart === null) return;
        if (event.changedTouches) {
            event.clientX = event.changedTouches[touchIdx].clientX;
            event.clientY = event.changedTouches[touchIdx].clientY;
        }
        const xDiff = event.clientX - dragStart.x;
        const yDiff = event.clientY - dragStart.y;
        const angle = Math.atan2(yDiff, xDiff);
        const distance = Math.min(maxDiff, Math.hypot(xDiff, yDiff));
        const xNew = distance * Math.cos(angle);
        const yNew = distance * Math.sin(angle);
        stick.style.transform = `translate3d(${xNew}px, ${yNew}px, 0px)`;
        currentPos = { x: xNew, y: yNew };

        const xRatio = xNew / maxDiff;
        const yRatio = yNew / maxDiff;
        checkDPadAxis(xRatio <= -0.5, "left");
        checkDPadAxis(xRatio >= 0.5, "right");
        checkDPadAxis(yRatio <= -0.5, "up");
        checkDPadAxis(yRatio >= 0.5, "down");
    }

    function handleMouseUp(event) {
        event.preventDefault();
        if (dragStart === null) return;
        stick.style.transition = '.2s';
        stick.style.transform = `translate3d(0px, 0px, 0px)`;
        dragStart = null;
        currentPos = { x: 0, y: 0 };
        resetDPad();

        $(".abxy .button").removeClass("pressed");
    }

    parent.append(stick);
    return {
        getPosition: () => currentPos,
    };

}


function handleButtonDown(event) {
    $(this).addClass("pressed");
}

function handleButtonUp(event) {
    $(this).removeClass("pressed");
}


// TODO: REMOVE MOUSE
$(".abxy .button").on("mousedown", handleButtonDown);
$(".abxy .button").on("mouseup", handleButtonUp);

$(".abxy .button").on("touchstart", handleButtonDown);
$(".abxy .button").on("touchend", handleButtonUp);