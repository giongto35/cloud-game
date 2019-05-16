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
    87: "join", // w
    65: "load", // a
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

