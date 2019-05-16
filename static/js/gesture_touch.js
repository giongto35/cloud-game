// Window rerender / rotate screen if needed
var isSwitch = false;

function fixScreen() {
    target = $(document);
    child = $("#gamebody");

    width = child.width();
    height = child.height();

    // Should have maximum box for desktop
    if (["win", "mac", "linux"].indexOf(getOS()) != -1) {
        targetWidth = Math.min(800, target.width());
        targetHeight = Math.min(600, target.height());
    } else {
        targetWidth = target.width();
        targetHeight = target.height();
    }

    screenWidth = targetWidth;
    screenHeight = targetHeight;

    st = "translate(-50%, -50%) ";
    // rotate ?
    if (isPortrait()) {
        st += `rotate(90deg) `;
        screenWidth = targetHeight;
        screenHeight = targetWidth;
        isSwitch = true;
    } else {
        isSwitch = false;
    }

    // zoom in/out ?
    st += `scale(${Math.min(screenWidth / width, screenHeight / height)}) `;

    child.css("transform", st);
    child.css("-webkit-transform", st);
    child.css("-moz-transform", st);
    child.css("-ms-transform", st);
}
fixScreen();

$(window).on("resize", fixScreen);
$(window).on("orientationchange", fixScreen);




// Virtual Joystick
// Ref: https://jsfiddle.net/aa0et7tr/5/

var dpadState = {};
var touchIdx = null;
const maxDiff = 20; // pixel
var dragStart = null;
var padHolder = $("#circle-pad-holder");
var padCircle = $("#circle-pad");


function resetJoystickState() {
    // trigger up event?
    checkDPadAxis("up", false);
    checkDPadAxis("down", false);
    checkDPadAxis("left", false);
    checkDPadAxis("right", false);
    dragStart = null;
    touchIdx = null;
    $(".dpad").removeClass("pressed");
}

function checkDPadAxis(axis, bo) {
    if (bo != dpadState[axis]) {
        dpadState[axis] = bo;

        if (dpadState[axis]) {
            doButtonDown(axis);
        } else {
            doButtonUp(axis);
        }
    }
}


function handleJoystickDown(event) {
    padCircle.css("transition", "0s");
    padCircle.css("-moz-transition", "0s");
    padCircle.css("-webkit-transition", "0s");
    if (event.changedTouches) {
        resetJoystickState();

        touchIdx = event.changedTouches[0].identifier;
        dragStart = {
            x: event.changedTouches[0].clientX,
            y: event.changedTouches[0].clientY,
        };

        return;
    }
    dragStart = {
        x: event.clientX,
        y: event.clientY,
    };

}

function handleJoystickMove(event) {
    // stop other events
    event.preventDefault();
    if (dragStart === null) return;

    if (event.changedTouches) {
        // check if moving source is from other touch?
        for (var i = 0; i < event.changedTouches.length; i++) {
            if (event.changedTouches[i].identifier === touchIdx) {
                event.clientX = event.changedTouches[i].clientX;
                event.clientY = event.changedTouches[i].clientY;
            }
        }
        if (event.clientX == undefined || event.clientY == undefined)
            return;
    }

    const xDiff = event.clientX - dragStart.x;
    const yDiff = event.clientY - dragStart.y;
    const angle = Math.atan2(yDiff, xDiff);
    const distance = Math.min(maxDiff, Math.hypot(xDiff, yDiff));
    xNew = distance * Math.cos(angle);
    yNew = distance * Math.sin(angle);

    // check if screen is switched or not
    if (isSwitch) {
        tmp = xNew;
        xNew = yNew;
        yNew = -tmp;
    }

    style = `translate(${xNew}px, ${yNew}px)`;
    padCircle.css("transform", style);
    padCircle.css("-webkit-transform", style);
    padCircle.css("-moz-transform", style);

    const xRatio = xNew / maxDiff;
    const yRatio = yNew / maxDiff;
    checkDPadAxis("left", xRatio <= -0.5);
    checkDPadAxis("right", xRatio >= 0.5);
    checkDPadAxis("up", yRatio <= -0.5);
    checkDPadAxis("down", yRatio >= 0.5);
}


function handleJoystickUp(event) {
    if (dragStart === null) return;
    padCircle.css("transition", ".2s");
    padCircle.css("-moz-transition", ".2s");
    padCircle.css("-webkit-transition", ".2s");
    padCircle.css("transform", "translate3d(0px, 0px, 0px)");
    padCircle.css("-moz-transform", "translate3d(0px, 0px, 0px)");
    padCircle.css("-webkit-transform", "translate3d(0px, 0px, 0px)");

    resetJoystickState();
}


function handleButtonDown(event) {
    doButtonDown($(this).attr("value"));
}

function handleButtonUp(event) {
    doButtonUp($(this).attr("value"));
}

// mouse events
$(".btn, .btn-big").on("mousedown", handleButtonDown);
$(".btn, .btn-big").on("mouseup", handleButtonUp);

padHolder.on('mousedown', handleJoystickDown);
$(window).on('mousemove', handleJoystickMove);
$(window).on('mouseup', handleJoystickUp);


// touch events
$(".btn, .btn-big").on("touchstart", handleButtonDown);
$(".btn, .btn-big").on("touchend", handleButtonUp);

padHolder.on('touchstart', handleJoystickDown);
window.addEventListener("touchmove", handleJoystickMove, { passive: false });
padHolder.on('touchend', handleJoystickUp);