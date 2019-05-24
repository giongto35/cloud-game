/*
    Global Constants
*/
DEBUG = true;

KEY_BIT = ["a", "b", "select", "start", "up", "down", "left", "right"];

INPUT_FPS = 100;
INPUT_STATE_PACKET = 1;

PINGPONGPS = 5;


/* 
    Global variables
*/


// Game state
let screenState = "loader";
let gameList = [];
let gameIdx = 5;
let gamePickerTimer = null;


// Game controller state
let keyState = {
    // control
    a: false,
    b: false,
    start: false,
    select: false,

    // dpad
    up: false,
    down: false,
    left: false,
    right: false
}

let unchangePacket = INPUT_STATE_PACKET;
let gameInputTimer = null;


// Network state
let pc, inputChannel;
let localSessionDescription = "";
let remoteSessionDescription = "";
let conn;


// Touch menu state
let menuDrag = null;
let menuTranslateY;


// Screen state
let isLayoutSwitched = false;

