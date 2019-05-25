/*
    Joystick gesture
*/

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

let joystickMap;
let joystickState;
let joystickIdx;
let joystickTimer = null;


// check state for each axis -> dpad
function checkJoystickAxisState(name, state) {
    if (joystickState[name] !== state) {
        joystickState[name] = state;
        if (state === true) {
            doButtonDown(name);
        } else {
            doButtonUp(button);
        }
    }
}


// loop timer for checking joystick state
function checkJoystickState() {
    var gamepad = navigator.getGamepads()[joystickIdx];
    if (gamepad) {
        // axis -> dpad
        var corX = gamepad.axes[0]; // -1 -> 1, left -> right
        var corY = gamepad.axes[1]; // -1 -> 1, up -> down
        checkJoystickAxisState("left", corX <= -0.5);
        checkJoystickAxisState("right", corX >= 0.5);
        checkJoystickAxisState("up", corY <= -0.5);
        checkJoystickAxisState("down", corY >= 0.5);

        // normal button map
        Object.keys(joystickMap).forEach(function (btnIdx) {
            var isPressed = false;

            if (navigator.webkitGetGamepads) {
                isPressed = (gamepad.buttons[btnIdx] === 1);
            } else {
                isPressed = (gamepad.buttons[btnIdx].value > 0 || gamepad.buttons[btnIdx].pressed === true);
            }

            if (joystickState[btnIdx] !== isPressed) {
                joystickState[btnIdx] = isPressed;
                if (isPressed === true) {
                    doButtonDown(joystickMap[btnIdx]);
                } else {
                    doButtonUp(joystickMap[btnIdx]);
                }
            }
        });
    }
}


// we only capture the last plugged joystick
$(window).on("gamepadconnected", function (event) {
    var gamepad = event.gamepad;
    log(`Gamepad connected at index ${gamepad.index}: ${gamepad.id}. ${gamepad.buttons.length} buttons, ${gamepad.axes.length} axes.`);

    joystickIdx = gamepad.index;

    // Ref: https://github.com/giongto35/cloud-game/issues/14
    // get mapping first (default KeyMap2)
    var os = getOS();
    var browser = getBrowser();

    if (os === "android") {
        // default of android is KeyMap1
        joystickMap = { 2: "a", 0: "b", 3: "start", 4: "select", 10: "load", 11: "save", 8: "full", 9: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    } else {
        // default of other OS is KeyMap2
        joystickMap = { 0: "a", 1: "b", 2: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    }

    if (os === "android" && (browser === "firefox" || browser === "uc")) { //KeyMap2
        joystickMap = { 0: "a", 1: "b", 2: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 12: "up", 13: "down", 14: "left", 15: "right" };
    }

    if (os === "win" && browser === "firefox") { //KeyMap3
        joystickMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit" };
    }

    if (os === "mac" && browser === "safari") { //KeyMap4
        joystickMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 14: "up", 15: "down", 16: "left", 17: "right" };
    }

    if (os === "mac" && browser === "firefox") { //KeyMap5
        joystickMap = { 1: "a", 2: "b", 0: "start", 3: "select", 8: "load", 9: "save", 6: "full", 7: "quit", 14: "up", 15: "down", 16: "left", 17: "right" };
    }

    // reset state
    joystickState = {
        left: false,
        right: false,
        up: false,
        down: false,
    };
    Object.keys(joystickMap).forEach(function (btnIdx) {
        joystickState[btnIdx] = false;
    });


    // looper, too intense?
    if (joystickTimer !== null) {
        clearInterval(joystickTimer);
    }

    joystickTimer = setInterval(checkJoystickState, 10); // miliseconds per hit

});


// disconnected event is triggered
$(window).on("gamepaddisconnected", (event) => {
    clearInterval(joystickTimer);
    log(`Gamepad disconnected at index ${e.gamepad.index}`);
});


