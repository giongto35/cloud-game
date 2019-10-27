log.setLevel('debug');

$(document).ready(() => {
    env.display().fixScreenLayout();

    keyboard.init();
    joystick.init();
    touch.init();

    const roomId = room.loadMaybe();
    // if from URL -> start game immediately!
    socket.init(roomId);
});
