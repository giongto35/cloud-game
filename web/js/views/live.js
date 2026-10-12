import {icon} from '../ui/icons.js?v=4';
import {ago, avatar, esc, on} from '../ui/dom.js?v=4';
import {cover} from '../ui/covers.js?v=4';
import {gameTitle, systemName} from '../i18n.js?v=4';

/**
 * Live rooms: watch or join games in progress.
 */

let root;
let handlers = {onWatch: () => ({}), onJoin: () => ({})};
let lobby = {rooms: [], workers: {total: 0, free: 0}, queue: {size: 0}};
let currentRoom = '';

const roomCard = (r) => {
    const {title} = gameTitle(r.game);
    const players = r.players || [];
    const mine = r.id === currentRoom;
    return `<article class="room-card">
        ${cover(r.game, r.system, '<span class="badge live">AO VIVO</span>')}
        <div class="room-body">
            <div>
                <h3 style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(title)}</h3>
                <small class="muted">${esc(systemName(r.system))}${r.host ? ` · anfitrião: ${esc(r.host)}` : ''}${r.since ? ` · começou ${esc(ago(r.since))}` : ''}</small>
            </div>
            <div class="room-players">
                <div class="avatar-stack">${players.map((p) => avatar(p.name, 'sm')).join('')}</div>
                <span>${players.length} ${players.length === 1 ? 'jogador' : 'jogadores'}</span>
                <span class="subtle">·</span>
                <span>${icon('eye')} ${r.spectators} ${r.spectators === 1 ? 'assistindo' : 'assistindo'}</span>
                ${r.slots_free > 0 ? `<span class="badge success">${r.slots_free} ${r.slots_free === 1 ? 'vaga' : 'vagas'}</span>` : '<span class="badge warning">Sala cheia</span>'}
            </div>
            <div class="room-actions">
                ${mine ? '<span class="badge accent">Você está nesta sala</span>' : `
                <button class="btn outline" data-watch="${esc(r.id)}">${icon('eye')}Assistir</button>
                <button class="btn primary" data-join="${esc(r.id)}" ${r.slots_free > 0 ? '' : 'disabled'}>${icon('gamepad')}Entrar</button>`}
            </div>
        </div>
    </article>`;
};

const render = () => {
    if (!root) return;
    const rooms = lobby.rooms || [];
    const w = lobby.workers || {total: 0, free: 0};
    root.querySelector('[data-stats]').innerHTML = `
        <div class="stat"><span>Salas ao vivo</span><b>${rooms.length}</b></div>
        <div class="stat"><span>Servidores livres</span><b>${w.free}<small class="subtle" style="font-size:16px"> / ${w.total}</small></b></div>
        <div class="stat"><span>Na fila</span><b>${lobby.queue?.size || 0}</b></div>`;
    root.querySelector('[data-rooms]').innerHTML = rooms.length
        ? `<div class="room-grid">${rooms.map(roomCard).join('')}</div>`
        : `<div class="empty">${icon('tv', 'xl')}<h3>Nenhuma partida acontecendo agora</h3>
            <p>Quando alguém começar a jogar, a partida aparece aqui para você assistir ou entrar.</p>
            <a class="btn primary" href="#/biblioteca">${icon('gamepad')}Começar uma partida</a></div>`;
};

export const live = {
    mount(el, h) {
        root = el;
        handlers = {...handlers, ...h};
        root.innerHTML = `<div class="page">
            <div class="page-head"><div><h1>Ao vivo</h1><p>Assista às partidas em andamento ou entre numa sala com vaga livre.</p></div></div>
            <div class="stat-row" data-stats></div>
            <div class="section" data-rooms></div>
        </div>`;
        on(root, 'click', '[data-watch]', (e, b) => handlers.onWatch(b.dataset.watch));
        on(root, 'click', '[data-join]', (e, b) => handlers.onJoin(b.dataset.join));
        render();
    },
    setLobby(l) {
        lobby = l || lobby;
        render();
    },
    setCurrentRoom(id) {
        currentRoom = id || '';
        render();
    },
    get rooms() {
        return lobby.rooms || [];
    },
};
