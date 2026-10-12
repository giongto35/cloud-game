import {http} from '../http.js?v=4';
import {icon} from '../ui/icons.js?v=4';
import {ago, avatar, esc, on} from '../ui/dom.js?v=4';
import {cover} from '../ui/covers.js?v=4';
import {confirmDialog, modal} from '../ui/modal.js?v=4';
import {toast} from '../ui/toast.js?v=4';
import {gameTitle} from '../i18n.js?v=4';

/**
 * Tournaments: list, creation, live bracket, match calls and replays.
 */

let root;
let me = null;
let h = {onPlayMatch: () => ({}), onWatchMatch: () => ({}), onWatchReplay: () => ({})};
let current = null; // tournament id on screen (null = list)
let data = null;
let loading = false;

const tStatus = {
    registering: ['Inscrições abertas', 'success'],
    running: ['Em andamento', 'live'],
    finished: ['Encerrado', ''],
    canceled: ['Cancelado', 'danger'],
};

const mStatus = {
    pending: ['Aguardando adversário', ''],
    ready: ['Na fila', 'warning'],
    starting: ['Chamando jogadores', 'cyan'],
    running: ['Ao vivo', 'live'],
    finished: ['Encerrada', ''],
    stalled: ['Decisão da administração', 'danger'],
};

const sourceName = {ram: 'resultado automático', admin: 'definido pela administração', walkover: 'W.O.'};

const badge = ([text, cls]) => `<span class="badge ${cls === 'live' ? 'danger live' : cls}">${esc(text)}</span>`;

// List

const renderList = async () => {
    root.innerHTML = `<div class="page"><div class="page-head"><div><h1>Torneios</h1>
        <p>Chaves eliminatórias com resultado detectado automaticamente e replays de todas as partidas.</p></div>
        <div data-actions></div></div><div data-body><div class="skeleton" style="height:180px"></div></div></div>`;
    let res;
    try {
        res = await http.get('/api/tournaments');
    } catch (e) {
        root.querySelector('[data-body]').innerHTML = `<div class="empty">${icon('alert', 'xl')}<h3>${esc(e.message)}</h3></div>`;
        return;
    }
    if (current !== null) return;
    const sch = res.scheduler;
    root.querySelector('[data-actions]').innerHTML = res.can_manage
        ? `<button class="btn primary" data-create>${icon('crown')}Criar torneio</button>` : '';
    const list = res.tournaments || [];
    root.querySelector('[data-body]').innerHTML = `
        <div class="stat-row" style="margin-bottom:24px">
            <div class="stat"><span>Partidas em andamento</span><b>${sch.running}<small class="subtle" style="font-size:16px"> / ${sch.max}</small></b></div>
            <div class="stat"><span>Partidas na fila</span><b>${sch.queued}</b></div>
            <div class="stat"><span>Torneios</span><b>${list.length}</b></div>
        </div>
        ${list.length ? `<div class="room-grid">${list.map((t) => `
            <article class="room-card t-card" data-open="${t.id}" role="button" tabindex="0">
                ${cover(t.game, t.system, badge(tStatus[t.status] || [t.status, '']))}
                <div class="room-body">
                    <div><h3 class="ellipsis">${esc(t.name)}</h3>
                    <small class="muted">${esc(t.rule_name || gameTitle(t.game).title)} · criado ${esc(ago(t.created_at))}</small></div>
                    <div class="room-players">${icon('users')} ${t.players} ${t.players === 1 ? 'participante' : 'participantes'}
                        ${t.joined ? '<span class="badge accent">Você está inscrito</span>' : ''}
                        ${t.winner ? `<span class="badge warning">${icon('crown')}${esc(t.winner.name)}</span>` : ''}</div>
                </div>
            </article>`).join('')}</div>`
        : `<div class="card empty">${icon('crown', 'xl')}<h3>Nenhum torneio ainda</h3>
            <p>${res.can_manage ? 'Crie o primeiro torneio e abra as inscrições.' : 'Quando a administração abrir um torneio, ele aparece aqui.'}</p></div>`}`;
};

const openCreate = async () => {
    let games = [];
    try {
        games = (await http.get('/api/tournaments/games')).games || [];
    } catch (e) {
        toast.error(e.message);
        return;
    }
    if (!games.length) {
        toast.warning('Nenhum jogo com detecção de resultado está disponível nos servidores.');
        return;
    }
    const m = modal({
        title: 'Criar torneio',
        body: `<form class="list" data-form>
            <div class="field"><label for="t-name">Nome do torneio</label>
                <input class="input" id="t-name" maxlength="60" required placeholder="Ex.: Copa Belko de Kombat" autofocus></div>
            <div class="field"><label for="t-game">Jogo</label>
                <select class="input" id="t-game">${games.map((g, i) => `<option value="${i}">${esc(gameTitle(g.title).title)} — ${esc(g.name)}</option>`).join('')}</select></div>
            <div class="banner" style="margin:0;border-color:var(--border)">${icon('info', 'lg')}<div class="grow"><p data-hint>${esc(games[0].hint)}</p></div></div>
            <p class="subtle" style="font-size:12px">Eliminatória simples com chaves sorteadas. O vencedor de cada partida é detectado automaticamente pela memória do jogo e cada partida é gravada em replay.</p>
        </form>`,
        footer: '<button class="btn ghost" data-close>Cancelar</button><button class="btn primary" data-save>Criar e abrir inscrições</button>',
    });
    const sel = m.el.querySelector('#t-game');
    sel.addEventListener('change', () => (m.el.querySelector('[data-hint]').textContent = games[+sel.value].hint));
    const submit = async () => {
        const btn = m.el.querySelector('[data-save]');
        btn.classList.add('is-loading');
        try {
            const {id} = await http.post('/api/tournaments', {name: m.el.querySelector('#t-name').value, game: games[+sel.value].title});
            m.close();
            toast.success('Torneio criado! As inscrições estão abertas.');
            location.hash = `#/torneios/${id}`;
        } catch (e) {
            toast.error(e.message);
        } finally {
            btn.classList.remove('is-loading');
        }
    };
    m.el.querySelector('[data-save]').addEventListener('click', submit);
    m.el.querySelector('[data-form]').addEventListener('submit', (e) => {
        e.preventDefault();
        submit();
    });
};

// Detail

const player = (p, winner, mine) => {
    if (!p) return '<div class="t-player empty"><span class="subtle">A definir</span></div>';
    const won = winner && winner === p.id;
    const lost = winner && winner !== p.id;
    return `<div class="t-player ${won ? 'won' : ''} ${lost ? 'lost' : ''} ${p.id === mine ? 'me' : ''}">
        ${avatar(p.name, 'sm')}<span class="ellipsis">${esc(p.name)}</span>${won ? icon('crown') : ''}</div>`;
};

const matchCard = (mt, manage) => {
    const st = mStatus[mt.status] || [mt.status, ''];
    const bye = mt.source === 'walkover' && (!mt.p1 || !mt.p2);
    const acts = [];
    if (mt.live) acts.push(`<button class="btn sm primary" data-watch="${mt.id}">${icon('eye')}Assistir</button>`);
    if (mt.replay) acts.push(`<button class="btn sm outline" data-replay="${mt.replay}">${icon('play')}Replay</button>`);
    if (manage && mt.p1 && mt.p2 && mt.status !== 'finished' && mt.status !== 'pending') {
        acts.push(`<button class="btn sm ghost" data-decide="${mt.id}">${icon('shield')}Definir vencedor</button>`);
    }
    return `<div class="t-match ${mt.status}">
        <div class="t-match-head">${bye ? '<span class="badge">Avanço direto</span>' : badge(st)}
            ${mt.score ? `<span class="mono subtle">${esc(mt.score)}</span>` : ''}</div>
        ${player(mt.p1, mt.winner, me?.id)}${player(mt.p2, mt.winner, me?.id)}
        ${mt.source && mt.status === 'finished' && !bye ? `<small class="subtle">${esc(sourceName[mt.source] || mt.source)}</small>` : ''}
        ${acts.length ? `<div class="t-match-actions">${acts.join('')}</div>` : ''}
    </div>`;
};

const renderDetail = () => {
    if (!data) return;
    const t = data.tournament;
    const manage = data.can_manage;
    const actions = [];
    if (t.status === 'registering') {
        actions.push(t.joined
            ? `<button class="btn outline" data-act="leave">${icon('x')}Cancelar inscrição</button>`
            : `<button class="btn primary" data-act="join">${icon('userPlus')}Participar</button>`);
        if (manage) actions.push(`<button class="btn success" data-act="start" ${t.players < 2 ? 'disabled title="São necessários 2 participantes"' : ''}>${icon('play')}Sortear chaves e iniciar</button>`);
    }
    if (manage && (t.status === 'registering' || t.status === 'running')) {
        actions.push(`<button class="btn danger" data-act="cancel">${icon('ban')}Cancelar torneio</button>`);
    }
    const mine = data.my_match;
    const sch = data.scheduler;
    root.innerHTML = `<div class="page">
        <a class="btn ghost sm" href="#/torneios" style="margin-bottom:12px">${icon('arrowLeft')}Torneios</a>
        <div class="page-head"><div>
            <h1>${esc(t.name)} ${badge(tStatus[t.status] || [t.status, ''])}</h1>
            <p>${esc(t.rule_name)} · ${t.players} ${t.players === 1 ? 'participante' : 'participantes'}${t.started_at ? ` · começou ${esc(ago(t.started_at))}` : ''}</p>
        </div><div class="row-actions">${actions.join('')}</div></div>
        ${t.winner ? `<div class="banner t-champion">${icon('crown', 'xl')}<div class="grow"><b>Campeão: ${esc(t.winner.name)}</b><p>Parabéns! Os replays de todas as partidas estão disponíveis abaixo.</p></div></div>` : ''}
        ${mine ? `<div class="banner t-mine">${icon('gamepad', 'lg')}<div class="grow"><b>Sua partida ${mine.status === 'running' ? 'está em andamento' : 'está pronta'}!</b>
            <p>${esc(mine.p1?.name || '')} × ${esc(mine.p2?.name || '')} — entre agora; se não entrar a tempo, a vitória vai para o adversário.</p></div>
            <button class="btn primary lg" data-play="${mine.id}">${icon('play')}Entrar na partida</button></div>` : ''}
        <div class="banner" style="border-color:var(--border);background:var(--surface-1)">${icon('info', 'lg')}<div class="grow"><b>Como funciona</b><p>${esc(t.hint)}</p></div>
            ${t.status === 'running' ? `<span class="badge cyan">${sch.running.length}/${sch.max} partidas em andamento · ${sch.queued.length} na fila</span>` : ''}</div>
        ${data.rounds.length ? `<div class="bracket">${data.rounds.map((r) => `
            <div class="bracket-col"><div class="section-title">${esc(r.name)}</div>
                <div class="bracket-matches">${r.matches.map((mt) => matchCard(mt, manage)).join('')}</div></div>`).join('')}</div>`
        : `<div class="section-title">${icon('users')}Inscritos</div>
            ${data.players.length ? `<div class="chips">${data.players.map((p) => `<span class="chip">${avatar(p.name, 'sm')}${esc(p.name)}</span>`).join('')}</div>`
            : '<p class="muted">Ninguém se inscreveu ainda.</p>'}`}
    </div>`;
};

const loadDetail = async (id) => {
    loading = true;
    try {
        data = await http.get(`/api/tournaments/${id}`);
        if (current === id) renderDetail();
    } catch (e) {
        root.innerHTML = `<div class="page"><div class="empty">${icon('alert', 'xl')}<h3>${esc(e.message)}</h3>
            <a class="btn outline" href="#/torneios">Voltar</a></div></div>`;
    } finally {
        loading = false;
    }
};

const act = async (action) => {
    const id = current;
    if (action === 'cancel' && !await confirmDialog({title: 'Cancelar o torneio?', text: 'As partidas em andamento serão encerradas.', confirm: 'Cancelar torneio', danger: true, iconName: 'ban'})) return;
    if (action === 'start' && !await confirmDialog({title: 'Sortear as chaves e iniciar?', text: 'As inscrições serão fechadas e as primeiras partidas chamadas.', confirm: 'Iniciar', iconName: 'crown'})) return;
    try {
        await http.post(`/api/tournaments/${id}/${action}`);
        toast.success({join: 'Inscrição confirmada!', leave: 'Inscrição cancelada.', start: 'Chaves sorteadas! As partidas estão sendo chamadas.', cancel: 'Torneio cancelado.'}[action]);
        loadDetail(id);
    } catch (e) {
        toast.error(e.message);
    }
};

const decide = (matchId) => {
    const mt = data.rounds.flatMap((r) => r.matches).find((x) => x.id === matchId);
    if (!mt) return;
    const m = modal({
        title: 'Definir vencedor',
        body: `<p class="muted">Use quando a detecção automática não for possível (desconexão, problema técnico ou ausência).</p>
            <div class="list" style="margin-top:16px">${[mt.p1, mt.p2].map((p) => `<button class="btn outline lg block" data-win="${p.id}">${avatar(p.name, 'sm')}${esc(p.name)} venceu</button>`).join('')}</div>`,
    });
    m.el.addEventListener('click', async (e) => {
        const b = e.target.closest('[data-win]');
        if (!b) return;
        try {
            await http.post(`/api/matches/${matchId}/winner`, {winner: +b.dataset.win});
            m.close();
            toast.success('Resultado registrado.');
            loadDetail(current);
        } catch (err) {
            toast.error(err.message);
        }
    });
};

export const tournaments = {
    mount(el, handlers) {
        root = el;
        h = {...h, ...handlers};
        on(root, 'click', '[data-create]', openCreate);
        on(root, 'click', '[data-open]', (e, c) => (location.hash = `#/torneios/${c.dataset.open}`));
        on(root, 'keydown', '[data-open]', (e, c) => e.key === 'Enter' && (location.hash = `#/torneios/${c.dataset.open}`));
        on(root, 'click', '[data-act]', (e, b) => act(b.dataset.act));
        on(root, 'click', '[data-play]', (e, b) => h.onPlayMatch(+b.dataset.play, data?.tournament?.game));
        on(root, 'click', '[data-watch]', (e, b) => h.onWatchMatch(+b.dataset.watch, data?.tournament?.game));
        on(root, 'click', '[data-replay]', (e, b) => h.onWatchReplay(+b.dataset.replay, data?.tournament?.game));
        on(root, 'click', '[data-decide]', (e, b) => decide(+b.dataset.decide));
    },
    configure({user}) {
        me = user;
    },
    /** Shows the list (id = null) or a tournament. */
    show(id = null) {
        current = id;
        data = null;
        if (id === null) renderList();
        else {
            root.innerHTML = '<div class="page"><div class="skeleton" style="height:320px"></div></div>';
            loadDetail(id);
        }
    },
    /** Called on real-time updates. */
    refresh() {
        if (!root || root.hidden || loading) return;
        current === null ? renderList() : loadDetail(current);
    },
};
