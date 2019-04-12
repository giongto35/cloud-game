/* 
  GLOBAL VARS
*/
LOG = true;
if (!("DEBUG" in window)) {
    window.DEBUG = false;
}

// Game state
screenState = "loader";
gameIdx = 0;

// Input vars
keyState = {
    // controllers
    a: false,
    b: false,
    start: false,
    select: false,

    // navigators
    up: false,
    down: false,
    left: false,
    right: false,

    // game meta keys
    save: false,
    load: false,

    // unofficial
    quit: false
}

stateUnchange = true;
unchangePacket = INPUT_STATE_PACKET;
inputTimer = null;

// Connection vars
var pc, inputChannel;
var localSessionDescription = "";
var remoteSessionDescription = "";
var conn;


/* 
  FUNCTIONS
*/

// miscs
function log(msg) {
    if (LOG) {
        document.getElementById('div').innerHTML += msg + '<br>'
        console.log(msg);
    }
}