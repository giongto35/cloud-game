/*
  GLOBAL CONSTS
*/
DEBUG = true;

KEY_BIT = ["a", "b", "select", "start", "up", "down", "left", "right"];

INPUT_FPS = 100;
PINGPONGPS = 5;
INPUT_STATE_PACKET = 5;


//------------------------------------------------------------------------
/* 
  GLOBAL VARS
*/

// Game state
var screenState = "loader";
var gameList = [];
var gameIdx = 5;

// Input vars
var keyState = {
    // controllers
    a: false,
    b: false,
    start: false,
    select: false,

    // navigators
    up: false,
    down: false,
    left: false,
    right: false
}

var unchangePacket = INPUT_STATE_PACKET;
var inputTimer = null;

// Connection vars
var pc, inputChannel;
var localSessionDescription = "";
var remoteSessionDescription = "";
var conn;