/**
 * App controller module.
 * @version 1
 */
/* const controller = */
(() => {
    // current app state
    let state;

    // flags
    // first user interaction
    // used for mute/unmute
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
    // keymap
    const keyButtons = {};
    Object.keys(KEY).forEach(button => {
        keyButtons[KEY[button]] = $(`#btn-${KEY[button]}`);
    });

    const setState = (newState) => {
        log.debug(`[control] [s] ${state ? state.name : '???'} -> ${newState.name}`);
        state = newState;
    };

    const onGameRoomAvailable = () => {
        //keyButtons[KEY.JOIN].html('share');
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

    const onLatencyCheckRequest = (data) => {
        popup('Ping check...');
        const timeoutMs = 2000;
        // TODO: why we use maximum timeout
        const maxTimeoutMs = timeoutMs > ajax.defaultTimeoutMs() ? timeoutMs : ajax.defaultTimeoutMs();

        Promise.all((data.addresses || []).map(address => {
            let beforeTime = Date.now();
            return ajax.fetch(`${address}?_=${beforeTime}`, {method: "GET", redirect: "follow"}, timeoutMs)
                .then(() => ({[address]: Date.now() - beforeTime}), () => ({[address]: maxTimeoutMs}));
        })).then(results => {
            // const latencies = Object.assign({}, ...results);
            const latencies = {};
            results.map(latency => Object.keys(latency).forEach(address => latencies[address] = latency[address]));
            log.info('[ping] <->', latencies);
            socket.latency(latencies, data.packetId);
        });
    };

    const helpScreen = {
        // don't call $ if holding the button
        shown: false,
        // undo the state when release the button
        prevState: null,
        // use function () if you need "this"
        show: function (show, event) {
            if (this.shown === show) return;

            // hack
            if (state === app.state.game || this.prevState === app.state.game) {
                gameScreen.toggle(!show);
            } else {
                keyButtons[KEY.SAVE].toggle(show);
                keyButtons[KEY.LOAD].toggle(show);
                menuScreen.toggle(!show);
            }
            helpOverlay.toggle(show);

            this.shown = show;

            if (show) {
                this.prevState = state;
                setState(app.state.help);
            } else {
                setState(this.prevState);
            }

            if (event) event.pub(HELP_OVERLAY_TOGGLED, {shown: show});
        }
    };

    const showMenuScreen = () => {
        // clear scenes
        gameScreen.hide();
        menuScreen.hide();
        gameList.hide();
        keyButtons[KEY.SAVE].hide();
        keyButtons[KEY.LOAD].hide();
        //keyButtons[KEY.JOIN].html('play');

        // show menu scene
        gameScreen.show().delay(0).fadeOut(0, () => {
            log.debug('[control] loading menu screen');
            menuScreen.fadeIn(0, () => {
                gameList.show();
                setState(app.state.menu);
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

        //const el = document.createElement('textarea');
        const playeridx = parseInt($('#playeridx').val(), 10) - 1

        log.info('[control] starting game screen');

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
        socket.startGame(gameList.getCurrentGame(), env.isMobileDevice(), room.getId(), playeridx);

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

    const onKeyPress = (data) => {
        if (data.key == "up" || data.key == "down" || data.key == "left" || data.key == "right") {
            keyButtons[data.key].addClass('dpad-pressed');
        } else {
            keyButtons[data.key].addClass('pressed');
        }

        if (KEY.HELP === data.key) {
            helpScreen.show(true, event);
        }

        state.keyPress(data.key);
    };

    const onKeyRelease = (data) => {
        if (data.key == "up" || data.key == "down" || data.key == "left" || data.key == "right") {
            keyButtons[data.key].removeClass('dpad-pressed');
        } else {
            keyButtons[data.key].removeClass('pressed');
        }

        if (KEY.HELP === data.key) {
            helpScreen.show(false, event);
        }

        // maybe move it somewhere
        if (!interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            interacted = true;
        }

        state.keyRelease(data.key);
    };

    const onAxisChanged = (data) => {
        // maybe move it somewhere
        if (!interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            interacted = true;
        }

        state.axisChanged(data.id, data.value);
    };

    const updatePlayerIndex = (idx) => {
        var slider = document.getElementById('playeridx');
        slider.value = idx + 1;
        socket.updatePlayerIndex(idx);
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
                axisChanged: () => {
                },
                keyPress: () => {
                },
                keyRelease: () => {
                },
                menuReady: () => {
                    showMenuScreen()
                }
            },

            help: {
                name: 'help',
                axisChanged: () => {
                },
                keyPress: () => {
                },
                keyRelease: () => {
                },
                menuReady: () => {
                    // show silently
                    gameScreen.hide();
                    menuScreen.hide();
                    gameList.hide();
                    //keyButtons[KEY.JOIN].html('play');

                    gameList.show();

                    helpScreen.prevState = app.state.menu;
                }
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
                            popup('Let\'s load the game!');
                            break;
                        case KEY.SAVE:
                            popup('Let\'s save the game!');
                            break;
                        case KEY.STATS:
                            event.pub(STATS_TOGGLE);
                            break;
                        case KEY.DTOGGLE:
                            handleToggle();
                            break;
                    }
                },
                menuReady: () => {
                }
            },

            game: {
                name: 'game',
                axisChanged: (id, value) => {
                    input.setAxisChanged(id, value);
                },
                keyPress: (key) => {
                    input.setKeyState(key, true);
                },
                keyRelease: function (key) {
                    input.setKeyState(key, false);

                    switch (key) {
                        // nani? why join / copy switch, it's confusing. Me: It's because of the original design to update label only :-s.
                        case KEY.JOIN: // or SHARE
                            // save when click share
                            event.pub(KEY_PRESSED, {key: KEY.SAVE})
                            room.copyToClipboard();
                            popup('Copy link to clipboard!');
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
                menuReady: () => {
                }
            }
        }
    };

    // subscriptions
    event.sub(GAME_ROOM_AVAILABLE, onGameRoomAvailable, 2);
    event.sub(GAME_SAVED, () => popup('Saved'));
    event.sub(GAME_LOADED, () => popup('Loaded'));
    event.sub(GAME_PLAYER_IDX, (idx) => popup(parseInt(idx)+1));

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
    event.sub(LATENCY_CHECK_REQUESTED, onLatencyCheckRequest);
    event.sub(GAMEPAD_CONNECTED, () => popup('Gamepad connected'));
    event.sub(GAMEPAD_DISCONNECTED, () => popup('Gamepad disconnected'));
    // touch stuff
    event.sub(MENU_HANDLER_ATTACHED, (data) => {
        menuScreen.on(data.event, data.handler);
    });
    event.sub(KEY_PRESSED, onKeyPress);
    event.sub(KEY_RELEASED, onKeyRelease);
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
})($, document, event, env, gameList, input, KEY, log, room, stats, socket, utils);
