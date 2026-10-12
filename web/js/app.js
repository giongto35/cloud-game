import {log} from 'log';
import {opts, settings} from 'settings';
import {api} from 'api';
import {
    APP_VIDEO_CHANGED,
    AXIS_CHANGED,
    CONTROLLER_UPDATED,
    DPAD_TOGGLE,
    GAMEPAD_CONNECTED,
    GAMEPAD_DISCONNECTED,
    KB_MOUSE_FLAG,
    KEY_PRESSED,
    KEY_RELEASED,
    KEYBOARD_KEY_DOWN,
    KEYBOARD_KEY_UP,
    MESSAGE,
    MOUSE_MOVED,
    MOUSE_PRESSED,
    SETTINGS_CHANGED,
    pub,
    sub,
} from 'event';
import {input, isTouchDevice, joystick, JOYPAD_KEYS, KEY, keyboard, pointer, touch} from 'input';
import {ajax, socket, webrtc} from 'network';

import {stream} from './stream.js?v=4';
import {http} from './http.js?v=4';
import {icon} from './ui/icons.js?v=4';
import {avatar, copyText, esc, html, isTyping} from './ui/dom.js?v=4';
import {toast} from './ui/toast.js?v=4';
import {confirmDialog, infoDialog, modal} from './ui/modal.js?v=4';
import {nav} from './ui/nav.js?v=4';
import {library, rememberGame} from './views/library.js?v=4';
import {live} from './views/live.js?v=4';
import {admin} from './views/admin.js?v=4';
import {tournaments} from './views/tournaments.js?v=4';
import {gameView} from './views/game.js?v=4';
import {queueView} from './views/queue.js?v=4';
import {settingsView, showHelp} from './views/settings.js?v=4';
import {gameTitle, roles, systemName} from './i18n.js?v=4';

// Settings & defaults

settings.init();
log.level = settings.loadOr(opts.LOG_LEVEL, log.WARN);
settings.loadOr(opts.WEBRTC_INIT_OFFER, true);
settings.loadOr(opts.INPUT_DPAD_MODE, true);
settings.loadOr(opts.TOUCH_MODE, 'auto');
settings.loadOr(opts.TOUCH_OPACITY, 0.55);
settings.loadOr(opts.TOUCH_SCALE, 1);
settings.loadOr(opts.VIDEO_PIXEL, false);
settings.loadOr(opts.FORCE_FULLSCREEN, false);
settings.loadOr(opts.SHOW_STATS, false);
settings.loadOr(opts.LIB_FAVORITES, []);
settings.loadOr(opts.LIB_RECENT, []);

// Application state

const app = {
    me: null,
    moderators: false,
    pending: 0,
    route: 'biblioteca',
    session: {wid: '', busy: false, games: []},
    lobby: {rooms: [], workers: {total: 0, free: 0}, queue: {size: 0}},
    intent: null, // {kind: 'play', game} | {kind: 'join', room, spectator, slot} | {kind: 'match', match} | {kind: 'replay', replay}
    room: {id: '', game: '', slot: -1, host: false, match: 0, replay: 0},
    inGame: false,
    kbm: false,
    kicked: false,
    retried: false,
    onInit: null, // one-shot action after the next session init
    reconnect: {attempts: 0, timer: null},
    statsTimer: null,
};

const can = (perm) => app.me?.perms?.includes(perm);
const gameOfRoom = (rid = '') => rid.split('___')[1] || '';
const systemOf = (title) => library.find(title)?.system || '';

// Top bar

const logoEl = document.getElementById('logo');
const navEl = document.getElementById('topnav');
const serverEl = document.getElementById('server-status');
const userBtn = document.getElementById('user-button');

logoEl.innerHTML = `<span class="logo-mark">${icon('gamepad')}</span><b>Belko</b><span>Arcade</span>`;

const renderNav = () => {
    const rooms = app.lobby.rooms?.length || 0;
    const links = [
        ['biblioteca', 'Biblioteca', 'library', ''],
        ['ao-vivo', 'Ao vivo', 'tv', rooms ? `<span class="badge live" style="background:rgba(239,68,68,.16);color:#fca5a5">${rooms}</span>` : ''],
        ['torneios', 'Torneios', 'crown', ''],
    ];
    if (can('admin.view')) {
        links.push(['admin', 'Admin', 'shield', app.pending ? `<span class="counter">${app.pending}</span>` : '']);
    }
    navEl.innerHTML = links.map(([r, t, i, extra]) =>
        `<a href="#/${r}" class="${app.route === r ? 'active' : ''}" ${app.route === r ? 'aria-current="page"' : ''}>${icon(i)}<span>${t}</span>${extra}</a>`).join('');
};

const renderServer = () => {
    const {total, free} = app.lobby.workers || {total: 0, free: 0};
    const linked = !!app.session.wid;
    let cls = 'danger', text = 'Sem servidores';
    if (total > 0) {
        if (free > 0 || linked) {
            cls = 'success';
            text = linked ? 'Servidor pronto' : `${free} ${free === 1 ? 'servidor livre' : 'servidores livres'}`;
        } else {
            cls = 'warning';
            text = 'Servidores ocupados';
        }
    }
    serverEl.innerHTML = `<span class="dot ${cls}"></span><span>${text}</span>`;
    serverEl.title = `${total} ${total === 1 ? 'servidor conectado' : 'servidores conectados'}`;
};

const renderUser = () => {
    userBtn.innerHTML = `${avatar(app.me.name, 'sm')}<span class="name">${esc(app.me.name)}</span>${icon('chevronDown')}`;
};

userBtn.addEventListener('click', () => {
    if (document.querySelector('.menu-pop.user-menu')) return;
    const pop = html(`<div class="menu-pop user-menu" role="menu">
        <div class="menu-head"><b>${esc(app.me.name)}</b><small>${esc(app.me.email)} · ${esc(roles[app.me.role] || '')}</small></div>
        <button class="menu-item" data-m="settings">${icon('settings')}Configurações</button>
        <button class="menu-item" data-m="help">${icon('keyboard')}Atalhos e controles</button>
        ${can('admin.view') ? `<a class="menu-item" href="#/admin" data-m="admin">${icon('shield')}Administração</a>` : ''}
        <div class="menu-sep"></div>
        <button class="menu-item danger" data-m="logout">${icon('logout')}Sair</button>
    </div>`);
    document.body.append(pop);
    const r = userBtn.getBoundingClientRect();
    pop.style.top = `${r.bottom + 8}px`;
    pop.style.right = `${Math.max(8, innerWidth - r.right)}px`;
    nav.pushScope(pop);
    const close = () => {
        nav.popScope(pop);
        pop.remove();
        document.removeEventListener('pointerdown', outside, true);
        document.removeEventListener('keydown', esc_, true);
    };
    const outside = (e) => !pop.contains(e.target) && !userBtn.contains(e.target) && close();
    const esc_ = (e) => e.key === 'Escape' && (e.stopPropagation(), close(), userBtn.focus());
    setTimeout(() => document.addEventListener('pointerdown', outside, true));
    document.addEventListener('keydown', esc_, true);
    pop.addEventListener('click', (e) => {
        const m = e.target.closest('[data-m]')?.dataset.m;
        if (!m) return;
        close();
        if (m === 'settings') settingsView.open();
        if (m === 'help') showHelp();
        if (m === 'logout') logout();
    });
    pop.querySelector('.menu-item')?.focus();
});

const logout = async () => {
    if (!await confirmDialog({title: 'Sair da conta?', text: 'Você precisará entrar novamente para jogar.', confirm: 'Sair', iconName: 'logout'})) return;
    try {
        await http.post('/api/auth/logout');
    } catch {
        // ignore
    }
    socket.close();
    location.href = '/entrar';
};

// Routing (#/biblioteca, #/ao-vivo, #/admin)

const views = {
    biblioteca: document.getElementById('view-library'),
    'ao-vivo': document.getElementById('view-live'),
    torneios: document.getElementById('view-tournaments'),
    admin: document.getElementById('view-admin'),
};

const route = () => {
    let [r, sub] = (location.hash.replace(/^#\/?/, '') || 'biblioteca').split('/');
    if (!views[r] || (r === 'admin' && !can('admin.view'))) r = 'biblioteca';
    app.route = r;
    Object.entries(views).forEach(([k, el]) => (el.hidden = k !== r));
    if (r === 'admin') admin.show();
    if (r === 'torneios') tournaments.show(sub ? +sub : null);
    renderNav();
    window.scrollTo({top: 0});
};
window.addEventListener('hashchange', route);

// Connection lifecycle

const teardown = () => {
    socket.close();
    webrtc.stop();
    stream.reset();
    stopGameInput();
    app.session.wid = '';
};

const connect = (intent = null) => {
    app.intent = intent;
    app.kicked = false;
    clearTimeout(app.reconnect.timer);
    const params = {};
    if (intent?.kind === 'join') params.room_id = intent.room;
    if (intent?.kind === 'match') params.match = intent.match;
    if (intent?.kind === 'replay') params.replay = intent.replay;
    socket.init(params, {
        onOpen: () => (app.reconnect.attempts = 0),
        onClose: onSocketClose,
    });
};

/** Drops the current server connection and connects again with a new intention. */
const restart = (intent = null) => {
    teardown();
    connect(intent);
};

const onSocketClose = async () => {
    webrtc.stop();
    stopGameInput();
    app.session.wid = '';
    renderServer();
    if (app.kicked) return;

    // the session may be gone (logout elsewhere, account blocked)
    try {
        const {user} = await http.get('/api/auth/me');
        if (user.status !== 'approved') {
            location.href = '/entrar';
            return;
        }
    } catch (e) {
        if (e.status === 401) {
            location.href = '/entrar';
            return;
        }
    }

    const attempt = ++app.reconnect.attempts;
    const delay = Math.min(10000, 800 * 2 ** (attempt - 1));
    if (app.inGame) {
        gameView.setLoading('connect', 'Conexão perdida. Reconectando…');
    } else if (attempt === 2) {
        toast.warning('Conexão com o servidor perdida. Tentando reconectar…');
    }
    // rejoin the same room when we were in one
    const intent = app.inGame && app.room.id ? rejoinIntent() : app.intent;
    app.reconnect.timer = setTimeout(() => connect(intent), delay);
};

const onConnectionReady = () => {
    if (app.reconnect.attempts > 1 && !app.inGame) toast.success('Conectado novamente.');
    app.reconnect.attempts = 0;
    if (app.intent) startIntent();
};

const startIntent = () => {
    const i = app.intent;
    if (!i) return;
    gameView.setLoading('start');
    if (i.kind === 'play') {
        api.game.start({game: i.game, player: 0});
    } else if (i.kind === 'join') {
        api.game.start({roomId: i.room, spectator: !!i.spectator, player: i.slot ?? -1});
    } else if (i.kind === 'match') {
        api.game.start({matchId: i.match});
    } else if (i.kind === 'replay') {
        api.game.start({replayId: i.replay});
    }
};

/** The intention that brings us back to the current room after a reconnection. */
const rejoinIntent = () => {
    if (app.room.match) return {kind: 'match', match: app.room.match};
    if (app.room.replay) return {kind: 'replay', replay: app.room.replay};
    return {kind: 'join', room: app.room.id, spectator: app.room.slot < 0, slot: app.room.slot};
};

const handleWebrtcStart = ({data, initiator}) => {
    webrtc.start({
        initiator,
        iceServers: data.ice,
        media: stream.video.el,
        onDataChannel: (ch) => {
            if (ch.label === 'data') {
                // ws and webrtc server messages are handled in one place
                ch.onmessage = (x) => onMessage(api.fromBytes(x.data));
            }
            return ch;
        },
        onConnect: onConnectionReady,
        onDisconnect: () => {
            stopGameInput();
            webrtc.stop();
            if (app.inGame && !app.kicked) {
                toast.warning('A transmissão caiu. Reconectando…');
                restart(rejoinIntent());
            }
        },
        signalling: {
            init: api.server.initWebrtcStream,
            sendIceCandidate: api.server.sendIceCandidate,
            sendSdp: api.server.sendSdp,
        },
    });
};

// Game view

const enterGame = ({game = '', roomId = '', match = 0, replay = 0} = {}) => {
    app.inGame = true;
    app.room = {id: roomId, game, slot: -1, host: false, match, replay};
    gameView.show({game, system: systemOf(game), roomId});
    gameView.setRole({slot: -1, host: false, canModerate: can('rooms.moderate')});
    gameView.toggleStats(settings.get(opts.SHOW_STATS));
    keyboard.capture = true;
    startStats();
};

const exitGame = () => {
    if (!app.inGame) return;
    app.inGame = false;
    app.room = {id: '', game: '', slot: -1, host: false, match: 0, replay: 0};
    app.kbm = false;
    input.kbm = false;
    stopGameInput();
    stopStats();
    keyboard.capture = false;
    if (document.fullscreenElement) document.exitFullscreen?.().catch(() => ({}));
    gameView.hide();
    live.setCurrentRoom('');
    if (location.search) history.replaceState(null, '', '/' + location.hash);
};

/** Leaves the room and goes back to the library. */
const leaveGame = () => {
    if (app.room.id) api.game.quit(app.room.id);
    exitGame();
    restart(null);
};

const play = (game) => {
    if (app.inGame) return;
    if (queueView.inQueue) {
        queueView.expect();
        api.queue.join(game);
        toast(`Fila atualizada: você vai jogar ${gameTitle(game).title}.`);
        return;
    }
    app.retried = false;
    if (app.session.wid) {
        enterGame({game});
        app.intent = {kind: 'play', game};
        if (webrtc.isConnected()) startIntent();
        return;
    }
    if ((app.lobby.workers?.free || 0) > 0) {
        enterGame({game});
        restart({kind: 'play', game});
        return;
    }
    joinQueue(game);
};

const joinQueue = (game) => {
    queueView.expect();
    api.queue.join(game);
    setTimeout(() => queueView.inQueue && app.lobby.rooms?.length && !app.inGame && queueView.openInfo(), 800);
};

const watch = (roomId) => {
    if (app.inGame) exitGame();
    enterGame({game: gameOfRoom(roomId), roomId});
    restart({kind: 'join', room: roomId, spectator: true});
};

/** Plays (participants) or watches (everyone else) a tournament match. */
const playMatch = (match, game = '') => {
    if (app.inGame) exitGame();
    enterGame({game, match});
    restart({kind: 'match', match});
};

const watchReplay = async (replay, game = '') => {
    try {
        await http.post(`/api/replays/${replay}/watch`);
    } catch (e) {
        toast.error(e.message);
        return;
    }
    if (app.inGame) exitGame();
    enterGame({game, replay});
    restart({kind: 'replay', replay});
};

const join = (roomId) => {
    if (app.inGame) exitGame();
    enterGame({game: gameOfRoom(roomId), roomId});
    restart({kind: 'join', room: roomId, spectator: false});
};

const onGameStarted = (payload) => {
    const roomId = payload.roomId;
    const game = gameOfRoom(roomId) || app.intent?.game || app.room.game;
    const slot = payload.slot ?? -1;
    const {match = 0, replay = 0} = app.room;
    app.intent = null;
    app.retried = false;
    app.room = {id: roomId, game, slot, host: !!payload.host, match, replay};
    if (!app.inGame) enterGame({game, roomId});
    gameView.setInfo({game, system: systemOf(game), roomId});
    gameView.setRole({slot, host: !!payload.host, canModerate: can('rooms.moderate')});
    gameView.setLoading('stream');
    live.setCurrentRoom(roomId);
    if (!match && !replay) history.replaceState(null, '', `/?id=${encodeURIComponent(roomId)}`);
    if (slot >= 0 && !match) rememberGame(game);

    if (payload.av) pub(APP_VIDEO_CHANGED, payload.av);
    if (payload.kb_mouse) pub(KB_MOUSE_FLAG);

    stream.toggle(true);
    stream.video.el.addEventListener('playing', () => gameView.setPlaying(true), {once: true});
    stream.play();
    // the video may already be playing (same worker, new room)
    if (!stream.video.el.paused) gameView.setPlaying(true);

    updateGameInput();
    if (settings.get(opts.FORCE_FULLSCREEN) && !isTouchDevice()) fullscreen(true);
    if (replay) toast('Replay: a partida é reproduzida exatamente como foi jogada.', 'info', 5000);
    else if (match && slot >= 0) toast(`Partida de torneio — você é o Jogador ${slot + 1}. Boa sorte!`, 'success', 6000);
    else if (match) toast('Você está assistindo a uma partida de torneio.', 'info', 5000);
    else if (slot < 0) toast('Você está assistindo. Ocupe uma vaga livre no painel de jogadores para jogar.', 'info', 5000);
};

const onNoFreeSlots = () => {
    const intent = app.intent;
    if (intent?.kind === 'match' || intent?.kind === 'replay' || app.room.match || app.room.replay) {
        exitGame();
        toast.warning('Esta partida não está disponível agora.');
        restart(null);
        return;
    }
    if (intent?.kind === 'join' || (!intent && app.inGame && app.room.id)) {
        exitGame();
        toast.warning('Essa sala não está mais disponível.');
        restart(null);
        return;
    }
    const game = intent?.game || app.room.game;
    if (!game) {
        toast.warning('Nenhum servidor livre no momento.');
        return;
    }
    // a free server may exist on another machine: try once with a fresh connection
    if (!app.retried && (app.lobby.workers?.free || 0) > 0) {
        app.retried = true;
        restart({kind: 'play', game});
        return;
    }
    app.retried = false;
    exitGame();
    app.onInit = () => joinQueue(game);
    restart(null);
};

// Input

const PAD_REPEAT_DELAY = 380, PAD_REPEAT_EVERY = 110;
let navRepeat = null;

const canPlay = () => app.inGame && app.room.slot >= 0 && !gameView.menuOpen && !modal.isOpen() && !settingsView.isOpen;

const releaseGameKeys = () => {
    JOYPAD_KEYS.forEach((k) => input.retropad.setKeyState(k, false));
    [0, 1, 2, 3].forEach((a) => input.retropad.setAxisChanged(a, 0));
};

const stopGameInput = () => {
    releaseGameKeys();
    input.retropad.toggle(false);
    touch.toggle(false);
    touch.show(false);
};

/** Enables the controls only for players (spectators can't send input). */
const updateGameInput = () => {
    const playing = canPlay();
    if (!playing) releaseGameKeys();
    // let the last (released) state go out before stopping the loop
    playing ? input.retropad.toggle(true) : setTimeout(() => !canPlay() && input.retropad.toggle(false), 50);
    touch.toggle(playing);
    const mode = settings.get(opts.TOUCH_MODE);
    touch.show(app.inGame && app.room.slot >= 0 && (mode === 'on' || (mode === 'auto' && isTouchDevice())));
};

const dirOf = {[KEY.UP]: 'up', [KEY.DOWN]: 'down', [KEY.LEFT]: 'left', [KEY.RIGHT]: 'right'};

const uiBack = () => {
    if (modal.closeTop()) return;
    if (document.querySelector('.menu-pop')) {
        document.dispatchEvent(new PointerEvent('pointerdown', {bubbles: true}));
        return;
    }
    if (app.inGame && gameView.menuOpen) gameView.setMenu(false);
    else if (app.inGame && gameView.panelOpen) gameView.setPanel(false);
    else if (app.inGame) gameView.setMenu(true);
};

const onUiKey = (data, down) => {
    const {key, pad, touch: fromTouch} = data;
    if (fromTouch) return;
    if (!pad) {
        // keyboard: native focus handling plus arrow navigation and Esc/F1
        if (!down) return;
        if (key === KEY.MENU) {
            uiBack();
            return;
        }
        if (dirOf[key] && !isTyping() && !document.activeElement?.matches?.('input[type=range], select')) {
            if (dirOf[key] && data.code?.startsWith('Arrow')) nav.move(dirOf[key], false);
        }
        return;
    }
    // gamepad: spatial navigation
    clearTimeout(navRepeat);
    if (!down) return;
    if (dirOf[key]) {
        nav.move(dirOf[key]);
        const repeat = (wait) => (navRepeat = setTimeout(() => {
            nav.move(dirOf[key]);
            repeat(PAD_REPEAT_EVERY);
        }, wait));
        repeat(PAD_REPEAT_DELAY);
        return;
    }
    if (key === KEY.B || key === KEY.START) nav.activate(); // bottom face button confirms
    else if (key === KEY.A) uiBack(); // right face button goes back
    else if (key === KEY.MENU) uiBack();
};

const onGameShortcut = (key) => {
    switch (key) {
        case KEY.MENU:
            gameView.setMenu(true);
            return true;
        case KEY.SAVE:
            saveGame();
            return true;
        case KEY.LOAD:
            loadGame();
            return true;
        case KEY.FULL:
            fullscreen();
            return true;
        case KEY.STATS: {
            const on = gameView.toggleStats();
            settings.set(opts.SHOW_STATS, on);
            return true;
        }
        case KEY.CHAT:
            releaseGameKeys();
            gameView.focusChat();
            return true;
    }
    return false;
};

const onKey = (data, down) => {
    if (settingsView.capturing) return;
    unmute();
    // in-game controls
    if (app.inGame && !gameView.menuOpen && !modal.isOpen() && !document.querySelector('.menu-pop')) {
        if (down && onGameShortcut(data.key)) return;
        if (app.room.slot >= 0) {
            if (JOYPAD_KEYS.includes(data.key)) input.retropad.setKeyState(data.key, down);
            return;
        }
        // spectators navigate the interface with the gamepad
    }
    onUiKey(data, down);
};

sub(KEY_PRESSED, (d) => onKey(d, true));
sub(KEY_RELEASED, (d) => onKey(d, false));
sub(AXIS_CHANGED, (d) => {
    unmute();
    if (canPlay()) input.retropad.setAxisChanged(d.id, d.value);
});
sub(CONTROLLER_UPDATED, (data) => canPlay() && webrtc.send('data', data));

// keyboard + mouse cores (DOSBox): input goes through dedicated channels in fullscreen
sub(KEYBOARD_KEY_DOWN, (e) => canPlay() && app.kbm && api.game.input.keyboard.press(true, e));
sub(KEYBOARD_KEY_UP, (e) => canPlay() && app.kbm && api.game.input.keyboard.press(false, e));
sub(MOUSE_MOVED, (e) => canPlay() && app.kbm && api.game.input.mouse.move(e.dx, e.dy));
sub(MOUSE_PRESSED, (e) => canPlay() && app.kbm && api.game.input.mouse.press(e.b, e.p));
sub(KB_MOUSE_FLAG, () => {
    app.kbm = true;
    input.kbm = true;
    toast('Este jogo usa teclado e mouse: entre em tela cheia (F) para jogar.', 'info', 6000);
});

const trackPointer = pointer.track(gameView.stage, () => ({...stream.video.size, s: true}));

const lockControls = async (locked) => {
    if (!app.kbm || isTouchDevice()) return;
    if (locked && canPlay()) {
        try {
            await pointer.lock(gameView.stage);
        } catch {
            // the browser may refuse without a gesture
        }
    }
    trackPointer(locked);
    await keyboard.lock(locked);
    locked ? input.retropad.toggle(false) : updateGameInput();
};

const fullscreen = (force) => {
    const on = force ?? !document.fullscreenElement;
    if (on) gameView.el.requestFullscreen?.().catch(() => toast.warning('Não foi possível entrar em tela cheia.'));
    else document.exitFullscreen?.().catch(() => ({}));
};

document.addEventListener('fullscreenchange', () => {
    const fs = !!document.fullscreenElement;
    gameView.setFullscreenIcon(fs);
    stream.onFullscreen(fs);
    setTimeout(() => lockControls(fs), 200);
});

// the first interaction unmutes the game sound (autoplay policies)
let interacted = false;
const unmute = () => {
    if (interacted) return;
    interacted = true;
    stream.audio.mute(false);
};
['pointerdown', 'keydown'].forEach((t) => document.addEventListener(t, unmute, {once: true, capture: true}));

// Room actions

const saveGame = () => {
    if (!app.room.id) return;
    api.game.save();
};
const loadGame = () => {
    if (!app.room.id) return;
    api.game.load();
};

gameView.mount({
    onLeave: leaveGame,
    onSave: saveGame,
    onLoad: loadGame,
    onReset: async () => {
        if (await confirmDialog({title: 'Reiniciar o jogo?', text: 'O jogo volta ao início. O progresso não salvo será perdido.', confirm: 'Reiniciar', danger: true, iconName: 'rotate'}))
            api.game.reset(app.room.id);
    },
    onSlot: (slot) => api.game.setPlayerIndex(slot),
    onAssign: (user, slot) => api.room.assign(user, slot),
    onKick: async (user, name) => {
        if (await confirmDialog({title: `Remover ${name}?`, text: 'A pessoa sai da sala e não poderá voltar para esta partida.', confirm: 'Remover', danger: true, iconName: 'ban'}))
            api.room.kick(user);
    },
    onChat: (text) => api.room.chat(text),
    onInvite: async (link) => {
        if (!link) return;
        (await copyText(link)) ? toast.success('Link de convite copiado!') : toast.error('Não foi possível copiar o link.');
    },
    onFullscreen: () => fullscreen(),
    onSettings: () => settingsView.open(),
    onHelp: () => showHelp(),
    onMenu: () => updateGameInput(),
    onLayout: () => stream.recalculateSize(),
});

// Network statistics

const bitrate = (() => {
    let prevBytes = 0, prevTs = 0;
    return (ts, bytes) => {
        const v = prevTs ? Math.round((8 * (bytes - prevBytes)) / (ts - prevTs)) : 0;
        prevBytes = bytes;
        prevTs = ts;
        return Math.max(0, v);
    };
})();

const startStats = () => {
    stopStats();
    app.statsTimer = setInterval(async () => {
        const report = await webrtc.stats();
        if (!report) return;
        const out = {};
        report.forEach((r) => {
            if ((r.type === 'candidate-pair' && (r.selected || r.nominated) && r.state === 'succeeded') && r.currentRoundTripTime !== undefined) {
                out.rtt = r.currentRoundTripTime * 1000;
            }
            if (r.type === 'inbound-rtp' && r.kind === 'video') {
                out.kbps = bitrate(r.timestamp, r.bytesReceived);
                out.fps = r.framesPerSecond;
                if (r.frameWidth) out.res = `${r.frameWidth}×${r.frameHeight}`;
                const lost = r.packetsLost || 0, recv = r.packetsReceived || 0;
                if (recv + lost > 0) out.loss = (lost / (recv + lost)) * 100;
                const codec = r.codecId && report.get(r.codecId);
                if (codec?.mimeType) out.codec = codec.mimeType.replace('video/', '').toUpperCase();
            }
        });
        gameView.setNet(out);
    }, 1000);
};

const stopStats = () => clearInterval(app.statsTimer);

// Latency check (the coordinator may ask to ping the servers)

const checkLatencies = async (data) => {
    const addresses = [...new Set(data || [])];
    const results = await Promise.all(addresses.map((address) => {
        const start = Date.now();
        return ajax.fetch(`${address}?_=${start}`, {method: 'GET', redirect: 'follow'}, 1111)
            .then(() => ({[address]: Date.now() - start}))
            .catch(() => ({[address]: 9999}));
    }));
    return Object.assign({}, ...results);
};

// Server messages

const onMessage = (m) => {
    const {id, t, p: payload} = m;
    log.debug(`[msg] ${api.endpointName[t] || t}`);
    const E = api.endpoint;
    switch (t) {
        case E.INIT: {
            app.session = {wid: payload.wid || '', busy: !!payload.busy, games: payload.games || []};
            if (payload.you) {
                app.me = {...app.me, ...payload.you};
                renderUser();
            }
            library.setGames(app.session.games);
            library.setLobby(app.lobby, !!app.session.wid);
            renderServer();
            if (app.onInit) {
                const fn = app.onInit;
                app.onInit = null;
                fn();
            }
            if (app.session.wid) {
                if (app.inGame) gameView.setLoading('connect');
                handleWebrtcStart({data: payload, initiator: !settings.get(opts.WEBRTC_INIT_OFFER)});
            } else if (app.intent) {
                const intent = app.intent;
                app.intent = null;
                exitGame();
                if (intent.kind === 'play') joinQueue(intent.game);
                else if (intent.kind === 'match') toast.warning('A partida ainda não tem um servidor. Aguarde ser chamado.');
                else toast.warning('Essa sala não está mais disponível.');
            }
            break;
        }
        case E.WEBRTC_SIGNAL:
            if (Object.hasOwn(payload, 'ice')) {
                webrtc.candidate(payload.ice ? api.fromJson(payload.ice) : '');
            } else if (Object.hasOwn(payload, 'sdp')) {
                webrtc.answer(api.fromJson(payload.sdp));
            }
            break;
        case E.GAME_START:
            onGameStarted(payload);
            break;
        case E.GAME_SAVE:
            payload === 'ok' ? toast.success('Progresso salvo.') : toast.error('Não foi possível salvar o progresso.');
            break;
        case E.GAME_LOAD:
            payload === 'ok' ? toast.success('Progresso carregado.') : toast.error('Não há progresso salvo para carregar.');
            break;
        case E.GAME_SET_PLAYER_INDEX: {
            const slot = +payload;
            app.room.slot = slot;
            gameView.setRole({slot});
            updateGameInput();
            slot >= 0 ? toast.success(`Agora você é o Jogador ${slot + 1}.`) : toast('Você agora está assistindo.');
            break;
        }
        case E.GAME_ERROR_NO_FREE_SLOTS:
            onNoFreeSlots();
            break;
        case E.APP_VIDEO_CHANGE:
            pub(APP_VIDEO_CHANGED, {...payload});
            break;
        case E.LATENCY_CHECK:
            checkLatencies(payload).then((l) => api.server.latencyCheck(id, l));
            break;
        case E.GAME_RECORDING:
            break;
        case E.LOBBY_STATE:
            app.lobby = payload;
            live.setLobby(payload);
            library.setLobby(payload, !!app.session.wid);
            renderServer();
            renderNav();
            break;
        case E.QUEUE_STATE:
            queueView.set(payload);
            library.setQueue(payload);
            break;
        case E.ROOM_STATE:
            if (payload.room_id !== app.room.id) break;
            gameView.setMembers(payload.members || []);
            {
                const me = (payload.members || []).find((x) => x.you);
                if (me) {
                    const changed = me.slot !== app.room.slot;
                    app.room.slot = me.slot;
                    app.room.host = me.host;
                    if (changed) updateGameInput();
                }
            }
            break;
        case E.CHAT_MESSAGE:
            gameView.addChat(payload);
            break;
        case E.KICKED:
            app.kicked = true;
            exitGame();
            teardown();
            infoDialog({
                title: 'Você saiu da sala', text: payload.reason || 'Você foi desconectado.', iconName: 'door', kind: 'warning',
                onClose: () => restart(null),
            });
            break;
        case E.NOTICE:
            onNotice(payload);
            break;
    }
};

const onNotice = async (n) => {
    switch (n.kind) {
        case 'pending_users': {
            const prev = app.pending;
            app.pending = n.count || 0;
            renderNav();
            if (app.pending > prev) {
                toast(`${app.pending === 1 ? 'Há 1 cadastro' : `Há ${app.pending} cadastros`} aguardando aprovação.`, 'info', 6000);
            }
            admin.pendingChanged();
            break;
        }
        case 'account_updated':
            try {
                const me = await http.get('/api/auth/me');
                app.me = me.user;
                app.moderators = me.moderators;
                admin.configure({user: app.me, moderators: app.moderators});
                renderUser();
                renderNav();
                gameView.setRole({canModerate: can('rooms.moderate')});
            } catch {
                // ignore
            }
            break;
        case 'room_gone':
            exitGame();
            toast.warning(n.text || 'Essa partida já terminou.');
            restart(null);
            break;
        case 'error':
            toast.error(n.text);
            break;
        case 'tournaments':
            tournaments.refresh();
            break;
        case 'tournament_over':
            toast.success(n.text ? `O torneio “${n.text}” terminou!` : 'Um torneio terminou!', 6000);
            tournaments.refresh();
            break;
        case 'match_rules':
            if (n.text) toast(n.text, 'info', 9000);
            break;
        case 'match_over':
            toast.success(n.text, 8000);
            tournaments.refresh();
            break;
        case 'match_closed':
            // the match room closes: back to the bracket, connected for the next call
            if (app.inGame && app.room.match === n.match) {
                exitGame();
                restart(null);
                location.hash = '#/torneios';
            }
            break;
        case 'match_ready':
            tournaments.refresh();
            if (app.room.match === n.match) break;
            joystick.rumble(400, 1);
            {
                const m = modal({
                    dismissible: false,
                    body: `<div class="modal-hero"><div class="hero-icon success">${icon('crown', 'xl')}</div>
                        <h2>Sua partida está pronta!</h2><p class="muted">${esc(n.text)}</p>
                        <p class="subtle">Entre em até 3 minutos ou a vitória vai para o adversário.</p></div>`,
                    footer: `<button class="btn ghost" data-later>Ver chaveamento</button><button class="btn primary lg" data-go autofocus>${icon('play')}Entrar agora</button>`,
                });
                m.el.querySelector('[data-go]').addEventListener('click', () => {
                    m.close();
                    playMatch(n.match);
                });
                m.el.querySelector('[data-later]').addEventListener('click', () => {
                    m.close();
                    location.hash = '#/torneios';
                });
            }
            break;
        default:
            if (n.text) toast(n.text);
    }
};

sub(MESSAGE, onMessage);

// Gamepad feedback

sub(GAMEPAD_CONNECTED, (d) => {
    if (d.switched) return;
    const names = {xbox: 'Controle Xbox', ps: 'Controle PlayStation', nintendo: 'Controle Nintendo'};
    toast.success(`${names[d.type] || 'Controle'} conectado.`);
});
sub(GAMEPAD_DISCONNECTED, () => {
    toast.warning('Controle desconectado.');
    if (app.inGame && app.room.slot >= 0) gameView.setMenu(true);
});

// Settings reactions

const applySettings = () => {
    pub(DPAD_TOGGLE, {checked: settings.get(opts.INPUT_DPAD_MODE) !== false});
    touch.style({opacity: settings.get(opts.TOUCH_OPACITY), scale: settings.get(opts.TOUCH_SCALE)});
    gameView.stage.classList.toggle('pixel', !!settings.get(opts.VIDEO_PIXEL));
    log.level = settings.get(opts.LOG_LEVEL);
    if (app.inGame) updateGameInput();
    library.render();
};
sub(SETTINGS_CHANGED, applySettings);

// Help: Esc/back also closes modals when nothing else handles it
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && modal.isOpen() && !settingsView.capturing && !keyboard.map.Escape) modal.closeTop();
});

// Boot

const boot = async () => {
    let me;
    try {
        me = await http.get('/api/auth/me');
    } catch {
        location.href = '/entrar';
        return;
    }
    if (me.user.status !== 'approved') {
        location.href = '/entrar';
        return;
    }
    app.me = me.user;
    app.moderators = me.moderators;
    app.pending = me.pending || 0;

    renderUser();
    renderServer();

    library.mount(views.biblioteca, {onPlay: play});
    live.mount(views['ao-vivo'], {onWatch: watch, onJoin: join});
    tournaments.configure({user: app.me});
    tournaments.mount(views.torneios, {onPlayMatch: playMatch, onWatchMatch: playMatch, onWatchReplay: watchReplay});
    admin.configure({user: app.me, moderators: app.moderators});
    admin.mount(views.admin, {
        onPending: (n) => {
            app.pending = n;
            renderNav();
        },
    });
    queueView.mount(document.getElementById('queue-pill'), {
        onAccept: (game) => {
            if (app.inGame) exitGame();
            enterGame({game});
            restart({kind: 'play', game});
        },
        onLeave: () => api.queue.leave(),
        onWatch: watch,
        rooms: () => live.rooms,
        rumble: () => joystick.rumble(300, 0.8),
    });
    settingsView.configure({
        me: app.me,
        onLogout: logout,
        onNameChanged: (name) => {
            app.me.name = name;
            renderUser();
        },
    });

    api.transport = {
        send: socket.send,
        keyboard: (data) => webrtc.send('keyboard', data),
        mouse: (data) => webrtc.send('mouse', data),
    };

    input.init();
    stream.init();
    touch.init(gameView.touchpad());
    applySettings();
    route();

    settings.attachRemote({
        load: () => http.get('/api/account/preferences'),
        save: (data) => http.put('/api/account/preferences', data),
    });

    // invite links: /?id=<room>
    const roomId = new URLSearchParams(location.search).get('id');
    if (roomId) {
        enterGame({game: gameOfRoom(roomId), roomId});
        connect({kind: 'join', room: roomId, spectator: false});
    } else {
        connect(null);
    }
};

boot();
