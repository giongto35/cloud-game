settings.init();
log.setLevel(settings.loadOr(opts.LOG_LEVEL, 'debug'));

keyboard.init();
joystick.init();
touch.init();
stream.init();

[roomId, zone] = room.loadMaybe();
// find worker id if present
const wid = new URLSearchParams(document.location.search).get('wid');
// if from URL -> start game immediately!
socket.init(roomId, wid, zone);
