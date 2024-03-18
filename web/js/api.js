import {log} from 'log';

const endpoints = {
    LATENCY_CHECK: 3,
    INIT: 4,
    INIT_WEBRTC: 100,
    OFFER: 101,
    ANSWER: 102,
    ICE_CANDIDATE: 103,
    GAME_START: 104,
    GAME_QUIT: 105,
    GAME_SAVE: 106,
    GAME_LOAD: 107,
    GAME_SET_PLAYER_INDEX: 108,
    GAME_RECORDING: 110,
    GET_WORKER_LIST: 111,
    GAME_ERROR_NO_FREE_SLOTS: 112,

    APP_VIDEO_CHANGE: 150,
}

let transport = {
    send: (packet) => {
        log.warn('Default transport is used! Change it with the api.transport variable.', packet)
    },
    keyboard: (packet) => {
        log.warn('Default transport is used! Change it with the api.transport variable.', packet)
    },
    mouse: (packet) => {
        log.warn('Default transport is used! Change it with the api.transport variable.', packet)
    }
}

const packet = (type, payload, id) => {
    const packet = {t: type}
    if (id !== undefined) packet.id = id
    if (payload !== undefined) packet.p = payload
    transport.send(packet)
}

const decodeBytes = (b) => String.fromCharCode.apply(null, new Uint8Array(b))

const keyboardPress = (() => {
    // 0 1 2 3 4 5 6
    // [CODE ] P MOD
    const buffer = new ArrayBuffer(7)
    const dv = new DataView(buffer)

    return (pressed = false, e) => {
        if (e.repeat) return // skip pressed key events

        const key = libretro.mod
        let code = libretro.map('', e.code)
        let shift = e.shiftKey

        // a special Esc for &$&!& Firefox
        if (shift && code === 96) {
            code = 27
            shift = false
        }

        const mod = 0
            | (e.altKey && key.ALT)
            | (e.ctrlKey && key.CTRL)
            | (e.metaKey && key.META)
            | (shift && key.SHIFT)
            | (e.getModifierState('NumLock') && key.NUMLOCK)
            | (e.getModifierState('CapsLock') && key.CAPSLOCK)
            | (e.getModifierState('ScrollLock') && key.SCROLLOCK)
        dv.setUint32(0, code)
        dv.setUint8(4, +pressed)
        dv.setUint16(5, mod)
        transport.keyboard(buffer)
    }
})()

const mouse = {
    MOVEMENT: 0,
    BUTTONS: 1
}

const mouseMove = (() => {
    // 0 1 2 3 4
    // T DX  DY
    const buffer = new ArrayBuffer(5)
    const dv = new DataView(buffer)

    return (dx = 0, dy = 0) => {
        dv.setUint8(0, mouse.MOVEMENT)
        dv.setInt16(1, dx)
        dv.setInt16(3, dy)
        transport.mouse(buffer)
    }
})()

const mousePress = (() => {
    // 0 1
    // T B
    const buffer = new ArrayBuffer(2)
    const dv = new DataView(buffer)

    // 0: Main button pressed, usually the left button or the un-initialized state
    // 1: Auxiliary button pressed, usually the wheel button or the middle button (if present)
    // 2: Secondary button pressed, usually the right button
    // 3: Fourth button, typically the Browser Back button
    // 4: Fifth button, typically the Browser Forward button

    const b2r = [1, 4, 2, 0, 0] // browser mouse button to retro button
    // assumed that only one button pressed / released

    return (button = 0, pressed = false) => {
        dv.setUint8(0, mouse.BUTTONS)
        dv.setUint8(1, pressed ? b2r[button] : 0)
        transport.mouse(buffer)
    }
})()


const libretro = function () {// RETRO_KEYBOARD
    const retro = {
        '': 0,
        'Unidentified': 0,
        'Unknown': 0, // ???
        'First': 0, // ???
        'Backspace': 8,
        'Tab': 9,
        'Clear': 12,
        'Enter': 13, 'Return': 13,
        'Pause': 19,
        'Escape': 27,
        'Space': 32,
        'Exclaim': 33,
        'Quotedbl': 34,
        'Hash': 35,
        'Dollar': 36,
        'Ampersand': 38,
        'Quote': 39,
        'Leftparen': 40, '(': 40,
        'Rightparen': 41, ')': 41,
        'Asterisk': 42,
        'Plus': 43,
        'Comma': 44,
        'Minus': 45,
        'Period': 46,
        'Slash': 47,
        'Digit0': 48,
        'Digit1': 49,
        'Digit2': 50,
        'Digit3': 51,
        'Digit4': 52,
        'Digit5': 53,
        'Digit6': 54,
        'Digit7': 55,
        'Digit8': 56,
        'Digit9': 57,
        'Colon': 58, ':': 58,
        'Semicolon': 59, ';': 59,
        'Less': 60, '<': 60,
        'Equal': 61, '=': 61,
        'Greater': 62, '>': 62,
        'Question': 63, '?': 63,
        // RETROK_AT = 64,
        'BracketLeft': 91, '[': 91,
        'Backslash': 92, '\\': 92,
        'BracketRight': 93, ']': 93,
        // RETROK_CARET = 94,
        // RETROK_UNDERSCORE = 95,
        'Backquote': 96, '`': 96,
        'KeyA': 97,
        'KeyB': 98,
        'KeyC': 99,
        'KeyD': 100,
        'KeyE': 101,
        'KeyF': 102,
        'KeyG': 103,
        'KeyH': 104,
        'KeyI': 105,
        'KeyJ': 106,
        'KeyK': 107,
        'KeyL': 108,
        'KeyM': 109,
        'KeyN': 110,
        'KeyO': 111,
        'KeyP': 112,
        'KeyQ': 113,
        'KeyR': 114,
        'KeyS': 115,
        'KeyT': 116,
        'KeyU': 117,
        'KeyV': 118,
        'KeyW': 119,
        'KeyX': 120,
        'KeyY': 121,
        'KeyZ': 122,
        '{': 123,
        '|': 124,
        '}': 125,
        'Tilde': 126, '~': 126,
        'Delete': 127,

        'Numpad0': 256,
        'Numpad1': 257,
        'Numpad2': 258,
        'Numpad3': 259,
        'Numpad4': 260,
        'Numpad5': 261,
        'Numpad6': 262,
        'Numpad7': 263,
        'Numpad8': 264,
        'Numpad9': 265,
        'NumpadDecimal': 266,
        'NumpadDivide': 267,
        'NumpadMultiply': 268,
        'NumpadSubtract': 269,
        'NumpadAdd': 270,
        'NumpadEnter': 271,
        'NumpadEqual': 272,

        'ArrowUp': 273,
        'ArrowDown': 274,
        'ArrowRight': 275,
        'ArrowLeft': 276,
        'Insert': 277,
        'Home': 278,
        'End': 279,
        'PageUp': 280,
        'PageDown': 281,

        'F1': 282,
        'F2': 283,
        'F3': 284,
        'F4': 285,
        'F5': 286,
        'F6': 287,
        'F7': 288,
        'F8': 289,
        'F9': 290,
        'F10': 291,
        'F11': 292,
        'F12': 293,
        'F13': 294,
        'F14': 295,
        'F15': 296,

        'NumLock': 300,
        'CapsLock': 301,
        'ScrollLock': 302,
        'ShiftRight': 303,
        'ShiftLeft': 304,
        'ControlRight': 305,
        'ControlLeft': 306,
        'AltRight': 307,
        'AltLeft': 308,
        'MetaRight': 309,
        'MetaLeft': 310,
        // RETROK_LSUPER = 311,
        // RETROK_RSUPER = 312,
        // RETROK_MODE = 313,
        // RETROK_COMPOSE = 314,

        // RETROK_HELP = 315,
        // RETROK_PRINT = 316,
        // RETROK_SYSREQ = 317,
        // RETROK_BREAK = 318,
        // RETROK_MENU = 319,
        'Power': 320,
        // RETROK_EURO = 321,
        // RETROK_UNDO = 322,
        // RETROK_OEM_102 = 323,
    }

    const retroMod = {
        NONE: 0x0000,
        SHIFT: 0x01,
        CTRL: 0x02,
        ALT: 0x04,
        META: 0x08,
        NUMLOCK: 0x10,
        CAPSLOCK: 0x20,
        SCROLLOCK: 0x40,
    }

    const _map = (key = '', code = '') => {
        return retro[code] || retro[key] || 0
    }

    return {
        map: _map,
        mod: retroMod,
    }
}()

/**
 * Server API.
 *
 * Requires the actual api.transport implementation.
 */
export const api = {
    set transport(t) {
        transport = t;
    },
    endpoint: endpoints,
    decode: (b) => JSON.parse(decodeBytes(b)),
    server: {
        initWebrtc: () => packet(endpoints.INIT_WEBRTC),
        sendIceCandidate: (candidate) => packet(endpoints.ICE_CANDIDATE, btoa(JSON.stringify(candidate))),
        sendSdp: (sdp) => packet(endpoints.ANSWER, btoa(JSON.stringify(sdp))),
        latencyCheck: (id, list) => packet(endpoints.LATENCY_CHECK, list, id),
        getWorkerList: () => packet(endpoints.GET_WORKER_LIST),
    },
    game: {
        input: {
            keyboard: {
                press: keyboardPress,
            },
            mouse: {
                move: mouseMove,
                press: mousePress,
            }
        },
        load: () => packet(endpoints.GAME_LOAD),
        save: () => packet(endpoints.GAME_SAVE),
        setPlayerIndex: (i) => packet(endpoints.GAME_SET_PLAYER_INDEX, i),
        start: (game, roomId, record, recordUser, player) => packet(endpoints.GAME_START, {
            game_name: game,
            room_id: roomId,
            player_index: player,
            record: record,
            record_user: recordUser,
        }),
        toggleRecording: (active = false, userName = '') =>
            packet(endpoints.GAME_RECORDING, {active: active, user: userName}),
        quit: (roomId) => packet(endpoints.GAME_QUIT, {room_id: roomId}),
    }
}
