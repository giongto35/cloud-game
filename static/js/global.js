/*
    Global Constants
*/
const DEBUG = false;

const KEY_BIT = ["a", "b", "select", "start", "up", "down", "left", "right"];
const INPUT_FPS = 100;
const INPUT_STATE_PACKET = 1;
const PINGPONGPS = 5;

const MENU_TOP_POSITION = 102;
const ICE_TIMEOUT = 2000;

/* 
    Global variables
*/


// Game state
let screenState = "loader";
let gameList = [];
let gameIdx = 5; // contra
let gamePickerTimer = null;
let roomID = null;


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
let menuTop = MENU_TOP_POSITION;


// Screen state
let isLayoutSwitched = false;

var gameReady = false;
var iceSuccess = true;
var iceSent = false; // TODO: set to false in some init event
var defaultICE = [{urls: "stun:stun.l.google.com:19302"}]
