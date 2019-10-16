
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

    var elem = $("#ribbon");
    var st = "";
    if (isLayoutSwitched) {
        var st = "rotate(90deg)";
        elem.css("bottom", 0);
        elem.css("top", "");
    } else {
        elem.css("bottom", "");
        elem.css("top", 0);
    }
    elem.css("transform", st);
    elem.css("-webkit-transform", st);
    elem.css("-moz-transform", st);
}

$(window).on("resize", fixScreenLayout);
$(window).on("orientationchange", fixScreenLayout);


$(document).ready(function () {
    fixScreenLayout();

    $("#room-txt").val(roomID);
});

