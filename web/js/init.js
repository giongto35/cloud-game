settings.init();
log.setLevel(settings.loadOr(opts.LOG_LEVEL, 'debug'));

keyboard.init();
joystick.init();
touch.init();

[roomId, zone] = room.loadMaybe();
// if from URL -> start game immediately!
socket.init(roomId, zone);
