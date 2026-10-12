import {opts, settings} from 'settings';
import {icon} from '../ui/icons.js?v=4';
import {ago, esc, on} from '../ui/dom.js?v=4';
import {cover} from '../ui/covers.js?v=4';
import {gameTitle, systemName, systemShort, tagName} from '../i18n.js?v=4';

/**
 * Game library: search, system filters, favorites, recently played.
 */

let root;
let handlers = {onPlay: () => ({}), onQueue: () => ({})};
let games = [];
let filter = 'all';
let query = '';
let lobby = {workers: {total: 0, free: 0}, queue: {size: 0}};
let queued = null; // current queue state
let linked = false; // has a server for this session

const favorites = () => new Set(settings.get(opts.LIB_FAVORITES) || []);
const recent = () => settings.get(opts.LIB_RECENT) || [];

const toggleFav = (title) => {
    const f = favorites();
    f.has(title) ? f.delete(title) : f.add(title);
    settings.set(opts.LIB_FAVORITES, [...f]);
};

/** Remembers a played game (max 8 entries). */
export const rememberGame = (title) => {
    const list = recent().filter((r) => r.title !== title);
    list.unshift({title, ts: Date.now()});
    settings.set(opts.LIB_RECENT, list.slice(0, 8));
};

const card = (g, extra = '') => {
    const {title, tags} = gameTitle(g.alias || g.title);
    const fav = favorites().has(g.title);
    return `<div class="game-card" role="button" tabindex="0" data-game="${esc(g.title)}" aria-label="Jogar ${esc(title)}">
        ${cover(g.title, g.system)}
        <button class="fav ${fav ? 'on' : ''}" data-fav="${esc(g.title)}" tabindex="-1" aria-label="${fav ? 'Remover dos favoritos' : 'Adicionar aos favoritos'}">${icon('star')}</button>
        <span class="play-fab">${icon('play')}</span>
        <div class="game-info">
            <span class="game-title" title="${esc(title)}">${esc(title)}</span>
            <span class="game-meta">
                <span class="badge">${esc(systemName(g.system))}</span>
                ${tags.slice(0, 2).map((t) => `<span class="badge accent">${esc(tagName(t))}</span>`).join('')}
                ${extra}
            </span>
        </div>
    </div>`;
};

const matches = (g) => {
    if (filter === 'fav' && !favorites().has(g.title)) return false;
    if (filter !== 'all' && filter !== 'fav' && g.system !== filter) return false;
    if (query) {
        const q = query.toLowerCase();
        return (g.alias || g.title).toLowerCase().includes(q) || systemName(g.system).toLowerCase().includes(q);
    }
    return true;
};

const renderBanner = () => {
    const el = root.querySelector('[data-banner]');
    const busy = !linked && lobby.workers.total > 0 && lobby.workers.free === 0;
    if (!busy || queued) {
        el.innerHTML = '';
        return;
    }
    el.innerHTML = `<div class="banner">
        ${icon('hourglass', 'lg')}
        <div class="grow"><b>Todos os servidores estão ocupados</b>
        <p>Escolha um jogo para entrar na fila. Enquanto espera, você pode assistir às partidas ao vivo.</p></div>
        <a class="btn outline" href="#/ao-vivo">${icon('tv')}Ver ao vivo</a>
    </div>`;
};

const renderChips = () => {
    const systems = [...new Set(games.map((g) => g.system))].sort();
    const count = (s) => games.filter((g) => g.system === s).length;
    const favCount = games.filter((g) => favorites().has(g.title)).length;
    root.querySelector('[data-chips]').innerHTML = [
        `<button class="chip ${filter === 'all' ? 'active' : ''}" data-filter="all">Todos <span class="count">${games.length}</span></button>`,
        favCount ? `<button class="chip ${filter === 'fav' ? 'active' : ''}" data-filter="fav">${icon('star')}Favoritos <span class="count">${favCount}</span></button>` : '',
        ...systems.map((s) => `<button class="chip ${filter === s ? 'active' : ''}" data-filter="${esc(s)}" title="${esc(systemName(s))}">${esc(systemShort(s))} <span class="count">${count(s)}</span></button>`),
    ].join('');
};

const renderRecent = () => {
    const el = root.querySelector('[data-recent]');
    const byTitle = new Map(games.map((g) => [g.title, g]));
    const items = recent().filter((r) => byTitle.has(r.title)).slice(0, 4);
    if (!items.length || query || filter !== 'all') {
        el.innerHTML = '';
        return;
    }
    el.innerHTML = `<div class="section-title">${icon('history')}Jogados recentemente</div>
        <div class="game-grid recent">${items.map((r) => card(byTitle.get(r.title), `<span class="badge cyan">${esc(ago(r.ts))}</span>`)).join('')}</div>`;
};

const renderGrid = () => {
    const el = root.querySelector('[data-grid]');
    const list = games.filter(matches);
    root.querySelector('[data-total]').textContent = games.length
        ? `${games.length} ${games.length === 1 ? 'jogo disponível' : 'jogos disponíveis'}`
        : '';
    if (!games.length) {
        el.innerHTML = `<div class="empty">${icon('server', 'xl')}<h3>Nenhum jogo disponível</h3>
            <p>Os servidores de jogos ainda não se conectaram. Tente novamente em instantes.</p></div>`;
        return;
    }
    if (!list.length) {
        el.innerHTML = `<div class="empty">${icon('search', 'xl')}<h3>Nada encontrado</h3>
            <p>Nenhum jogo corresponde a “${esc(query)}”. Tente outro nome ou limpe os filtros.</p>
            <button class="btn outline" data-clear>Limpar filtros</button></div>`;
        return;
    }
    el.innerHTML = `<div class="section-title">${icon('library')}${filter === 'fav' ? 'Favoritos' : filter === 'all' ? 'Todos os jogos' : esc(systemName(filter))}</div>
        <div class="game-grid">${list.map((g) => card(g)).join('')}</div>`;
};

const render = () => {
    if (!root) return;
    renderBanner();
    renderChips();
    renderRecent();
    renderGrid();
};

export const library = {
    mount(el, h) {
        root = el;
        handlers = {...handlers, ...h};
        root.innerHTML = `<div class="page">
            <div class="page-head">
                <div><h1>Biblioteca</h1><p data-total></p></div>
            </div>
            <div data-banner></div>
            <div class="toolbar">
                <div class="input-wrap search">${icon('search')}
                    <input class="input with-icon" type="search" placeholder="Buscar jogo… (atalho: /)" aria-label="Buscar jogo" data-search>
                </div>
                <div class="chips" data-chips></div>
            </div>
            <div data-recent></div>
            <div class="section" data-grid></div>
        </div>`;

        const search = root.querySelector('[data-search]');
        search.addEventListener('input', () => {
            query = search.value.trim();
            renderRecent();
            renderGrid();
        });
        document.addEventListener('keydown', (e) => {
            if (e.key === '/' && !root.hidden && document.activeElement !== search && !e.target.closest?.('input,textarea')) {
                e.preventDefault();
                search.focus();
            }
        });

        on(root, 'click', '[data-filter]', (e, b) => {
            filter = b.dataset.filter;
            render();
        });
        on(root, 'click', '[data-clear]', () => {
            filter = 'all';
            query = '';
            search.value = '';
            render();
        });
        on(root, 'click', '[data-fav]', (e, b) => {
            e.stopPropagation();
            toggleFav(b.dataset.fav);
            render();
        });
        on(root, 'click', '[data-game]', (e, c) => handlers.onPlay(c.dataset.game));
        on(root, 'keydown', '[data-game]', (e, c) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handlers.onPlay(c.dataset.game);
            }
        });
        render();
    },
    setGames(list = []) {
        games = [...list].sort((a, b) => (a.alias || a.title).localeCompare(b.alias || b.title, 'pt-BR'));
        render();
    },
    setLobby(l, isLinked) {
        lobby = l || lobby;
        linked = isLinked;
        renderBanner();
    },
    setQueue(q) {
        queued = q && q.status !== 'none' && q.status !== 'expired' ? q : null;
        renderBanner();
    },
    find: (title) => games.find((g) => g.title === title),
    render,
};
