/**
 * Touch controls.
 *
 * Virtual Gamepad / Joystick
 * Left panel - Dpad
 *
 * @link https://jsfiddle.net/aa0et7tr/5/
 * @version 1
 */
const touch = (() => {
    const MAX_DIFF = 20; // radius of circle boundary

    // vpad state, use for mouse button down
    let vpadState = {[KEY.UP]: false, [KEY.DOWN]: false, [KEY.LEFT]: false, [KEY.RIGHT]: false};
    let analogState = [0, 0];

    let vpadTouchIdx = null;
    let vpadTouchDrag = null;
    let vpadHolder = $("#circle-pad-holder");
    let vpadCircle = $("#circle-pad");

    const window_ = $(window);
    const buttons = $(".btn");
    const playerSlider = $("#playeridx")
    const dpad = $(".dpad");

    const dpadToggle = document.getElementById('dpad-toggle')
    dpadToggle.addEventListener('change', (e) => {
      event.pub(DPAD_TOGGLE, {checked: e.target.checked});
    });

    let dpadMode = true;
    const deadZone = 0.1;

    function onDpadToggle(checked) {
      if (dpadMode === checked) {
        return //error?
      }
      if (dpadMode) {
        dpadMode = false;
        vpadHolder.addClass('dpad-empty');
        vpadCircle.addClass('bong-full');
        // reset dpad keys pressed before moving to analog stick mode
        resetVpadState()
      } else {
        dpadMode = true;
        vpadHolder.removeClass('dpad-empty');
        vpadCircle.removeClass('bong-full');
      }
    }

    function resetVpadState() {
        if (dpadMode) {
            // trigger up event?
            checkVpadState(KEY.UP, false);
            checkVpadState(KEY.DOWN, false);
            checkVpadState(KEY.LEFT, false);
            checkVpadState(KEY.RIGHT, false);
        } else {
            checkAnalogState(0, 0);
            checkAnalogState(1, 0);
        }

        vpadTouchDrag = null;
        vpadTouchIdx = null;
        dpad.removeClass('pressed');
    }

    function checkVpadState(axis, state) {
        if (state !== vpadState[axis]) {
            vpadState[axis] = state;
            event.pub(state ? KEY_PRESSED : KEY_RELEASED, {key: axis});
        }
    }

    function checkAnalogState(axis, value) {
        if (-deadZone < value && value < deadZone) value = 0;
        if (analogState[axis] !== value) {
            analogState[axis] = value;
            event.pub(AXIS_CHANGED, {id: axis, value: value});
        }
    }

    function handleVpadJoystickDown(event) {
        vpadCircle.css('transition', '0s');
        vpadCircle.css('-moz-transition', '0s');
        vpadCircle.css('-webkit-transition', '0s');

        if (event.changedTouches) {
            resetVpadState();
            vpadTouchIdx = event.changedTouches[0].identifier;
            event.clientX = event.changedTouches[0].clientX;
            event.clientY = event.changedTouches[0].clientY;
        }

        vpadTouchDrag = {x: event.clientX, y: event.clientY};
    }

    function handleVpadJoystickUp() {
        if (vpadTouchDrag === null) return;

        vpadCircle.css('transition', '.2s');
        vpadCircle.css('-moz-transition', '.2s');
        vpadCircle.css('-webkit-transition', '.2s');
        vpadCircle.css('transform', 'translate3d(0px, 0px, 0px)');
        vpadCircle.css('-moz-transform', 'translate3d(0px, 0px, 0px)');
        vpadCircle.css('-webkit-transform', 'translate3d(0px, 0px, 0px)');

        resetVpadState();
    }

    function handleVpadJoystickMove(event) {
        if (vpadTouchDrag === null) return;

        if (event.changedTouches) {
            // check if moving source is from other touch?
            for (let i = 0; i < event.changedTouches.length; i++) {
                if (event.changedTouches[i].identifier === vpadTouchIdx) {
                    event.clientX = event.changedTouches[i].clientX;
                    event.clientY = event.changedTouches[i].clientY;
                }
            }
            if (event.clientX === undefined || event.clientY === undefined)
                return;
        }

        let xDiff = event.clientX - vpadTouchDrag.x;
        let yDiff = event.clientY - vpadTouchDrag.y;
        let angle = Math.atan2(yDiff, xDiff);
        let distance = Math.min(MAX_DIFF, Math.hypot(xDiff, yDiff));
        let xNew = distance * Math.cos(angle);
        let yNew = distance * Math.sin(angle);

        if (env.display().isLayoutSwitched) {
            let tmp = xNew;
            xNew = yNew;
            yNew = -tmp;
        }

        let style = `translate(${xNew}px, ${yNew}px)`;
        vpadCircle.css('transform', style);
        vpadCircle.css('-webkit-transform', style);
        vpadCircle.css('-moz-transform', style);

        let xRatio = xNew / MAX_DIFF;
        let yRatio = yNew / MAX_DIFF;

        if (dpadMode) {
            checkVpadState(KEY.LEFT, xRatio <= -0.5);
            checkVpadState(KEY.RIGHT, xRatio >= 0.5);
            checkVpadState(KEY.UP, yRatio <= -0.5);
            checkVpadState(KEY.DOWN, yRatio >= 0.5);
        } else {
            checkAnalogState(0, xRatio);
            checkAnalogState(1, yRatio);
        }
    }

    /*
        Right side - Control buttons
    */

    function handleButtonDown() {
        checkVpadState($(this).attr('value'), true);
        // add touchIdx?
    }

    function handleButtonUp() {
        checkVpadState($(this).attr('value'), false);
    }


    /*
        Player index slider
    */

    function handlePlayerSlider() {
            socket.updatePlayerIndex($(this).val() - 1);
    }


    /*
        Touch menu
    */

    let menuTouchIdx = null;
    let menuTouchDrag = null;
    let menuTouchTime = null;

    function handleMenuDown(event) {
        // Identify of touch point
        if (event.changedTouches) {
            menuTouchIdx = event.changedTouches[0].identifier;
            event.clientX = event.changedTouches[0].clientX;
            event.clientY = event.changedTouches[0].clientY;
        }

        menuTouchDrag = {x: event.clientX, y: event.clientY,};
        menuTouchTime = Date.now();
    }

    function handleMenuMove(evt) {
        if (menuTouchDrag === null) return;

        if (evt.changedTouches) {
            // check if moving source is from other touch?
            for (let i = 0; i < evt.changedTouches.length; i++) {
                if (evt.changedTouches[i].identifier === menuTouchIdx) {
                    evt.clientX = evt.changedTouches[i].clientX;
                    evt.clientY = evt.changedTouches[i].clientY;
                }
            }
            if (evt.clientX === undefined || evt.clientY === undefined)
                return;
        }

        const pos = env.display().isLayoutSwitched ? evt.clientX - menuTouchDrag.x : menuTouchDrag.y - evt.clientY;
        event.pub(MENU_PRESSED, pos);
    }

    function handleMenuUp(evt) {
        if (menuTouchDrag === null) return;
        if (evt.changedTouches) {
            if (evt.changedTouches[0].identifier !== menuTouchIdx)
                return;
            evt.clientX = evt.changedTouches[0].clientX;
            evt.clientY = evt.changedTouches[0].clientY;
        }

        let newY = env.display().isLayoutSwitched ? -menuTouchDrag.x + evt.clientX : menuTouchDrag.y - evt.clientY;

        let interval = Date.now() - menuTouchTime; // 100ms?
        if (interval < 200) {
            // calc velo
            newY = newY / interval * 250;
        }

        // current item?
        event.pub(MENU_RELEASED, newY);
        menuTouchDrag = null;
    }

    /*
        Common events
    */
    function handleWindowMove(event) {
        event.preventDefault();
        handleVpadJoystickMove(event);
        handleMenuMove(event);

        // moving touch
        if (event.changedTouches) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                if (event.changedTouches[i].identifier !== menuTouchIdx && event.changedTouches[i].identifier !== vpadTouchIdx) {
                    // check class

                    let elem = document.elementFromPoint(event.changedTouches[i].clientX, event.changedTouches[i].clientY);
                    if (elem.classList.contains('btn')) {
                        $(elem).trigger('touchstart');
                    } else {
                        buttons.trigger('touchend');
                    }
                }
            }
        }
    }

    function handleWindowUp(ev) {
        handleVpadJoystickUp(ev);
        handleMenuUp(ev);
        buttons.trigger('touchend');
    }

    // touch/mouse events for control buttons. mouseup events is binded to window.
    buttons.on('mousedown', handleButtonDown);
    buttons.on('touchstart', handleButtonDown);
    buttons.on('touchend', handleButtonUp);

    // touch/mouse events for dpad. mouseup events is binded to window.
    vpadHolder.on('mousedown', handleVpadJoystickDown);
    vpadHolder.on('touchstart', handleVpadJoystickDown);
    vpadHolder.on('touchend', handleVpadJoystickUp);

    // touch/mouse events for player slider.
    playerSlider.on('oninput', handlePlayerSlider);
    playerSlider.on('onchange', handlePlayerSlider);
    playerSlider.on('mouseup', handlePlayerSlider);

    // Bind events for menu
    // TODO change this flow
    event.pub(MENU_HANDLER_ATTACHED, {event: 'mousedown', handler: handleMenuDown});
    event.pub(MENU_HANDLER_ATTACHED, {event: 'touchstart', handler: handleMenuDown});
    event.pub(MENU_HANDLER_ATTACHED, {event: 'touchend', handler: handleMenuUp});

    event.sub(DPAD_TOGGLE, (data) => onDpadToggle(data.checked));

    return {
        init: () => {
            // add buttons into the state 🤦
            $('.btn, .btn-big').each((_, el) => {
                vpadState[$(el).attr('value')] = false;
            });

            // Bind events for window
            window_.on('mousemove', handleWindowMove);
            window_[0].addEventListener('touchmove', handleWindowMove, {passive: false});
            window_.on('mouseup', handleWindowUp);

            log.info('[input] touch input has been initialized');
        }
    }
})($, document, event, KEY, window);
