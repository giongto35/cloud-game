import {opts, settings} from 'settings';
import {icon} from '../ui/icons.js?v=4';
import {ago, esc, on} from '../ui/dom.js?v=4';
import {bumpCover, cover, coverUrl} from '../ui/covers.js?v=4';
import {gameTitle, systemName, tagName} from '../i18n.js?v=4';
import {http} from '../http.js?v=4';
import {confirmDialog, modal} from '../ui/modal.js?v=4';
import {toast} from '../ui/toast.js?v=4';

/**
 * Game catalog, laid out like a streaming service: a featured game, horizontal rows
 * (continue playing, my list, one per genre...) and a simple browse page for search and filters.
 * Catalog data (genre, year, developer, players) comes from /api/meta.
 */

let root;
let handlers = {onPlay: () => ({}), onQueue: () => ({})};
let games = [];
let metaByKey = new Map(); // "system|title" -> catalog entry
let categoryOrder = [];
let lobby = {workers: {total: 0, free: 0}, queue: {size: 0}};
let queued = null; // current queue state
let linked = false; // has a server for this session
let metaTimer = null;
let canEditCovers = false;

const NO_CATEGORY = 'Outros';
const ROW_LIMIT = 30;

// what the page shows: no filter = home (rows); any filter or search = browse (grid)
const state = {
    tab: 'home', // home | fav | multi
    category: '',
    system: '',
    sort: 'az', // az | za | new | old
    query: '',
};

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

const metaOf = (g) => metaByKey.get(`${(g.system || '').toLowerCase()}|${g.title}`) || null;
const categoryOf = (g) => metaOf(g)?.genre || NO_CATEGORY;
const nameOf = (g) => g.alias || g.title;
const cleanName = (g) => gameTitle(nameOf(g)).title;
const norm = (s = '') => s.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
const byName = (a, b) => cleanName(a).localeCompare(cleanName(b), 'pt-BR');
const isBrowse = () => state.tab !== 'home' || state.category || state.system || state.query;

const infoLine = (g) => {
    const m = metaOf(g);
    return [m?.year, m?.genre, m?.players > 1 ? `${m.players} jogadores` : ''].filter(Boolean).join(' · ');
};

const sentence = (g) => {
    const m = metaOf(g);
    if (m?.description) return m.description.length > 260 ? `${m.description.slice(0, 257)}…` : m.description;
    const parts = [`${m?.genre ? `Clássico de ${m.genre.toLowerCase()}` : 'Clássico'} para ${systemName(g.system)}`];
    if (m?.year) parts.push(`lançado em ${m.year}`);
    if (m?.developer) parts.push(`pela ${m.developer}`);
    return `${parts.join(', ')}.`;
};

// ---------- pieces ----------

const tile = (g, extra = '') => {
    const fav = favorites().has(g.title);
    const line = infoLine(g);
    return `<div class="tile" role="button" tabindex="0" data-game="${esc(g.title)}" aria-label="Jogar ${esc(cleanName(g))}">
        ${cover(g.title, g.system)}
        <button class="fav ${fav ? 'on' : ''}" data-fav="${esc(g.title)}" tabindex="-1" aria-label="${fav ? 'Remover da minha lista' : 'Adicionar à minha lista'}">${icon('star')}</button>
        ${canEditCovers ? `<button class="edit-cover" data-edit-cover="${esc(g.title)}" tabindex="-1" title="Trocar capa" aria-label="Trocar capa">${icon('edit')}</button>` : ''}
        <div class="tile-info">
            <b title="${esc(cleanName(g))}">${esc(cleanName(g))}</b>
            <small>${esc(extra || line || systemName(g.system))}</small>
        </div>
        <span class="tile-play">${icon('play')}</span>
    </div>`;
};

const rail = (title, list, opts_ = {}) => {
    if (!list.length) return '';
    const shown = list.slice(0, ROW_LIMIT);
    const more = list.length > shown.length && opts_.more
        ? `<button class="tile more" data-more='${esc(JSON.stringify(opts_.more))}'><span>Ver todos<br><b>${list.length}</b></span>${icon('chevronRight')}</button>` : '';
    return `<section class="rail">
        <h2 class="rail-title">${esc(title)}${opts_.more ? `<button class="rail-all" data-more='${esc(JSON.stringify(opts_.more))}'>Ver todos ${icon('chevronRight')}</button>` : ''}</h2>
        <div class="rail-wrap">
            <button class="rail-btn prev" data-rail="-1" tabindex="-1" aria-label="Voltar">${icon('chevronLeft')}</button>
            <div class="rail-scroll">${shown.map((g) => tile(g, opts_.extra?.(g))).join('')}${more}</div>
            <button class="rail-btn next" data-rail="1" tabindex="-1" aria-label="Avançar">${icon('chevronRight')}</button>
        </div>
    </section>`;
};

// the featured game rotates daily among games that have catalog data
const featured = () => {
    const pool = games.filter((g) => metaOf(g)?.genre && metaOf(g)?.year).sort(byName);
    const list = pool.length ? pool : [...games].sort(byName);
    if (!list.length) return null;
    const day = Math.floor(Date.now() / 86400000);
    return list[day % list.length];
};

const renderHero = () => {
    const el = root.querySelector('[data-hero]');
    const g = isBrowse() ? null : featured();
    if (!g) {
        el.innerHTML = '';
        return;
    }
    const fav = favorites().has(g.title);
    el.innerHTML = `<section class="hero" style="--cover:url('${esc(coverUrl(g.title, g.system))}')">
        <div class="hero-art"></div>
        <div class="hero-body">
            <span class="hero-tag">DESTAQUE DO DIA</span>
            <h1>${esc(cleanName(g))}</h1>
            <p class="hero-line">${esc(infoLine(g) || systemName(g.system))} · ${esc(systemName(g.system))}</p>
            <p class="hero-text">${esc(sentence(g))}</p>
            <div class="hero-actions">
                <button class="btn primary lg" data-game="${esc(g.title)}">${icon('play')}Jogar agora</button>
                <button class="btn lg" data-fav="${esc(g.title)}">${icon('star')}${fav ? 'Na minha lista' : 'Minha lista'}</button>
                ${canEditCovers ? `<button class="btn ghost lg" data-edit-cover="${esc(g.title)}">${icon('edit')}Trocar capa</button>` : ''}
            </div>
        </div>
    </section>`;
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

// ---------- menu ----------

const categories = () => {
    const counts = new Map();
    games.forEach((g) => counts.set(categoryOf(g), (counts.get(categoryOf(g)) || 0) + 1));
    return [...counts.entries()].sort((a, b) => {
        const ia = categoryOrder.indexOf(a[0]), ib = categoryOrder.indexOf(b[0]);
        return (ia < 0 ? 999 : ia) - (ib < 0 ? 999 : ib);
    });
};

const systems = () => {
    const counts = new Map();
    games.forEach((g) => counts.set(g.system, (counts.get(g.system) || 0) + 1));
    return [...counts.entries()].sort((a, b) => systemName(a[0]).localeCompare(systemName(b[0]), 'pt-BR'));
};

const renderMenu = () => {
    const favCount = games.filter((g) => favorites().has(g.title)).length;
    const tab = (id, label, count) => `<button class="nav-tab ${state.tab === id && !state.category && !state.system && !state.query ? 'active' : ''}" data-tab="${id}">${label}${count ? ` <small>${count}</small>` : ''}</button>`;
    const dd = (id, label, active, items) => `<div class="dd ${active ? 'active' : ''}" data-dd="${id}">
        <button class="nav-tab" data-dd-toggle aria-haspopup="true" aria-expanded="false">${label}${active ? `: <b>${esc(active)}</b>` : ''} ${icon('chevronDown')}</button>
        <div class="dd-menu" role="menu" hidden>${items}</div></div>`;
    root.querySelector('[data-menu]').innerHTML = `
        ${tab('home', 'Início')}
        ${tab('fav', 'Minha lista', favCount)}
        ${tab('multi', 'Multijogador')}
        ${dd('category', 'Gêneros', state.category, categories().map(([c, n]) =>
            `<button role="menuitem" class="${state.category === c ? 'on' : ''}" data-category="${esc(c)}">${esc(c)}<small>${n}</small></button>`).join(''))}
        ${dd('system', 'Consoles', state.system ? systemName(state.system) : '', systems().map(([s, n]) =>
            `<button role="menuitem" class="${state.system === s ? 'on' : ''}" data-system="${esc(s)}">${esc(systemName(s))}<small>${n}</small></button>`).join(''))}`;
};

// ---------- pages ----------

const filtered = () => games.filter((g) => {
    if (state.tab === 'fav' && !favorites().has(g.title)) return false;
    if (state.tab === 'multi' && !(metaOf(g)?.players >= 2)) return false;
    if (state.category && categoryOf(g) !== state.category) return false;
    if (state.system && g.system !== state.system) return false;
    if (state.query) {
        const m = metaOf(g);
        const hay = norm([nameOf(g), systemName(g.system), m?.genre, m?.developer, m?.publisher, m?.year].filter(Boolean).join(' '));
        if (!state.query.split(/\s+/).every((w) => hay.includes(norm(w)))) return false;
    }
    return true;
});

const sorter = () => {
    const year = (g) => metaOf(g)?.year || 0;
    switch (state.sort) {
        case 'za':
            return (a, b) => byName(b, a);
        case 'new':
            return (a, b) => year(b) - year(a) || byName(a, b);
        case 'old':
            return (a, b) => (year(a) || 9999) - (year(b) || 9999) || byName(a, b);
        default:
            return byName;
    }
};

const browseTitle = () => {
    if (state.query) return `Resultados para “${state.query}”`;
    if (state.tab === 'fav') return 'Minha lista';
    if (state.tab === 'multi') return 'Multijogador';
    if (state.category) return state.category;
    if (state.system) return systemName(state.system);
    return 'Todos os jogos';
};

const renderBrowse = (el) => {
    const list = filtered().sort(sorter());
    const chips = [
        state.category && `<button class="chip active" data-clear-key="category">${esc(state.category)} ${icon('x')}</button>`,
        state.system && `<button class="chip active" data-clear-key="system">${esc(systemName(state.system))} ${icon('x')}</button>`,
    ].filter(Boolean).join('');
    el.innerHTML = `<div class="browse-head">
            <div><h1>${esc(browseTitle())}</h1><p>${list.length} ${list.length === 1 ? 'jogo' : 'jogos'}</p></div>
            <div class="browse-tools">${chips}
                <label class="select-wrap"><span>Ordem</span><select data-sort>
                    ${[['az', 'A → Z'], ['za', 'Z → A'], ['new', 'Mais novos'], ['old', 'Mais antigos']].map(([v, l]) => `<option value="${v}" ${state.sort === v ? 'selected' : ''}>${l}</option>`).join('')}
                </select></label>
            </div></div>
        ${list.length ? `<div class="tile-grid">${list.map((g) => tile(g)).join('')}</div>`
        : `<div class="empty">${icon(state.tab === 'fav' ? 'star' : 'search', 'xl')}<h3>${state.tab === 'fav' ? 'Sua lista está vazia' : 'Nada encontrado'}</h3>
            <p>${state.tab === 'fav' ? 'Toque na estrela de um jogo para guardá-lo aqui.' : 'Tente outro nome ou limpe os filtros.'}</p>
            <button class="btn outline" data-home>Voltar ao início</button></div>`}`;
};

const renderHome = (el) => {
    const byTitle = new Map(games.map((g) => [g.title, g]));
    const played = recent().filter((r) => byTitle.has(r.title));
    const favs = games.filter((g) => favorites().has(g.title)).sort(byName);
    const multi = games.filter((g) => metaOf(g)?.players >= 2).sort(byName);
    const rows = [
        rail('Continue jogando', played.map((r) => byTitle.get(r.title)), {extra: (g) => ago(played.find((r) => r.title === g.title)?.ts)}),
        rail('Minha lista', favs, {more: {tab: 'fav'}}),
        rail('Para jogar em dupla ou mais', multi, {more: {tab: 'multi'}}),
    ];
    categories().forEach(([c, n]) => {
        if (n < 2) return;
        rows.push(rail(c, games.filter((g) => categoryOf(g) === c).sort(byName), {more: {category: c}}));
    });
    systems().forEach(([s, n]) => {
        if (n < 3) return;
        rows.push(rail(systemName(s), games.filter((g) => g.system === s).sort(byName), {more: {system: s}}));
    });
    el.innerHTML = rows.join('') || `<div class="empty">${icon('server', 'xl')}<h3>Nenhum jogo disponível</h3>
        <p>Os servidores de jogos ainda não se conectaram. Tente novamente em instantes.</p></div>`;
};

const render = () => {
    if (!root) return;
    renderBanner();
    renderMenu();
    renderHero();
    const el = root.querySelector('[data-body]');
    if (!games.length) {
        el.innerHTML = `<div class="empty">${icon('server', 'xl')}<h3>Nenhum jogo disponível</h3>
            <p>Os servidores de jogos ainda não se conectaram. Tente novamente em instantes.</p></div>`;
        return;
    }
    isBrowse() ? renderBrowse(el) : renderHome(el);
};

/** Re-renders keeping the horizontal scroll of every row (so favoriting doesn't jump). */
const renderKeepScroll = () => {
    const pos = [...root.querySelectorAll('.rail-scroll')].map((r) => r.scrollLeft);
    const top = root.closest('.view')?.scrollTop ?? window.scrollY;
    render();
    root.querySelectorAll('.rail-scroll').forEach((r, i) => (r.scrollLeft = pos[i] || 0));
    window.scrollTo({top});
};

// ---------- admin: change a cover ----------

const openCoverEditor = (title) => {
    const g = games.find((x) => x.title === title);
    if (!g) return;
    const m = modal({
        title: `Capa — ${cleanName(g)}`,
        body: `<div class="cover-editor">
            <div class="cover-preview" data-preview>${cover(g.title, g.system)}</div>
            <div class="list">
                <p class="muted">Envie uma imagem para trocar a capa deste jogo, ou para criar uma quando ainda não existir. A imagem é ajustada e guardada neste servidor.</p>
                <input type="file" accept="image/png,image/jpeg,image/gif" data-file hidden>
                <button class="btn primary" data-pick>${icon('upload')}Escolher imagem…</button>
                <button class="btn outline" data-reset>${icon('rotate')}Voltar à capa automática</button>
                <small class="subtle">PNG, JPEG ou GIF, até 8 MB.</small>
            </div></div>`,
        footer: '<button class="btn ghost" data-close>Fechar</button>',
    });
    const refresh = () => {
        bumpCover(g.title, g.system);
        m.el.querySelector('[data-preview]').innerHTML = cover(g.title, g.system);
        renderKeepScroll();
    };
    const file = m.el.querySelector('[data-file]');
    m.el.querySelector('[data-pick]').addEventListener('click', () => file.click());
    file.addEventListener('change', async () => {
        const f = file.files?.[0];
        file.value = '';
        if (!f) return;
        try {
            const res = await fetch(`/api/admin/covers?system=${encodeURIComponent(g.system)}&game=${encodeURIComponent(g.title)}`, {
                method: 'PUT', credentials: 'same-origin', headers: {'Content-Type': f.type || 'image/png'}, body: f,
            });
            if (!res.ok) throw new Error((await res.json().catch(() => ({})))?.error?.message || 'Não foi possível enviar a imagem.');
            toast.success('Capa atualizada!');
            refresh();
        } catch (e) {
            toast.error(e.message);
        }
    });
    m.el.querySelector('[data-reset]').addEventListener('click', async () => {
        if (!await confirmDialog({title: 'Voltar à capa automática?', text: 'A capa enviada será apagada e o servidor procura uma nova.', confirm: 'Restaurar', iconName: 'rotate'})) return;
        try {
            await http.del(`/api/admin/covers?system=${encodeURIComponent(g.system)}&game=${encodeURIComponent(g.title)}`);
            toast.success('Capa automática restaurada.');
            refresh();
        } catch (e) {
            toast.error(e.message);
        }
    });
};

// ---------- data ----------

/** Loads the catalog data. The server fills it in the background, so repeat while games lack data. */
const loadMeta = async (attempt = 0) => {
    clearTimeout(metaTimer);
    try {
        const res = await http.get('/api/meta');
        categoryOrder = res.categories || [];
        const next = new Map();
        (res.entries || []).forEach((e) => next.set(`${(e.system || '').toLowerCase()}|${e.title}`, e));
        const changed = next.size !== metaByKey.size;
        metaByKey = next;
        if (changed) renderKeepScroll();
    } catch {
        // the library works without catalog data
    }
    const missing = games.some((g) => !metaOf(g));
    if (games.length && missing && attempt < 8) {
        metaTimer = setTimeout(() => loadMeta(attempt + 1), 4000 + attempt * 3000);
    }
};

const closeMenus = () => root.querySelectorAll('.dd.open').forEach((d) => {
    d.classList.remove('open');
    d.querySelector('.dd-menu').hidden = true;
    d.querySelector('[data-dd-toggle]').setAttribute('aria-expanded', 'false');
});

export const library = {
    configure(c) {
        canEditCovers = !!c.canEditCovers;
    },
    mount(el, h) {
        root = el;
        handlers = {...handlers, ...h};
        root.innerHTML = `<div class="page catalog">
            <div data-banner></div>
            <div class="catalog-bar">
                <nav class="catalog-menu" data-menu aria-label="Catálogo"></nav>
                <div class="input-wrap search">${icon('search')}
                    <input class="input with-icon" type="search" placeholder="Buscar jogo, gênero, ano… ( / )" aria-label="Buscar jogo" data-search>
                </div>
            </div>
            <div data-hero></div>
            <div data-body></div>
        </div>`;

        const search = root.querySelector('[data-search]');
        search.addEventListener('input', () => {
            state.query = search.value.trim();
            render();
        });
        document.addEventListener('keydown', (e) => {
            if (e.key === '/' && !root.hidden && document.activeElement !== search && !e.target.closest?.('input,textarea,select')) {
                e.preventDefault();
                search.focus();
            }
        });
        document.addEventListener('pointerdown', (e) => {
            if (!e.target.closest?.('.dd')) closeMenus();
        });

        const go = (patch) => {
            Object.assign(state, {tab: 'home', category: '', system: ''}, patch);
            closeMenus();
            if (!patch.query && patch.query !== undefined) search.value = '';
            render();
            window.scrollTo({top: 0});
        };
        on(root, 'click', '[data-tab]', (e, b) => {
            search.value = '';
            go({tab: b.dataset.tab, query: ''});
        });
        on(root, 'click', '[data-dd-toggle]', (e, b) => {
            const d = b.closest('.dd');
            const open = !d.classList.contains('open');
            closeMenus();
            if (open) {
                d.classList.add('open');
                d.querySelector('.dd-menu').hidden = false;
                b.setAttribute('aria-expanded', 'true');
            }
        });
        on(root, 'click', '[data-category]', (e, b) => go({category: b.dataset.category, query: state.query}));
        on(root, 'click', '[data-system]', (e, b) => go({system: b.dataset.system, query: state.query}));
        on(root, 'click', '[data-clear-key]', (e, b) => {
            state[b.dataset.clearKey] = '';
            render();
        });
        on(root, 'click', '[data-home]', () => {
            search.value = '';
            go({query: ''});
        });
        on(root, 'click', '[data-more]', (e, b) => {
            search.value = '';
            go({...JSON.parse(b.dataset.more), query: ''});
        });
        on(root, 'change', '[data-sort]', (e, s) => {
            state.sort = s.value;
            render();
        });
        on(root, 'click', '[data-rail]', (e, b) => {
            const sc = b.closest('.rail-wrap').querySelector('.rail-scroll');
            sc.scrollBy({left: Number(b.dataset.rail) * sc.clientWidth * 0.9, behavior: 'smooth'});
        });
        on(root, 'click', '[data-fav]', (e, b) => {
            e.stopPropagation();
            toggleFav(b.dataset.fav);
            renderKeepScroll();
        });
        on(root, 'click', '[data-edit-cover]', (e, b) => {
            e.stopPropagation();
            openCoverEditor(b.dataset.editCover);
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
        games = [...list];
        if (state.system && !games.some((g) => g.system === state.system)) state.system = '';
        render();
        loadMeta();
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
