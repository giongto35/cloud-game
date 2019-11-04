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
        keyButtons[KEY.JOIN].html('share');
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
        const maxTimeoutMs = timeoutMs > ajax.defaultTimeoutMs() ? timeoutMs : ajax.defaultTimeoutMs();

        Promise.all((data.addresses || []).map(address => {
            let beforeTime = Date.now();
            return ajax.fetch(`http://${address}:9000/echo?_=${beforeTime}`, {}, timeoutMs)
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
        show: function (show) {
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
        }
    };

    const showMenuScreen = () => {
        // clear scenes
        gameScreen.hide();
        menuScreen.hide();
        gameList.hide();
        keyButtons[KEY.SAVE].hide();
        keyButtons[KEY.LOAD].hide();
        keyButtons[KEY.JOIN].html('play');

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
        socket.startGame(gameList.getCurrentGame(), env.isMobileDevice(), room.getId(), 1);

        // clear menu screen
        input.poll().disable();
        menuScreen.hide();
        gameScreen.show();
        keyButtons[KEY.SAVE].show();
        keyButtons[KEY.LOAD].show();
        // end clear
        input.poll().enable();
    };

    // !to add debounce
    const popup = (msg) => {
        popupBox.html(msg);
        popupBox.fadeIn().delay(0).fadeOut();
    };

    const onKeyPress = (data) => {
        keyButtons[data.key].addClass('pressed');

        if (KEY.HELP === data.key) helpScreen.show(true);

        state.keyPress(data.key);
    };

    const onKeyRelease = (data) => {
        keyButtons[data.key].removeClass('pressed');

        if (KEY.HELP === data.key) helpScreen.show(false);

        // maybe move it somewhere
        if (!interacted) {
            // unmute when there is user interaction
            gameScreen[0].muted = false;
            interacted = true;
        }

        state.keyRelease(data.key);
    };

    const app = {
        state: {
            eden: {
                name: 'eden',
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
                keyPress: () => {
                },
                keyRelease: () => {
                },
                menuReady: () => {
                    // show silently
                    gameScreen.hide();
                    menuScreen.hide();
                    gameList.hide();
                    keyButtons[KEY.JOIN].html('play');

                    gameList.show();

                    helpScreen.prevState = app.state.menu;
                }
            },

            menu: {
                name: 'menu',
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
                            popup('Lets play to load game!');
                            break;
                        case KEY.SAVE:
                            popup('Lets play to save game!');
                            break;
                    }
                },
                menuReady: () => {
                }
            },

            game: {
                name: 'game',
                keyPress: (key) => {
                    input.setKeyState(key, true);
                },
                keyRelease: function (key) {
                    input.setKeyState(key, false);

                    switch (key) {
                        // nani? why join / copy switch, it's confusing
                        case KEY.JOIN:
                            room.copyToClipboard();
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
    event.sub(MEDIA_STREAM_INITIALIZED, (data) => {
        rtcp.start(data.stunturn);
        gameList.set(data.games);
    });
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
    event.sub(KEY_PRESSED, onKeyPress);
    event.sub(KEY_RELEASED, onKeyRelease);
    event.sub(KEY_STATE_UPDATED, data => rtcp.input(data));

    // initial app state
    setState(app.state.eden);

})($, document, event, env, gameList, input, KEY, log, room);
