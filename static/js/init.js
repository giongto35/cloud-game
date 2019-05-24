// Window rerender / rotate screen if needed
function fixScreenLayout() {
    var targetWidth = $(document).width();
    var targetHeight = $(document).height();

    // Should have maximum box for desktop?
    targetWidth = 800; targetHeight = 600;

    fixElementLayout($("#gamebody"), targetWidth, targetHeight);
}

$(window).on("resize", fixScreenLayout);
$(window).on("orientationchange", fixScreenLayout);


$(document).ready(function () {
    fixScreenLayout();

});

