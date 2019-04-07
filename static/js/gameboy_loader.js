// A tiny bit of javascript code for color selection
window.onload = function() {
    var colors = ['red', 'purple', 'green', 'yellow', 'teal', 'transparent'];
    var last = null;
    Array.prototype.slice.call(document.querySelectorAll('.color')).forEach(function(el) {
    el.addEventListener('click', function() {
        log("Starting gameboy");
        screenState = "loader";

        if (last) {
        last.classList.remove('active');
        }
        var color = el.getAttribute('data-color');
        var gameboy = document.querySelector('#gameboy');
        gameboy.style.opacity = 0;
        gameboy.classList.remove(gameboy.classList[0]);
        var clone = gameboy.cloneNode(true);
        gameboy.remove();
        clone.classList.add(color);
        clone.style.opacity = 1;
        var colors = document.querySelector('#colors');
        colors.parentNode.insertBefore(clone, colors);
        el.classList.add('active');
        last = el;
    });
    });

    document.querySelector('.color[data-color="green"]').dispatchEvent(new MouseEvent('click', {
    'view': window,
    'bubbles': true
    }));

    $("#screen-gameboy-text").on("webkitAnimationEnd", showMenuScreen);
    $("#screen-gameboy-text").on("animationend", showMenuScreen);
}
