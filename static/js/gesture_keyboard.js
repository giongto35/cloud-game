/*
    Keyboard gesture
*/

const KEYBOARD_MAP = {
    37: "left",
    38: "up",
    39: "right",
    40: "down",

    90: "a", // z
    88: "b", // x
    67: "x", // c
    86: "y", // v
    13: "start", // start
    16: "select", // select

    // non-game
    81: "quit", // q
    83: "save", // s
    87: "join", // w
    65: "load", // a
    70: "full", // f
    72: "help", // h
}

$("body").on("keyup", function (event) {
    if (event.keyCode in KEYBOARD_MAP) {
        doButtonUp(KEYBOARD_MAP[event.keyCode]);
    }
});

$("body").on("keydown", function (event) {
    if (event.keyCode in KEYBOARD_MAP) {
        doButtonDown(KEYBOARD_MAP[event.keyCode]);
    }
});

