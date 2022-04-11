settings.init();

(() => {
    let lvl = settings.loadOr(opts.LOG_LEVEL, log.DEFAULT);
    // migrate old log level options
    // !to remove at some point
    if (isNaN(lvl)) {
        console.warn(
            `The log value [${lvl}] is not supported! ` +
            `The default value [debug] will be used instead.`);
        settings.set(opts.LOG_LEVEL, `${log.DEFAULT}`)
        lvl = log.DEFAULT
    }
    log.level = lvl
})();

keyboard.init();
joystick.init();
touch.init();
stream.init();

[roomId, zone] = room.loadMaybe();
// find worker id if present
const wid = new URLSearchParams(document.location.search).get('wid');
// if from URL -> start game immediately!
socket.init(roomId, wid, zone);
