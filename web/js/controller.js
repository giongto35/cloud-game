/**
 * App controller module.
 * @version 1
 */
/*const controller = */
(() => {
    const state = {
        screenState: 'loader',
        // flags first key press of a user
        interacted: false
    };

    // UI elements
    // use $element[0] for DOM element
    const gameScreen = $('#game-screen');
    const menuScreen = $('#menu-screen');
    const saveButton = $('#btn-save');
    const loadButton = $('#btn-load');
    const joinButton = $('#btn-join');
    const helpOverlay = $('#help-overlay');
    const popupBox = $("#noti-box");

    const onGameRoomAvailable = () => {
        joinButton.html('share');
        popup('Started! You can share you game!');
    };

    const onConnectionReady = () => {
        if (room.getId()) startGame();
        if (state.screenState !== 'game') showMenuScreen();
    };

    const onMediaStreamInitialize = (data) => {
        // TODO: Read from struct
        // init package has 2 part [stunturn, game1, game2, game3 ...]
        // const [stunturn, ...games] = data;
        rtcp.start(data.data[0]);
        data.data.shift();
        gameList.set(data.data);
    };

    const onLatencyCheckRequest = (data) => {
        popup('Checking latency...');
        const timeoutMs = 2000;
        Promise.all((data.addresses || []).map(address => {
            let beforeTime = Date.now();
            return ajax.fetch(`http://${address}:9000/echo?_=${beforeTime}`, {}, timeoutMs)
                .then(() => ({[address]: Date.now() - beforeTime}), () => ({[address]: ajax.timeout()}));
        })).then(results => {
            // const latencies = Object.assign({}, ...results);
            const latencies = {};
            results.map(latency => Object.keys(latency).forEach(address => latencies[address] = latency[address]));
            log.info('[ping] <->', latencies);
            socket.latency(latencies, data.packetId);
        });
    };

    const toggleHelp = (show) => {
        if (state.screenState === 'menu') {
            saveButton.toggle(show);
            loadButton.toggle(show);
            menuScreen.toggle(!show);
        } else {
            gameScreen.toggle(!show);
        }

        helpOverlay.toggle(show);
    };

    const showMenuScreen = () => {
        // clear scenes
        gameScreen.hide();
        menuScreen.hide();
        gameList.hide();
        saveButton.hide();
        loadButton.hide();
        joinButton.html('play');

        // show menu scene
        gameScreen.show().delay(0).fadeOut(0, function () {
            log.debug('[control] loading menu screen');
            menuScreen.fadeIn(0, function () {
                gameList.show();
                state.screenState = 'menu';
            });
        });
    };

    const startGame = () => {
        if (!rtcp.isConnected()) {
            popup('Game cannot load. Please refresh');
            return;
        }

        if (!rtcp.isInputReady()) {
            popup('Game is not ready yet. Please wait');
            return;
        }

        log.info('[control] starting game screen');
        state.screenState = 'game';

        gameScreen.muted = false;
        const promise = gameScreen[0].play();
        if (promise !== undefined) {
            promise.then(_ => log.info('Media can autoplay'))
                .catch(error => {
                    // Usually error happens when we autoplay unmuted video, browser requires manual play.
                    // We already muted video and use separate audio encoding so it's fine now
                    log.info('Media Failed to autoplay');
                    log.info(error)
                    // TODO: Consider workaround
                });
        }

        // TODO get current game from the URL and not from the list?
        // if we are opening a share link it will send the default game name to the server
        // currently it's a game with the index 1
        // on the server this game is ignored and the actual game will be extracted from the share link
        // so there's no point in doing this and this' really confusing
        socket.startGame(gameList.getCurrentGame(), env.isMobileDevice(), room.getId(), 1);

        // clear menu screen
        input.poll().disable();
        menuScreen.hide();
        gameScreen.show();
        saveButton.show();
        loadButton.show();
        // end clear
        input.poll().enable();
    };

    const popup = (msg) => {
        popupBox.html(msg);
        popupBox.fadeIn().delay(0).fadeOut();
    };

    const copyToClipboard = (text) => {
        const el = document.createElement('textarea');
        el.value = text;
        document.body.appendChild(el);
        el.select();
        document.execCommand('copy');
        document.body.removeChild(el);
    };

    const doButtonDown = (name) => {
        $(`#btn-${name}`).addClass('pressed');

        if (state.screenState === 'menu') {
            if (name === KEY.UP || name === KEY.DOWN) {
                gameList.startGamePickerTimer(name === KEY.UP);
            }
        } else if (state.screenState === 'game') {
            input.setKeyState(name, true);
        }

        if (name === KEY.HELP) toggleHelp(true);
    };

    const doButtonUp = (name) => {
        $(`#btn-${name}`).removeClass('pressed');

        // log.debug(`[control] pressed: ${name}`);

        if (state.screenState === 'menu') {
            switch (name) {
                case KEY.UP:
                case KEY.DOWN:
                    gameList.stopGamePickerTimer();
                    break;
                case KEY.JOIN:
                case KEY.A:
                case KEY.B:
                case KEY.X:
                case KEY.Y:
                case KEY.START:
                case KEY.SELECT:
                    startGame();
                    break;
                case KEY.QUIT:
                    popup('You are already in menu screen!');
                    break;
                case KEY.LOAD:
                    popup('Lets play to load game!');
                    break;
                case KEY.SAVE:
                    popup('Lets play to save game!');
                    break;
            }
        } else if (state.screenState === 'game') {
            input.setKeyState(name, false);

            switch (name) {
                case KEY.JOIN:
                    copyToClipboard(room.getLink());
                    popup('Copy link to clipboard!');
                    break;
                case KEY.SAVE:
                    socket.saveGame();
                    break;
                case KEY.LOAD:
                    socket.loadGame();
                    break;
                case KEY.FULL:
                    env.display().toggleFullscreen(gameScreen.height() !== window.innerHeight, gameScreen[0]);
                    break;
                case KEY.QUIT:
                    input.poll().disable();

                    // TODO: Stop game
                    socket.quitGame(room.getId());
                    room.reset();

                    popup('Quit!');

                    location.reload();
                    break;
            }
        }

        if (name === KEY.HELP) toggleHelp(false);
    };

    // subscriptions
    event.sub(GAME_ROOM_AVAILABLE, onGameRoomAvailable, 2);
    event.sub(GAME_SAVED, () => popup('Saved'));
    event.sub(GAME_LOADED, () => popup('Loaded'));
    event.sub(MEDIA_STREAM_INITIALIZED, onMediaStreamInitialize);
    event.sub(MEDIA_STREAM_SDP_AVAILABLE, (data) => rtcp.setRemoteDescription(data.sdp, gameScreen[0]));
    event.sub(MEDIA_STREAM_READY, () => rtcp.start());
    event.sub(CONNECTION_READY, onConnectionReady);
    event.sub(CONNECTION_CLOSED, () => input.poll().disable());
    event.sub(LATENCY_CHECK_REQUESTED, onLatencyCheckRequest);
    event.sub(GAMEPAD_CONNECTED, () => popup('Gamepad connected'));
    event.sub(GAMEPAD_DISCONNECTED, () => popup('Gamepad disconnected'));
    // touch stuff
    event.sub(MENU_HANDLER_ATTACHED, (data) => {
        menuScreen.on(data.event, data.handler);
    });
    event.sub(KEY_PRESSED, (data) => doButtonDown(data.key));
    event.sub(KEY_RELEASED, (data) => {
        if (!state.interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            state.interacted = true;
        }

        doButtonUp(data.key)
    });
    event.sub(KEY_STATE_UPDATED, data => rtcp.input(data));

})($, room, event, env, gameList, input, log, KEY);
