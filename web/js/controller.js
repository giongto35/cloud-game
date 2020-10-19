/**
 * App controller module.
 * @version 1
 */
(() => {
    // application state
    let state;
    let lastState;

    // flags
    // first user interaction
    let interacted = false;

    const DIR = (() => {
        return {
            IDLE: 'idle',
            UP: 'up',
            DOWN: 'down',
        }
    })();
    let prevDir = DIR.IDLE;

    // UI elements
    // use $element[0] for DOM element
    const gameScreen = $('#game-screen');
    const menuScreen = $('#menu-screen');
    const helpOverlay = $('#help-overlay');
    const popupBox = $('#noti-box');
    const playerIndex = document.getElementById('playeridx');

    // keymap
    const keyButtons = {};
    Object.keys(KEY).forEach(button => {
        keyButtons[KEY[button]] = $(`#btn-${KEY[button]}`);
    });

    /**
     * State machine transition.
     * @param newState A new state strictly from app.state.*
     * @example
     * setState(app.state.eden)
     */
    const setState = (newState = app.state.eden) => {
        if (newState === state) return;

        const prevState = state;

        // keep the current state intact for one of the "uber" states
        if (state && state._uber) {
            // if we are done with the uber state
            if (lastState === newState) state = newState;
            lastState = newState;
        } else {
            lastState = state
            state = newState;
        }

        if (log.is(log.level.debug)) {
            const previous = prevState ? prevState.name : '???';
            const current = state ? state.name : '???';
            const kept = lastState ? lastState.name : '???';

            log.debug(`[state] ${previous} -> ${current} [${kept}]`);
        }
    };

    const onGameRoomAvailable = () => {
        popup('Now you can share you game!');
    };

    const onConnectionReady = () => {
        // start a game right away or show the menu
        if (room.getId()) {
            startGame();
        } else {
            state.menuReady();
        }
    };

    const onLatencyCheck = (data) => {
        popup('Connecting to fastest server...');
        const timeoutMs = 1111;
        // deduplicate
        const addresses = [...new Set(data.addresses || [])];

        Promise.all(addresses.map(address => {
            const start = Date.now();
            return ajax.fetch(`${address}?_=${start}`, {method: "GET", redirect: "follow"}, timeoutMs)
                .then(() => ({[address]: Date.now() - start}))
                .catch(() => ({[address]: 9999}));
        })).then(servers => {
            const latencies = Object.assign({}, ...servers);
            log.info('[ping] <->', latencies);
            socket.latency(latencies, data.packetId);
        });
    };

    const helpScreen = {
        // don't call $ if holding the button
        shown: false,
        // use function () if you need "this"
        show: function (show, event) {
            if (this.shown === show) return;

            if (state === app.state.game) {
                gameScreen.toggle(!show);
            } else {
                menuScreen.toggle(!show);
            }

            keyButtons[KEY.SAVE].toggle(show);
            keyButtons[KEY.LOAD].toggle(show);

            helpOverlay.toggle(show);

            this.shown = show;

            if (event) event.pub(HELP_OVERLAY_TOGGLED, {shown: show});
        }
    };

    const showMenuScreen = () => {
        log.debug('[control] loading menu screen');

        gameScreen.hide();
        keyButtons[KEY.SAVE].hide();
        keyButtons[KEY.LOAD].hide();

        gameList.show();
        menuScreen.show();

        setState(app.state.menu);
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

        log.info('[control] game start');

        setState(app.state.game);

        const promise = gameScreen[0].play();
        if (promise !== undefined) {
            promise.then(() => log.info('Media can autoplay'))
                .catch(error => {
                    // Usually error happens when we autoplay unmuted video, browser requires manual play.
                    // We already muted video and use separate audio encoding so it's fine now
                    log.error('Media Failed to autoplay');
                    log.error(error)
                    // TODO: Consider workaround
                });
        }

        // TODO get current game from the URL and not from the list?
        // if we are opening a share link it will send the default game name to the server
        // currently it's a game with the index 1
        // on the server this game is ignored and the actual game will be extracted from the share link
        // so there's no point in doing this and this' really confusing
        socket.startGame(gameList.getCurrentGame(), env.isMobileDevice(), room.getId(), +playerIndex.value - 1);

        // clear menu screen
        input.poll().disable();
        menuScreen.hide();
        gameScreen.show();
        keyButtons[KEY.SAVE].show();
        keyButtons[KEY.LOAD].show();
        // end clear
        input.poll().enable();
    };

    const saveGame = utils.debounce(socket.saveGame, 1000);
    const loadGame = utils.debounce(socket.loadGame, 1000);

    const _popup = (message) => popupBox.html(message).fadeIn().fadeOut();
    const popup = utils.throttle(_popup, 1000);

    const _dpadArrowKeys = [KEY.UP, KEY.DOWN, KEY.LEFT, KEY.RIGHT];

    // pre-state key press handler
    const onKeyPress = (data) => {
        const button = keyButtons[data.key];

        if (_dpadArrowKeys.includes(data.key)) {
            button.addClass('dpad-pressed');
        } else {
            if (button) button.addClass('pressed');
        }

        if (state !== app.state.settings) {
            if (KEY.HELP === data.key) helpScreen.show(true, event);
        }

        state.keyPress(data.key);
    };

    // pre-state key release handler
    const onKeyRelease = data => {
        const button = keyButtons[data.key];

        if (!button) return;

        if (_dpadArrowKeys.includes(data.key)) {
            button.removeClass('dpad-pressed');
        } else {
            if (button) button.removeClass('pressed');
        }

        if (state !== app.state.settings) {
            if (KEY.HELP === data.key) helpScreen.show(false, event);
        }

        // maybe move it somewhere
        if (!interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            interacted = true;
        }

        // change app state if settings
        if (KEY.SETTINGS === data.key) setState(app.state.settings);

        state.keyRelease(data.key);
    };

    const updatePlayerIndex = idx => {
        playerIndex.value = idx + 1;
        socket.updatePlayerIndex(idx);
    };

    // noop function for the state
    const _nil = () => {
    }

    const onAxisChanged = (data) => {
        // maybe move it somewhere
        if (!interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            interacted = true;
        }

        state.axisChanged(data.id, data.value);
    };

    const handleToggle = () => {
        var toggle = document.getElementById('dpad-toggle');
        toggle.checked = !toggle.checked;
        event.pub(DPAD_TOGGLE, {checked: toggle.checked});
    };

    const app = {
        state: {
            eden: {
                name: 'eden',
                axisChanged: _nil,
                keyPress: _nil,
                keyRelease: _nil,
                menuReady: showMenuScreen
            },

            settings: {
                _uber: true,
                name: 'settings',
                axisChanged: _nil,
                keyPress: _nil,
                keyRelease: key => {
                    if (key === KEY.SETTINGS) {
                        const isSettingsOpened = settings.ui.toggle();
                        if (!isSettingsOpened) setState(lastState);
                    }
                },
                menuReady: showMenuScreen
            },

            menu: {
                name: 'menu',
                axisChanged: (id, value) => {
                    if (id === 1) { // Left Stick, Y Axis
                        let dir = DIR.IDLE;
                        if (value < -0.5) dir = DIR.UP;
                        if (value > 0.5) dir = DIR.DOWN;
                        if (dir !== prevDir) {
                            prevDir = dir;
                            switch (dir) {
                                case DIR.IDLE:
                                    gameList.stopGamePickerTimer();
                                    break;
                                case DIR.UP:
                                    gameList.startGamePickerTimer(true);
                                    break;
                                case DIR.DOWN:
                                    gameList.startGamePickerTimer(false);
                                    break;
                            }
                        }
                    }
                },
                keyPress: (key) => {
                    switch (key) {
                        case KEY.UP:
                        case KEY.DOWN:
                            gameList.startGamePickerTimer(key === KEY.UP);
                            break;
                    }
                },
                keyRelease: (key) => {
                    switch (key) {
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
                            popup('Loading the game.');
                            break;
                        case KEY.SAVE:
                            popup('Saving the game.');
                            break;
                        case KEY.STATS:
                            event.pub(STATS_TOGGLE);
                            break;
                        case KEY.SETTINGS:
                            break;
                        case KEY.DTOGGLE:
                            handleToggle();
                            break;
                    }
                },
                menuReady: _nil
            },

            game: {
                name: 'game',
                axisChanged: (id, value) => {
                    input.setAxisChanged(id, value);
                },
                keyPress: key => {
                    input.setKeyState(key, true);
                },
                keyRelease: function (key) {
                    input.setKeyState(key, false);

                    switch (key) {
                        case KEY.JOIN: // or SHARE
                            // save when click share
                            saveGame();
                            room.copyToClipboard();
                            popup('Shared link copied to the clipboard!');
                            break;
                        case KEY.SAVE:
                            saveGame();
                            break;
                        case KEY.LOAD:
                            loadGame();
                            break;
                        case KEY.FULL:
                            env.display().toggleFullscreen(gameScreen.height() !== window.innerHeight, gameScreen[0]);
                            break;

                        // update player index
                        case KEY.PAD1:
                            updatePlayerIndex(0);
                            break;
                        case KEY.PAD2:
                            updatePlayerIndex(1);
                            break;
                        case KEY.PAD3:
                            updatePlayerIndex(2);
                            break;
                        case KEY.PAD4:
                            updatePlayerIndex(3);
                            break;

                        // toggle multitap
                        case KEY.MULTITAP:
                            socket.toggleMultitap();
                            break;

                        // quit
                        case KEY.QUIT:
                            input.poll().disable();

                            // TODO: Stop game
                            socket.quitGame(room.getId());
                            room.reset();

                            popup('Quit!');

                            window.location = window.location.pathname;
                            break;

                        case KEY.STATS:
                            event.pub(STATS_TOGGLE);
                            break;
                        case KEY.DTOGGLE:
                            handleToggle();
                            break;
                    }
                },
                menuReady: _nil
            }
        }
    };

    // subscriptions
    event.sub(GAME_ROOM_AVAILABLE, onGameRoomAvailable, 2);
    event.sub(GAME_SAVED, () => popup('Saved'));
    event.sub(GAME_LOADED, () => popup('Loaded'));
    event.sub(GAME_PLAYER_IDX, idx => popup(+idx + 1));

    event.sub(MEDIA_STREAM_INITIALIZED, (data) => {
        rtcp.start(data.stunturn);
        gameList.set(data.games);
    });
    event.sub(MEDIA_STREAM_SDP_AVAILABLE, (data) => rtcp.setRemoteDescription(data.sdp, gameScreen[0]));
    event.sub(MEDIA_STREAM_CANDIDATE_ADD, (data) => rtcp.addCandidate(data.candidate));
    event.sub(MEDIA_STREAM_CANDIDATE_FLUSH, () => rtcp.flushCandidate());
    event.sub(MEDIA_STREAM_READY, () => rtcp.start());
    event.sub(CONNECTION_READY, onConnectionReady);
    event.sub(CONNECTION_CLOSED, () => input.poll().disable());
    event.sub(LATENCY_CHECK_REQUESTED, onLatencyCheck);
    event.sub(GAMEPAD_CONNECTED, () => popup('Gamepad connected'));
    event.sub(GAMEPAD_DISCONNECTED, () => popup('Gamepad disconnected'));
    // touch stuff
    event.sub(MENU_HANDLER_ATTACHED, (data) => {
        menuScreen.on(data.event, data.handler);
    });
    event.sub(KEY_PRESSED, onKeyPress);
    event.sub(KEY_RELEASED, onKeyRelease);
    event.sub(KEY_STATE_UPDATED, data => rtcp.input(data));
    event.sub(SETTINGS_CHANGED, () => popup('Settings have been updated'));
    event.sub(SETTINGS_CLOSED, () => {
        state.keyRelease(KEY.SETTINGS);
    });
    event.sub(AXIS_CHANGED, onAxisChanged);
    event.sub(CONTROLLER_UPDATED, data => rtcp.input(data));

    // game screen stuff
    gameScreen.on('loadstart', () => {
        gameScreen[0].volume = 0.5;
        gameScreen[0].poster = '/static/img/screen_loading.gif';
    });
    gameScreen.on('canplay', () => {
        gameScreen[0].poster = '';
    });

    // initial app state
    setState(app.state.eden);
})($, document, event, env, gameList, input, KEY, log, room, settings, socket, stats, utils);
