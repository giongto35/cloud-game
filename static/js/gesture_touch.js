/*
    Touch gesture
*/

// Virtual Gamepad/Joystick
// Ref: https://jsfiddle.net/aa0et7tr/5/

const MAX_DIFF = 20; // pixel

let vpadState = {
    up: false,
    down: false,
    left: false,
    right: false,
};
$(".btn, .btn-big").each(function () {
    vpadState[$(this).attr("value")] = false;
});

let vpadTouchIdx = null;
let vpadTouchDrag = null;
let vpadHolder = $("#circle-pad-holder");
let vpadCircle = $("#circle-pad");


function resetVpadState() {
    // trigger up event?
    checkVpadState("up", false);
    checkVpadState("down", false);
    checkVpadState("left", false);
    checkVpadState("right", false);

    vpadTouchDrag = null;
    vpadTouchIdx = null;
    $(".dpad").removeClass("pressed");
}


function checkVpadState(axis, state) {
    if (state !== vpadState[axis]) {
        vpadState[axis] = state;

        if (state) {
            doButtonDown(axis);
        } else {
            doButtonUp(axis);
        }
    }
}


function handleVpadJoystickDown(event) {
    vpadCircle.css("transition", "0s");
    vpadCircle.css("-moz-transition", "0s");
    vpadCircle.css("-webkit-transition", "0s");
    if (event.changedTouches) {
        resetVpadState();

        vpadTouchIdx = event.changedTouches[0].identifier;
        event.clientX = event.changedTouches[0].clientX;
        event.clientY = event.changedTouches[0].clientY;
    }
    vpadTouchDrag = {
        x: event.clientX,
        y: event.clientY,
    };
}


function handleVpadJoystickUp(event) {
    if (vpadTouchDrag === null) return;

    vpadCircle.css("transition", ".2s");
    vpadCircle.css("-moz-transition", ".2s");
    vpadCircle.css("-webkit-transition", ".2s");
    vpadCircle.css("transform", "translate3d(0px, 0px, 0px)");
    vpadCircle.css("-moz-transform", "translate3d(0px, 0px, 0px)");
    vpadCircle.css("-webkit-transform", "translate3d(0px, 0px, 0px)");

    resetVpadState();
}


function handleVpadJoystickMove(event) {
    if (vpadTouchDrag === null) return;

    if (event.changedTouches) {
        // check if moving source is from other touch?
        for (var i = 0; i < event.changedTouches.length; i++) {
            if (event.changedTouches[i].identifier === vpadTouchIdx) {
                event.clientX = event.changedTouches[i].clientX;
                event.clientY = event.changedTouches[i].clientY;
            }
        }
        if (event.clientX === undefined || event.clientY === undefined)
            return;
    }

    var xDiff = event.clientX - vpadTouchDrag.x;
    var yDiff = event.clientY - vpadTouchDrag.y;
    var angle = Math.atan2(yDiff, xDiff);
    var distance = Math.min(MAX_DIFF, Math.hypot(xDiff, yDiff));
    var xNew = distance * Math.cos(angle);
    var yNew = distance * Math.sin(angle);

    // check if screen is switched or not
    if (isLayoutSwitched) {
        tmp = xNew;
        xNew = yNew;
        yNew = -tmp;
    }

    style = `translate(${xNew}px, ${yNew}px)`;
    vpadCircle.css("transform", style);
    vpadCircle.css("-webkit-transform", style);
    vpadCircle.css("-moz-transform", style);

    var xRatio = xNew / MAX_DIFF;
    var yRatio = yNew / MAX_DIFF;
    checkVpadState("left", xRatio <= -0.5);
    checkVpadState("right", xRatio >= 0.5);
    checkVpadState("up", yRatio <= -0.5);
    checkVpadState("down", yRatio >= 0.5);
}



// touch/mouse events for dpad
vpadHolder.on('mousedown', handleVpadJoystickDown);
vpadHolder.on('touchstart', handleVpadJoystickDown);
vpadHolder.on('touchend', handleVpadJoystickUp);



////////


function handleButtonDown(event) {
    checkVpadState($(this).attr("value"), true);
    // add touchIdx?
}

function handleButtonUp(event) {
    checkVpadState($(this).attr("value"), false);
}


// touch/mouse events for control buttons
$(".btn").on("mousedown", handleButtonDown);
$(".btn").on("touchstart", handleButtonDown);
$(".btn").on("touchend", handleButtonUp);


////////



let menuTouchIdx = null;
let menuTouchDrag = null;
let menuTouchTime = null;

function handleMenuDown(event) {
    // Identify of touch point
    if (event.changedTouches) {
        menuTouchIdx = event.changedTouches[0].identifier;
        event.clientX = event.changedTouches[0].clientX;
        event.clientY = event.changedTouches[0].clientY;
    }

    menuTouchDrag = {
        x: event.clientX,
        y: event.clientY,
    };

    menuTouchTime = Date.now();
}

function handleMenuMove(event) {
    if (menuTouchDrag === null) return;

    if (event.changedTouches) {
        // check if moving source is from other touch?
        for (var i = 0; i < event.changedTouches.length; i++) {
            if (event.changedTouches[i].identifier === menuTouchIdx) {
                event.clientX = event.changedTouches[i].clientX;
                event.clientY = event.changedTouches[i].clientY;
            }
        }
        if (event.clientX == undefined || event.clientY == undefined)
            return;
    }

    var listbox = $("#menu-container");
    listbox.css("transition", ``);
    listbox.css("transform", `translateY(${menuTranslateY - (menuTouchDrag.y - event.clientY)}px)`);
}

function handleMenuUp(event) {
    if (menuTouchDrag === null) return;
    if (event.changedTouches) {
        if (event.changedTouches[0].identifier !== menuTouchIdx)
            return;
        event.clientX = event.changedTouches[0].clientX;
        event.clientY = event.changedTouches[0].clientY;
    }

    var interval = Date.now() - dragTime; // 100ms?
    var newY = 0;
    if (interval < 200) {
        // calc velo
        newY = (menuTouchDrag.y - event.clientY) / interval * 300;
    } else {
        newY = (menuTouchDrag.y - event.clientY);
    }
    // current item?
    menuTranslateY -= newY;
    idx = Math.round(menuTranslateY / -36);
    pickGame(idx);

    menuTouchDrag = null;
}


$("#menu-screen").on("mousedown", handleMenuDown);
$("#menu-screen").on("touchstart", handleMenuDown);
$("#menu-screen").on("touchend", handleMenuUp);


////////



// wtf

function handleWindowMove(event) {
    event.preventDefault();
    handleVpadJoystickMove(event);
    handleMenuMove(event);
    
    // // moving touch
    // if (event.changedTouches) {
    //     for (var i = 0; i < event.changedTouches.length; i++) {
    //         if (event.changedTouches[i].identifier !== menuTouchIdx && event.changedTouches[i].identifier !== vpadTouchIdx) {
    //             // check class
    //             var elem = document.elementFromPoint(event.changedTouches[i].clientX, event.changedTouches[i].clientY);
    //             if (elem.classList.contains("btn")) {
    //                 $(elem).
    //             }
    //         }
    //     }
    // }
}

function handleWindowUp(event) {
    handleVpadJoystickUp(event);
    handleMenuUp(event);
    $(".btn").trigger("touchend");
}



$(window).on("mousemove", handleWindowMove);
window.addEventListener("touchmove", handleWindowMove, {passive: false});

$(window).on("mouseup", handleWindowUp);