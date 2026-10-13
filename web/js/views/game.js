import {icon} from '../ui/icons.js?v=4';
import {avatar, esc, html, on} from '../ui/dom.js?v=4';
import {nav} from '../ui/nav.js?v=4';
import {gameTitle, systemName} from '../i18n.js?v=4';

/**
 * Game view: stage, HUD, players/chat panel and the in-game menu.
 * The <video> element and play button are owned by stream.js.
 */

const el = document.getElementById('view-game');
const stage = document.getElementById('stage');

const MAX_SLOTS = 4;
const HUD_HIDE_MS = 3000;

let h = {};
let state = {
    game: '',
    system: '',
    roomId: '',
    slot: -1,
    host: false,
    canModerate: false,
    members: [],
    panel: false,
    tab: 'players',
    unread: 0,
    menu: false,
    playing: false,
};
let hudTimer = null;

const steps = [
    ['connect', 'Conectando ao servidor'],
    ['start', 'Preparando o emulador'],
    ['stream', 'Recebendo a transmissão'],
];

// Markup

const build = () => {
    stage.insertAdjacentHTML('beforeend', `
        <div class="loader" data-loader>
            <div class="loader-box">
                <div class="loader-title" data-loader-title>Carregando…</div>
                <div class="steps">${steps.map(([k, t]) => `<div class="step" data-step="${k}"><span class="mark"></span>${t}</div>`).join('')}</div>
                <button class="btn ghost sm" data-action="leave">${icon('arrowLeft')}Cancelar</button>
            </div>
        </div>
        <div class="hud" data-hud>
            <button class="btn icon-only" data-action="leave" title="Sair da sala" aria-label="Sair da sala">${icon('arrowLeft')}</button>
            <div class="title"><b data-title></b><small data-subtitle></small></div>
            <div class="grow"></div>
            <span class="role-badge" data-role></span>
            <span class="net hide-sm" data-net title="Qualidade da conexão"><span class="dot"></span><span>—</span></span>
            <button class="btn hide-sm" data-action="invite">${icon('link')}Convidar</button>
            <button class="btn icon-only" data-action="panel" title="Jogadores e chat" aria-label="Jogadores e chat">${icon('users')}</button>
            <button class="btn icon-only" data-action="fullscreen" title="Tela cheia (F)" aria-label="Tela cheia">${icon('maximize')}</button>
            <button class="btn icon-only" data-action="menu" title="Menu (Esc)" aria-label="Menu do jogo">${icon('menu')}</button>
        </div>
        <div class="stats-overlay" data-stats hidden></div>
        <div class="touchpad" data-touchpad hidden></div>
        <div class="game-menu" data-menu hidden></div>`);

    el.insertAdjacentHTML('beforeend', `
        <aside class="side-panel" data-panel aria-label="Jogadores e chat">
            <div class="side-head">
                <div class="tabs" role="tablist">
                    <button class="tab active" data-tab="players">${icon('users')}Jogadores</button>
                    <button class="tab" data-tab="chat">${icon('message')}Chat <span class="counter" data-unread hidden></span></button>
                </div>
                <button class="btn ghost icon-only sm" data-action="panel" aria-label="Fechar painel">${icon('x')}</button>
            </div>
            <div class="side-body" data-pane="players">
                <div class="slots" data-slots></div>
                <div class="spectators" data-spectators></div>
                <div class="invite-box">
                    <b>Convide amigos</b>
                    <p class="subtle" style="font-size:12px">Quem tiver o link entra direto na sua sala.</p>
                    <div class="row"><input class="input mono" readonly data-invite aria-label="Link da sala">
                    <button class="btn primary icon-only" data-action="invite" aria-label="Copiar link">${icon('copy')}</button></div>
                </div>
            </div>
            <div class="side-body chat" data-pane="chat" hidden>
                <div class="chat-log" data-chat-log aria-live="polite"></div>
                <form class="chat-form" data-chat-form>
                    <input class="input" data-chat-input maxlength="300" placeholder="Escreva uma mensagem…" aria-label="Mensagem" autocomplete="off">
                    <button class="btn primary icon-only" type="submit" aria-label="Enviar">${icon('send')}</button>
                </form>
            </div>
        </aside>`);
};

const $ = (s) => el.querySelector(s);

// Loader

const setLoading = (step, title) => {
    const loader = $('[data-loader]');
    if (!step) {
        loader.hidden = true;
        return;
    }
    loader.hidden = false;
    if (title) $('[data-loader-title]').textContent = title;
    const idx = steps.findIndex(([k]) => k === step);
    el.querySelectorAll('[data-step]').forEach((s, i) => {
        s.classList.toggle('done', i < idx);
        s.classList.toggle('active', i === idx);
    });
};

// HUD

const showHud = () => {
    el.classList.remove('hud-hidden');
    stage.classList.remove('no-pointer');
    clearTimeout(hudTimer);
    hudTimer = setTimeout(() => {
        if (state.menu || !state.playing || $('[data-hud]').matches(':focus-within')) return;
        el.classList.add('hud-hidden');
        stage.classList.add('no-pointer');
    }, HUD_HIDE_MS);
};

const renderHud = () => {
    const {title} = gameTitle(state.game);
    $('[data-title]').textContent = title || 'Arcade';
    $('[data-subtitle]').textContent = state.system ? systemName(state.system) : '';
    const role = $('[data-role]');
    if (state.slot < 0) {
        role.className = 'role-badge watching';
        role.innerHTML = `${icon('eye')}ASSISTINDO`;
    } else {
        role.className = 'role-badge';
        role.innerHTML = `${state.host ? icon('crown') : icon('gamepad')}Jogador ${state.slot + 1}`;
    }
    $('[data-invite]').value = inviteLink();
};

const inviteLink = () => state.roomId ? `${location.origin}/?id=${encodeURIComponent(state.roomId)}` : '';

const setNet = ({rtt, fps, kbps, codec, res, loss}) => {
    const net = $('[data-net]');
    const level = rtt === undefined ? '' : rtt < 60 ? 'success' : rtt < 130 ? 'warning' : 'danger';
    net.innerHTML = `<span class="dot ${level}"></span><span>${rtt === undefined ? '—' : `${Math.round(rtt)} ms`}</span>`;
    const stats = $('[data-stats]');
    if (!stats.hidden) {
        stats.innerHTML = [
            ['Latência', rtt === undefined ? '—' : `${Math.round(rtt)} ms`],
            ['Quadros', fps ? `${Math.round(fps)} fps` : '—'],
            ['Taxa', kbps ? `${kbps} kb/s` : '—'],
            ['Perda', loss !== undefined ? `${loss.toFixed(1)}%` : '—'],
            ['Codec', codec || '—'],
            ['Resolução', res || '—'],
        ].map(([k, v]) => `<span>${k}</span><span>${esc(v)}</span>`).join('');
    }
};

// Panel

const setPanel = (open, tab) => {
    state.panel = open === undefined ? !state.panel : open;
    if (tab) setTab(tab);
    el.classList.toggle('panel-closed', !state.panel);
    if (state.panel && state.tab === 'chat') clearUnread();
    setTimeout(() => h.onLayout?.(), 260);
};

const setTab = (tab) => {
    state.tab = tab;
    el.querySelectorAll('[data-tab]').forEach((t) => t.classList.toggle('active', t.dataset.tab === tab));
    el.querySelectorAll('[data-pane]').forEach((p) => (p.hidden = p.dataset.pane !== tab));
    if (tab === 'chat') clearUnread();
};

const clearUnread = () => {
    state.unread = 0;
    $('[data-unread]').hidden = true;
};

const memberActions = (m) => {
    if (m.you || !(state.host || state.canModerate)) return '';
    return `<button class="btn ghost icon-only sm" data-member="${esc(m.id)}" aria-label="Opções para ${esc(m.name)}">${icon('more')}</button>`;
};

const renderRoster = () => {
    const bySlot = new Map(state.members.filter((m) => m.slot >= 0).map((m) => [m.slot, m]));
    $('[data-slots]').innerHTML = Array.from({length: MAX_SLOTS}, (_, i) => {
        const m = bySlot.get(i);
        if (!m) {
            return `<div class="slot free"><span class="num">${i + 1}</span>
                <div class="who"><b>Vaga livre</b><small>Jogador ${i + 1}</small></div>
                <button class="btn outline sm" data-take="${i}">Ocupar</button></div>`;
        }
        return `<div class="slot ${m.you ? 'me' : ''}"><span class="num p${i + 1}">${i + 1}</span>
            ${avatar(m.name, 'sm', m.account)}
            <div class="who"><b>${esc(m.name)}${m.host ? `<span class="host-star" title="Anfitrião">${icon('star')}</span>` : ''}</b>
            <small>${m.you ? 'Você' : `Jogador ${i + 1}`}${m.host ? ' · anfitrião' : ''}</small></div>
            ${m.you ? `<button class="btn ghost sm" data-take="-1" title="Deixar a vaga e assistir">${icon('eye')}</button>` : memberActions(m)}
        </div>`;
    }).join('');

    const watchers = state.members.filter((m) => m.slot < 0);
    $('[data-spectators]').innerHTML = watchers.length ? `
        <div class="section-title" style="margin:8px 4px">${icon('eye')}Assistindo (${watchers.length})</div>
        ${watchers.map((m) => `<div class="spectator-row">${avatar(m.name, 'sm', m.account)}
            <span class="name">${esc(m.name)}${m.you ? ' <span class="subtle">(você)</span>' : ''}${m.host ? ` <span class="host-star">${icon('star')}</span>` : ''}</span>
            ${memberActions(m)}</div>`).join('')}` : '';
};

const openMemberMenu = (anchor, memberId) => {
    const m = state.members.find((x) => x.id === memberId);
    if (!m) return;
    const taken = new Set(state.members.filter((x) => x.slot >= 0).map((x) => x.slot));
    const moves = Array.from({length: MAX_SLOTS}, (_, i) => i)
        .filter((i) => !taken.has(i))
        .map((i) => `<button class="menu-item" data-assign="${i}">${icon('gamepad')}Mover para Jogador ${i + 1}</button>`);
    if (m.slot >= 0) moves.push(`<button class="menu-item" data-assign="-1">${icon('eye')}Mover para espectadores</button>`);
    const pop = html(`<div class="menu-pop" role="menu">
        <div class="menu-head"><b>${esc(m.name)}</b><small>${m.slot >= 0 ? `Jogador ${m.slot + 1}` : 'Assistindo'}</small></div>
        ${moves.join('')}
        <div class="menu-sep"></div>
        <button class="menu-item danger" data-kick>${icon('ban')}Remover da sala</button>
    </div>`);
    document.body.append(pop);
    const r = anchor.getBoundingClientRect();
    pop.style.top = `${Math.min(r.bottom + 6, innerHeight - pop.offsetHeight - 8)}px`;
    pop.style.left = `${Math.max(8, r.right - pop.offsetWidth)}px`;
    nav.pushScope(pop);
    const close = () => {
        nav.popScope(pop);
        pop.remove();
        document.removeEventListener('pointerdown', outside, true);
    };
    const outside = (e) => !pop.contains(e.target) && close();
    setTimeout(() => document.addEventListener('pointerdown', outside, true));
    pop.addEventListener('click', (e) => {
        const a = e.target.closest('[data-assign]');
        if (a) h.onAssign?.(memberId, +a.dataset.assign);
        if (e.target.closest('[data-kick]')) h.onKick?.(memberId, m.name);
        if (a || e.target.closest('[data-kick]')) close();
    });
    pop.querySelector('.menu-item')?.focus();
};

// Chat

const time = (ts) => new Date(ts).toLocaleTimeString('pt-BR', {hour: '2-digit', minute: '2-digit'});

const addChat = (msg) => {
    const log = $('[data-chat-log]');
    const node = msg.system
        ? html(`<div class="chat-msg system">${esc(msg.text)}</div>`)
        : html(`<div class="chat-msg">${avatar(msg.name, 'sm')}<div class="body">
            <div class="meta"><b>${esc(msg.name)}</b><small>${time(msg.ts)}</small></div>
            <div class="text">${esc(msg.text)}</div></div></div>`);
    const atBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 40;
    log.append(node);
    while (log.children.length > 200) log.firstElementChild.remove();
    if (atBottom) log.scrollTop = log.scrollHeight;
    if (!msg.system && !(state.panel && state.tab === 'chat')) {
        state.unread++;
        const u = $('[data-unread]');
        u.textContent = state.unread > 9 ? '9+' : state.unread;
        u.hidden = false;
    }
};

const focusChat = () => {
    setPanel(true, 'chat');
    setTimeout(() => $('[data-chat-input]').focus(), 50);
};

// In-game menu

const renderMenu = () => {
    const can = state.host || state.canModerate;
    const hostOnly = can ? '' : 'disabled title="Somente o anfitrião"';
    const watching = state.slot < 0;
    $('[data-menu]').innerHTML = `<div class="game-menu-box" role="dialog" aria-label="Menu do jogo">
        <h2>${esc(gameTitle(state.game).title || 'Menu')}</h2>
        <button class="btn primary" data-action="resume">${icon('play')}Continuar<span class="hint"><kbd>Esc</kbd></span></button>
        ${watching ? '' : `
        <button class="btn" data-action="save" ${hostOnly}>${icon('save')}Salvar progresso<span class="hint"><kbd>F2</kbd></span></button>
        <button class="btn" data-action="load" ${hostOnly}>${icon('upload')}Carregar progresso<span class="hint"><kbd>F4</kbd></span></button>
        <button class="btn" data-action="reset" ${hostOnly}>${icon('rotate')}Reiniciar jogo</button>`}
        ${can ? '' : '<small class="subtle" style="padding:0 4px">Salvar, carregar e reiniciar ficam com o anfitrião da sala.</small>'}
        <div class="sep"></div>
        <button class="btn" data-action="players">${icon('users')}Jogadores e chat</button>
        <button class="btn" data-action="invite">${icon('link')}Copiar convite</button>
        <button class="btn" data-action="fullscreen">${icon('maximize')}Tela cheia<span class="hint"><kbd>F</kbd></span></button>
        <button class="btn" data-action="settings">${icon('settings')}Configurações</button>
        <button class="btn" data-action="help">${icon('keyboard')}Atalhos e controles</button>
        <div class="sep"></div>
        <button class="btn danger" data-action="leave">${icon('door')}Sair da sala</button>
    </div>`;
};

const setMenu = (open) => {
    state.menu = open === undefined ? !state.menu : open;
    const m = $('[data-menu]');
    if (state.menu) {
        renderMenu();
        m.hidden = false;
        nav.pushScope(m);
        requestAnimationFrame(() => m.querySelector('.btn')?.focus());
        showHud();
    } else {
        m.hidden = true;
        nav.popScope(m);
        stage.focus?.();
    }
    h.onMenu?.(state.menu);
};

// Actions

const actions = {
    leave: () => h.onLeave?.(),
    invite: () => h.onInvite?.(inviteLink()),
    panel: () => setPanel(),
    players: () => {
        setMenu(false);
        setPanel(true, 'players');
    },
    fullscreen: () => h.onFullscreen?.(),
    menu: () => setMenu(),
    resume: () => setMenu(false),
    save: () => {
        setMenu(false);
        h.onSave?.();
    },
    load: () => {
        setMenu(false);
        h.onLoad?.();
    },
    reset: () => {
        setMenu(false);
        h.onReset?.();
    },
    settings: () => h.onSettings?.(),
    help: () => h.onHelp?.(),
};

export const gameView = {
    el,
    stage,
    mount(handlers) {
        h = handlers;
        build();
        on(el, 'click', '[data-action]', (e, b) => actions[b.dataset.action]?.());
        on(el, 'click', '[data-tab]', (e, b) => setTab(b.dataset.tab));
        on(el, 'click', '[data-take]', (e, b) => h.onSlot?.(+b.dataset.take));
        on(el, 'click', '[data-member]', (e, b) => openMemberMenu(b, b.dataset.member));
        $('[data-chat-form]').addEventListener('submit', (e) => {
            e.preventDefault();
            const input = $('[data-chat-input]');
            const text = input.value.trim();
            if (text) h.onChat?.(text);
            input.value = '';
        });
        $('[data-chat-input]').addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                e.stopPropagation();
                e.target.blur();
            }
        });
        stage.addEventListener('pointermove', showHud);
        stage.addEventListener('pointerdown', showHud);
        el.classList.add('panel-closed');
    },
    /** Opens the game view for a room. */
    show({game = '', system = '', roomId = ''} = {}) {
        Object.assign(state, {game, system, roomId, members: [], unread: 0, menu: false, playing: false});
        $('[data-chat-log]').innerHTML = '';
        clearUnread();
        el.hidden = false;
        document.body.style.overflow = 'hidden';
        setLoading('connect', 'Entrando na partida…');
        renderHud();
        renderRoster();
        $('[data-menu]').hidden = true;
        showHud();
    },
    hide() {
        el.hidden = true;
        document.body.style.overflow = '';
        state.menu = false;
        state.playing = false;
        $('[data-menu]').hidden = true;
        nav.popScope($('[data-menu]'));
        clearTimeout(hudTimer);
    },
    get visible() {
        return !el.hidden;
    },
    setLoading,
    setPlaying(v) {
        state.playing = v;
        if (v) setLoading(null);
        showHud();
    },
    setInfo({game, system, roomId}) {
        if (game !== undefined) state.game = game;
        if (system !== undefined) state.system = system;
        if (roomId !== undefined) state.roomId = roomId;
        renderHud();
    },
    setRole({slot, host, canModerate}) {
        if (slot !== undefined) state.slot = slot;
        if (host !== undefined) state.host = host;
        if (canModerate !== undefined) state.canModerate = canModerate;
        renderHud();
        renderRoster();
        if (state.menu) renderMenu();
    },
    setMembers(members = []) {
        state.members = members;
        const me = members.find((m) => m.you);
        if (me) {
            state.slot = me.slot;
            state.host = me.host;
        }
        renderHud();
        renderRoster();
        if (state.menu) renderMenu();
    },
    get role() {
        return {slot: state.slot, host: state.host};
    },
    addChat,
    focusChat,
    setNet,
    toggleStats(v) {
        const s = $('[data-stats]');
        s.hidden = v === undefined ? !s.hidden : !v;
        return !s.hidden;
    },
    setMenu,
    get menuOpen() {
        return state.menu;
    },
    setPanel,
    get panelOpen() {
        return state.panel;
    },
    touchpad: () => $('[data-touchpad]'),
    showHud,
    setFullscreenIcon(fs) {
        el.querySelectorAll('[data-action="fullscreen"]').forEach((b) => {
            const svg = b.querySelector('svg');
            if (svg) svg.outerHTML = icon(fs ? 'minimize' : 'maximize');
        });
    },
    inviteLink,
};
