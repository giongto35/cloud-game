
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


