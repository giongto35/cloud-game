
// VIRTUAL JOYSTICK
// Ref: https://jsfiddle.net/aa0et7tr/5/

var dpadState = {};
var touchIdx = null;
const maxDiff = 20;


function resetDPad() {
    dpadState = {
        up: false,
        down: false,
        left: false,
        right: false,
    };
    $(".dpad").removeClass("pressed");
}

function checkDPadAxis(bo, axis) {
    if (bo != dpadState[axis]) {
        dpadState[axis] = bo;

        if (dpadState[axis]) {
            $(`#dpad-${axis}`).addClass("pressed");
        } else {
            $(`#dpad-${axis}`).removeClass("pressed");
        }

        // doButton(bo, axis);
    }
}


parent = $("#circle-pad-holder");
stick = document.getElementById("circle-pad");

// TODO: REMOVE MOUSE
parent.on('mousedown', handleMouseDown);
$(document).on('mousemove', handleMouseMove);
$(document).on('mouseup', handleMouseUp);

parent.on('touchstart', handleMouseDown);
parent.on('touchmove', handleMouseMove);
parent.on('touchend', handleMouseUp);

let dragStart = null;
let currentPos = { x: 0, y: 0 };

function handleMouseDown(event) {
    // event.preventDefault();
    stick.style.transition = '0s';
    if (event.changedTouches) {
        touchIdx = event.changedTouches[0].identifier;
        dragStart = {
            x: event.changedTouches[0].clientX,
            y: event.changedTouches[0].clientY,
        };
        resetDPad();
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
        event.clientX = event.changedTouches[touchIdx].clientX;
        event.clientY = event.changedTouches[touchIdx].clientY;
    }
    const xDiff = event.clientX - dragStart.x;
    const yDiff = event.clientY - dragStart.y;
    const angle = Math.atan2(yDiff, xDiff);
    const distance = Math.min(maxDiff, Math.hypot(xDiff, yDiff));
    const xNew = distance * Math.cos(angle);
    const yNew = distance * Math.sin(angle);
    stick.style.transform = `translate(${xNew}px, ${yNew}px)`;
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
    stick.style.transition = '.2s';
    stick.style.transform = `translate3d(0px, 0px, 0px)`;
    dragStart = null;
    currentPos = { x: 0, y: 0 };
    resetDPad();

    // $(".abxy .button").removeClass("pressed");
}







function handleButtonDown(event) {
    $(this).addClass("pressed");
    // doButtonDown($(this).attr("value"));
}

function handleButtonUp(event) {
    $(this).removeClass("pressed");
    // doButtonUp($(this).attr("value"));
}


// TODO: REMOVE MOUSE
// $(".abxy .button").on("mousedown", handleButtonDown);
// $(".abxy .button").on("mouseup", handleButtonUp);
$(".btn").on("mousedown", handleButtonDown);
$(".btn").on("mouseup", handleButtonUp);
$(".btn").on("touchstart", handleButtonDown);
$(".btn").on("touchend", handleButtonUp);

$(".btn-big").on("mousedown", handleButtonDown);
$(".btn-big").on("mouseup", handleButtonUp);
$(".btn-big").on("touchstart", handleButtonDown);
$(".btn-big").on("touchend", handleButtonUp);
