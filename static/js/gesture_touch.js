
// VIRTUAL JOYSTICK
// Ref: https://jsfiddle.net/aa0et7tr/5/
var dpadState = {};
var touchIdx = null;
const maxDiff = 20; // pixel
var dragStart = null;
var currentPos = { x: 0, y: 0 };
var padHolder = $("#circle-pad-holder");
var padCircle = $("#circle-pad");


function imAlive() {
    $("#room-txt").val(Math.random());
}

function resetTouchState() {
    checkDPadAxis(false, "up");
    checkDPadAxis(false, "down");
    checkDPadAxis(false, "left");
    checkDPadAxis(false, "right");
    
    $(".dpad").removeClass("pressed");
}

function checkDPadAxis(bo, axis) {
    if (bo != dpadState[axis]) {
        dpadState[axis] = bo;

        if (dpadState[axis]) {
            doButtonDown(axis);
        } else {
            doButtonUp(axis);
        }
    }
}


function handleMouseDown(event) {
    // event.preventDefault();
    
    padCircle.css("transition", "0s");
    padCircle.css("-moz-transition", "0s");
    padCircle.css("-webkit-transition", "0s");
    if (event.changedTouches) {
        touchIdx = event.changedTouches[0].identifier;
        
        dragStart = {
            x: event.changedTouches[0].screenX,
            y: event.changedTouches[0].screenY,
        };
        resetTouchState();
        return;
    }
    dragStart = {
        x: event.clientX,
        y: event.clientY,
    };

}

function handleMouseMove(event) {
    // event.preventDefault();
    if (dragStart === null) return;
    if (event.changedTouches) {
        // check if move source is from other touch?
        event.clientX = event.changedTouches[touchIdx].screenX;
        event.clientY = event.changedTouches[touchIdx].screenY;
        imAlive();
    }
    
    const xDiff = event.clientX - dragStart.x;
    const yDiff = event.clientY - dragStart.y;
    const angle = Math.atan2(yDiff, xDiff);
    const distance = Math.min(maxDiff, Math.hypot(xDiff, yDiff));
    const xNew = distance * Math.cos(angle);
    const yNew = distance * Math.sin(angle);
    style = `translate(${xNew}px, ${yNew}px)`;
    // $("#room-txt").val(style);
    padCircle.css("transform", style);
    padCircle.css("-webkit-transform", style);
    padCircle.css("-moz-transform", style);
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
    padCircle.css("transition", ".2s");
    padCircle.css("-moz-transition", ".2s");
    padCircle.css("-webkit-transition", ".2s");
    padCircle.css("transform", "translate3d(0px, 0px, 0px)");
    padCircle.css("-moz-transform", "translate3d(0px, 0px, 0px)");
    padCircle.css("-webkit-transform", "translate3d(0px, 0px, 0px)");

    dragStart = null;
    currentPos = { x: 0, y: 0 };
    resetTouchState();
}


function handleButtonDown(event) {
    doButtonDown($(this).attr("value"));
}

function handleButtonUp(event) {
    doButtonUp($(this).attr("value"));
}


// $(".btn").on("mousedown", handleButtonDown);
// $(".btn").on("mouseup", handleButtonUp);
// $(".btn-big").on("mousedown", handleButtonDown);
// $(".btn-big").on("mouseup", handleButtonUp);
// $(".btn").on("touchstart", handleButtonDown);
// $(".btn").on("touchend", handleButtonUp);
// $(".btn-big").on("touchstart", handleButtonDown);
// $(".btn-big").on("touchend", handleButtonUp);


// padHolder.on('mousedown', handleMouseDown);
// $(document).on('mousemove', handleMouseMove);
// $(document).on('mouseup', handleMouseUp);
padHolder.on('touchstart', handleMouseDown);
$(window).on('touchmove', handleMouseMove);
padHolder.on('touchend', handleMouseUp);