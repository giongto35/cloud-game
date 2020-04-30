settings.init();
log.setLevel(settings.loadOr('log.level', 'debug'));

keyboard.init();
joystick.init();
touch.init();

[roomId, zone] = room.loadMaybe();
// if from URL -> start game immediately!
socket.init(roomId, zone);
