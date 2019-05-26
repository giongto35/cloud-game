function log(msg) {
    // if (LOG) {
    //     document.getElementById('div').innerHTML += msg + '<br>'
    // }
    console.log(msg);
}

function popup(msg) {
    $("#noti-box").html(msg);
    $("#noti-box").fadeIn().delay(DEBUG ? 0 : 0).fadeOut();
}


function openFullscreen(elem) {
    if (elem.requestFullscreen) {
        elem.requestFullscreen();
    } else if (elem.mozRequestFullScreen) { /* Firefox */
        elem.mozRequestFullScreen();
    } else if (elem.webkitRequestFullscreen) { /* Chrome, Safari and Opera */
        elem.webkitRequestFullscreen();
    } else if (elem.msRequestFullscreen) { /* IE/Edge */
        elem.msRequestFullscreen();
    }
}


function closeFullscreen() {
    if (document.exitFullscreen) {
        document.exitFullscreen();
    } else if (document.mozCancelFullScreen) { /* Firefox */
        document.mozCancelFullScreen();
    } else if (document.webkitExitFullscreen) { /* Chrome, Safari and Opera */
        document.webkitExitFullscreen();
    } else if (document.msExitFullscreen) { /* IE/Edge */
        document.msExitFullscreen();
    }
}


function getOS() {
    // linux? ios?
    var OSName = "unknown";
    if (navigator.appVersion.indexOf("Win") !== -1) OSName = "win";
    else if (navigator.appVersion.indexOf("Mac") !== -1) OSName = "mac";
    else if (navigator.userAgent.indexOf("Linux") !== -1) OSName = "linux";
    else if (navigator.userAgent.indexOf("Android") !== -1) OSName = "android";
    return OSName;
}


function getBrowser() {
    var browserName = "unknown";
    if (navigator.userAgent.indexOf("Firefox") !== -1) browserName = "firefox";
    if (navigator.userAgent.indexOf("Chrome") !== -1) browserName = "chrome";
    if (navigator.userAgent.indexOf("Edge") !== -1) browserName = "edge";
    if (navigator.userAgent.indexOf("Version/") !== -1) browserName = "safari";
    if (navigator.userAgent.indexOf("UCBrowser") !== -1) browserName = "uc";
    return browserName;
}


function isPortrait() {
    // ios / mobile app
    switch (window.orientation) {
        case 0:
        case 180:
            return true;
            break;
    }

    // desktop
    var orient = screen.msOrientation || screen.mozOrientation || (screen.orientation || {}).type;
    if (orient === "portrait-primary") {
        return true;
    }

    return false;
}


function fixElementLayout(elem, targetWidth, targetHeight) {
    var width = elem.width();
    var height = elem.height();

    var st = "translate(-50%, -50%) ";

    // rotate portrait layout
    if (isPortrait()) {
        st += `rotate(90deg) `;
        var tmp = targetHeight;
        targetHeight = targetWidth;
        targetWidth = tmp;
        isLayoutSwitched = true;
    } else {
        isLayoutSwitched = false;
    }

    // scale, fit to target size
    st += `scale(${Math.min(targetWidth / width, targetHeight / height)}) `;

    elem.css("transform", st);
    elem.css("-webkit-transform", st);
    elem.css("-moz-transform", st);
}


function loadRoomID() {
    return localStorage.getItem("roomID");
}


function saveRoomID(roomIdx) {
    localStorage.setItem("roomID", roomIdx);
}


function copyToClipboard(str) {
    const el = document.createElement('textarea');
    el.value = str;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
};
