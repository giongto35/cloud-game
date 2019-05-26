
// Window rerender / rotate screen if needed
function fixScreenLayout() {
    
    var targetWidth = $(document).width() * 0.9;
    var targetHeight = $(document).height() * 0.9;

    // mobile == full screen
    if (getOS() === "android") {
        var targetWidth = $(document).width();
        var targetHeight = $(document).height();
    }

    // Should have maximum box for desktop?
    // targetWidth = 800; targetHeight = 600; // test on desktop

    fixElementLayout($("#gamebody"), targetWidth, targetHeight);
}

$(window).on("resize", fixScreenLayout);
$(window).on("orientationchange", fixScreenLayout);


function parseURLForRoom() {
    var queryDict = {}
    location.search.substr(1).split("&").forEach(function(item) {
        queryDict[item.split("=")[0]] = item.split("=")[1]
    });
    if (typeof queryDict["id"] === "string") {
        return queryDict["id"];
    }
    return null;
}

$(document).ready(function () {
    fixScreenLayout();


    // localStorage first
    //roomID = loadRoomID();
    roomID = "";

    // Shared URL second
    var rid = parseURLForRoom();
    if (rid !== null) {
        roomID = rid;
    }
    // if from URL -> start game immediately!

    $("#room-txt").val(roomID);
});

