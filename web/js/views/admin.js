import {http} from '../http.js?v=4';
import {icon} from '../ui/icons.js?v=4';
import {ago, avatar, dateTime, esc, on} from '../ui/dom.js?v=4';
import {confirmDialog, modal} from '../ui/modal.js?v=4';
import {toast} from '../ui/toast.js?v=4';
import {auditActions, gameTitle, roles, statuses} from '../i18n.js?v=4';

/**
 * Administration: account approval, users, rooms/queue, servers and audit.
 */

let root;
let me = null;
let moderatorsEnabled = false;
let tab = 'pending';
let users = [];
let overview = null;
let audit = [];
let search = '';
let statusFilter = '';
let refreshTimer = null;
let onPendingCount = () => ({});

const can = (p) => me?.perms?.includes(p);

const statusBadge = (s) => {
    const cls = {pending: 'warning', approved: 'success', rejected: 'danger', blocked: 'danger'}[s] || '';
    return `<span class="badge ${cls}">${esc(statuses[s] || s)}</span>`;
};

const roleBadge = (r) => `<span class="badge ${r === 'admin' ? 'accent' : r === 'moderator' ? 'cyan' : ''}">${esc(roles[r] || r)}</span>`;

// Data

const loadUsers = async () => {
    const {users: list} = await http.get('/api/admin/users');
    users = list || [];
    onPendingCount(users.filter((u) => u.status === 'pending').length);
};

const loadOverview = async () => {
    if (!can('rooms.moderate')) return;
    overview = await http.get('/api/admin/overview');
};

const loadAudit = async () => {
    if (!can('users.manage')) return;
    const {entries} = await http.get('/api/admin/audit');
    audit = entries || [];
};

const refresh = async (what = tab) => {
    try {
        if (what === 'pending' || what === 'users') await loadUsers();
        if (what === 'rooms' || what === 'servers') await loadOverview();
        if (what === 'audit') await loadAudit();
    } catch (e) {
        toast.error(e.message);
    }
    render();
};

// Actions

const act = async (id, action, okText) => {
    try {
        await http.post(`/api/admin/users/${id}/${action}`);
        toast.success(okText);
        await refresh('users');
    } catch (e) {
        toast.error(e.message);
    }
};

const userById = (id) => users.find((u) => u.id === +id);

const actions = {
    approve: (u) => act(u.id, 'approve', `${u.name} foi aprovado.`),
    reject: async (u) => {
        if (await confirmDialog({title: `Recusar ${u.name}?`, text: 'A pessoa não poderá acessar o Arcade com esta conta.', confirm: 'Recusar', danger: true, iconName: 'x'}))
            act(u.id, 'reject', 'Cadastro recusado.');
    },
    block: async (u) => {
        if (await confirmDialog({title: `Bloquear ${u.name}?`, text: 'A pessoa será desconectada imediatamente e não poderá mais entrar.', confirm: 'Bloquear', danger: true, iconName: 'ban'}))
            act(u.id, 'block', `${u.name} foi bloqueado.`);
    },
    unblock: (u) => act(u.id, 'unblock', `${u.name} foi desbloqueado.`),
    reset: async (u) => {
        if (!await confirmDialog({title: `Redefinir a senha de ${u.name}?`, text: 'Uma senha temporária será gerada e as sessões atuais serão encerradas.', confirm: 'Redefinir', iconName: 'key'})) return;
        try {
            const {password} = await http.post(`/api/admin/users/${u.id}/reset-password`);
            modal({
                title: 'Senha temporária',
                body: `<p class="muted">Envie esta senha para <b>${esc(u.name)}</b> por um canal seguro. Ela não será mostrada novamente.</p>
                    <div class="input-wrap" style="margin-top:16px"><input class="input mono" readonly value="${esc(password)}" style="font-size:18px;letter-spacing:.08em">
                    <button class="btn primary sm input-action" data-copy="${esc(password)}">${icon('copy')}Copiar</button></div>`,
                footer: '<button class="btn primary" data-close>Pronto</button>',
            }).el.querySelector('[data-copy]').addEventListener('click', async (e) => {
                await navigator.clipboard?.writeText(e.currentTarget.dataset.copy);
                toast.success('Senha copiada.');
            });
        } catch (e) {
            toast.error(e.message);
        }
    },
    role: async (u, role) => {
        try {
            await http.put(`/api/admin/users/${u.id}/role`, {role});
            toast.success(`${u.name} agora é ${roles[role]}.`);
            await refresh('users');
        } catch (e) {
            toast.error(e.message);
        }
    },
    retryCovers: async () => {
        try {
            await http.post('/api/admin/covers/retry');
            toast.success('Os jogos sem capa serão buscados novamente na próxima exibição.');
            await refresh('servers');
        } catch (e) {
            toast.error(e.message);
        }
    },
    retryMeta: async () => {
        try {
            await http.post('/api/admin/meta/retry');
            toast.success('Atualizando o catálogo em segundo plano. Os dados aparecem na biblioteca em instantes.');
            await refresh('servers');
        } catch (e) {
            toast.error(e.message);
        }
    },
    closeRoom: async (id, game) => {
        if (!await confirmDialog({title: 'Encerrar sala?', text: `Todos em “${gameTitle(game).title}” serão desconectados. O progresso não salvo será perdido.`, confirm: 'Encerrar', danger: true, iconName: 'door'})) return;
        try {
            await http.post('/api/admin/rooms/close', {room: id});
            toast.success('Sala encerrada.');
            setTimeout(() => refresh('rooms'), 800);
        } catch (e) {
            toast.error(e.message);
        }
    },
};

// Rendering

const tabs = () => {
    const pending = users.filter((u) => u.status === 'pending').length;
    const list = [
        ['pending', 'Pendentes', 'hourglass', pending, true],
        ['users', 'Usuários', 'users', 0, true],
        ['rooms', 'Salas e fila', 'tv', 0, can('rooms.moderate')],
        ['servers', 'Servidores', 'server', 0, can('rooms.moderate')],
        ['audit', 'Auditoria', 'list', 0, can('users.manage')],
    ];
    return `<div class="tabs" role="tablist">${list.filter((t) => t[4]).map(([k, t, i, n]) =>
        `<button class="tab ${tab === k ? 'active' : ''}" data-atab="${k}">${icon(i)}${t}${n ? ` <span class="counter">${n}</span>` : ''}</button>`).join('')}</div>`;
};

const renderPending = () => {
    const list = users.filter((u) => u.status === 'pending');
    if (!list.length) {
        return `<div class="card empty">${icon('check', 'xl')}<h3>Nenhum cadastro aguardando</h3>
            <p>Quando alguém criar uma conta, ela aparece aqui e você recebe um aviso.</p></div>`;
    }
    return `<div class="list">${list.map((u) => `<div class="pending-card">
        ${avatar(u.name, 'lg', u.id)}
        <div class="grow"><b>${esc(u.name)}</b><small>${esc(u.email)} · cadastrou-se ${esc(ago(u.created_at))}</small></div>
        <button class="btn danger" data-act="reject" data-id="${u.id}">${icon('x')}Recusar</button>
        <button class="btn success" data-act="approve" data-id="${u.id}">${icon('check')}Aprovar</button>
    </div>`).join('')}</div>`;
};

const renderUsers = () => {
    const q = search.toLowerCase();
    const list = users.filter((u) => (!statusFilter || u.status === statusFilter) &&
        (!q || u.name.toLowerCase().includes(q) || u.email.toLowerCase().includes(q)));
    const manage = can('users.manage');
    const roleSelect = (u) => {
        if (!manage || u.primary || u.id === me.id) return roleBadge(u.role);
        // only the primary account is an administrator
        const opts = [['user', roles.user], ...(moderatorsEnabled || u.role === 'moderator' ? [['moderator', roles.moderator]] : [])];
        if (opts.length < 2) return roleBadge(u.role);
        return `<select class="input" data-role="${u.id}" style="height:32px;width:150px">${opts.map(([v, l]) =>
            `<option value="${v}" ${u.role === v ? 'selected' : ''}>${l}</option>`).join('')}</select>`;
    };
    const rowActions = (u) => {
        if (u.primary || u.id === me.id) return '<span class="subtle">—</span>';
        const out = [];
        if (u.status === 'pending' || u.status === 'rejected') out.push(`<button class="btn success sm" data-act="approve" data-id="${u.id}">Aprovar</button>`);
        if (manage && u.status === 'approved') out.push(`<button class="btn danger sm" data-act="block" data-id="${u.id}">Bloquear</button>`);
        if (manage && u.status === 'blocked') out.push(`<button class="btn sm" data-act="unblock" data-id="${u.id}">Desbloquear</button>`);
        if (manage) out.push(`<button class="btn ghost icon-only sm" data-act="reset" data-id="${u.id}" title="Redefinir senha" aria-label="Redefinir senha">${icon('key')}</button>`);
        return `<div class="row-actions">${out.join('')}</div>`;
    };
    return `<div class="admin-toolbar">
            <div class="input-wrap" style="flex:1;max-width:360px">${icon('search')}
                <input class="input with-icon" type="search" placeholder="Buscar por nome ou e-mail" value="${esc(search)}" data-usearch></div>
            <div class="chips">${[['', 'Todos'], ['approved', 'Aprovados'], ['pending', 'Pendentes'], ['blocked', 'Bloqueados'], ['rejected', 'Recusados']].map(([v, l]) =>
                `<button class="chip ${statusFilter === v ? 'active' : ''}" data-sfilter="${v}">${l}</button>`).join('')}</div>
        </div>
        <div class="table-wrap"><table class="table">
            <thead><tr><th>Pessoa</th><th>Situação</th><th>Papel</th><th>Cadastro</th><th>Último acesso</th><th></th></tr></thead>
            <tbody>${list.length ? list.map((u) => `<tr>
                <td><div class="cell-user">${avatar(u.name, '', u.id)}<div><b>${esc(u.name)}${u.primary ? ` <span class="badge accent">${icon('shield')}Principal</span>` : ''}</b>
                    <small>${esc(u.email)}</small></div>${u.online ? '<span class="dot success" title="Online agora"></span>' : ''}</div></td>
                <td>${statusBadge(u.status)}</td>
                <td>${roleSelect(u)}</td>
                <td class="muted">${esc(dateTime(u.created_at))}</td>
                <td class="muted">${u.last_login_at ? esc(ago(u.last_login_at)) : 'Nunca'}</td>
                <td>${rowActions(u)}</td></tr>`).join('')
                : '<tr><td colspan="6" class="muted" style="text-align:center;padding:32px">Nenhum usuário encontrado.</td></tr>'}</tbody>
        </table></div>
        ${moderatorsEnabled ? '' : '<p class="subtle" style="font-size:12px">O papel de Moderador (pode aprovar cadastros) está desativado. Ative em <span class="mono">coordinator.auth.moderators</span> quando quiser delegar aprovações.</p>'}`;
};

const renderRooms = () => {
    if (!overview) return '<div class="skeleton" style="height:200px"></div>';
    const {rooms = [], queue = [], online = []} = overview;
    return `<div class="stat-row">
            <div class="stat"><span>Online agora</span><b>${online.length}</b></div>
            <div class="stat"><span>Salas ativas</span><b>${rooms.length}</b></div>
            <div class="stat"><span>Na fila</span><b>${queue.length}</b></div>
        </div>
        <h3>Salas ativas</h3>
        ${rooms.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>Jogo</th><th>Anfitrião</th><th>Jogadores</th><th>Assistindo</th><th></th></tr></thead><tbody>
            ${rooms.map((r) => `<tr><td><b>${esc(gameTitle(r.game).title)}</b></td><td>${esc(r.host || '—')}</td>
                <td>${(r.players || []).map((p) => `${esc(p.name)} <span class="subtle">(J${p.slot + 1})</span>`).join(', ') || '—'}</td>
                <td>${r.spectators}</td>
                <td><div class="row-actions"><button class="btn danger sm" data-close-room="${esc(r.id)}" data-game="${esc(r.game)}">${icon('door')}Encerrar</button></div></td></tr>`).join('')}
            </tbody></table></div>` : '<div class="card empty" style="padding:24px"><p>Nenhuma sala ativa.</p></div>'}
        <h3>Fila de espera</h3>
        ${queue.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>#</th><th>Pessoa</th><th>Jogo</th><th>Situação</th><th>Desde</th></tr></thead><tbody>
            ${queue.map((q, i) => `<tr><td>${i + 1}º</td><td>${esc(q.name)}${q.connected ? '' : ' <span class="badge warning">desconectado</span>'}</td>
                <td>${esc(gameTitle(q.game).title)}</td><td>${q.status === 'called' ? '<span class="badge success">Chamado</span>' : '<span class="badge">Aguardando</span>'}</td>
                <td class="muted">${esc(ago(q.since))}</td></tr>`).join('')}
            </tbody></table></div>` : '<div class="card empty" style="padding:24px"><p>Ninguém na fila.</p></div>'}
        <h3>Pessoas online</h3>
        ${online.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>Pessoa</th><th>Onde está</th></tr></thead><tbody>
            ${online.map((o) => `<tr><td><div class="cell-user">${avatar(o.name, 'sm')}<div><b>${esc(o.name)}</b><small>${esc(o.email)}</small></div></div></td>
                <td>${o.room ? `${esc(gameTitle(o.room).title)} · ${o.slot >= 0 ? `Jogador ${o.slot + 1}` : 'assistindo'}` : '<span class="muted">Navegando</span>'}</td></tr>`).join('')}
            </tbody></table></div>` : ''}`;
};

const renderServers = () => {
    if (!overview) return '<div class="skeleton" style="height:200px"></div>';
    const ws = overview.workers || [];
    const free = ws.filter((w) => !w.busy && !w.reserved).length;
    const cv = overview.covers;
    const mt = overview.meta;
    return `<div class="stat-row">
            <div class="stat"><span>Servidores</span><b>${ws.length}</b></div>
            <div class="stat"><span>Livres</span><b>${free}</b></div>
            <div class="stat"><span>Ocupados</span><b>${ws.filter((w) => w.busy).length}</b></div>
            ${mt ? `<div class="stat"><span>Jogos catalogados</span><b>${mt.found}<small class="subtle" style="font-size:14px"> · ${mt.missing} sem dados</small></b></div>` : ''}
            ${cv ? `<div class="stat"><span>Capas salvas</span><b>${cv.found}<small class="subtle" style="font-size:14px"> · ${cv.missing} sem capa</small></b></div>` : ''}
        </div>
        ${cv && can('users.manage') ? `<div class="card" style="display:flex;align-items:center;gap:16px;padding:16px">
            ${icon('library', 'lg')}
            <div style="flex:1"><b>Capas dos jogos</b><p class="muted" style="font-size:13px">Baixadas uma única vez do acervo gratuito libretro-thumbnails e guardadas neste servidor. Jogos sem capa são verificados de novo a cada 7 dias.</p></div>
            <button class="btn outline" data-retry-covers>${icon('refresh')}Buscar capas faltantes</button>
        </div>` : ''}
        ${mt && can('users.manage') ? `<div class="card" style="display:flex;align-items:center;gap:16px;padding:16px">
            ${icon('library', 'lg')}
            <div style="flex:1"><b>Catálogo dos jogos</b><p class="muted" style="font-size:13px">Gênero, ano, desenvolvedora e número de jogadores vêm do banco público libretro-database e ficam guardados neste servidor. Atualizar busca tudo de novo.</p></div>
            <button class="btn outline" data-retry-meta>${icon('refresh')}Atualizar catálogo</button>
        </div>` : ''}
        ${ws.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>Servidor</th><th>Endereço</th><th>Região</th><th>Jogos</th><th>Situação</th></tr></thead><tbody>
            ${ws.map((w) => `<tr><td class="mono">${esc(w.tag || w.id.slice(-8))}</td><td class="mono">${esc(w.addr)}</td><td>${esc(w.zone || '—')}</td><td>${w.games}</td>
                <td>${w.busy ? `<span class="badge warning">Em jogo</span> <span class="muted">${esc(gameTitle(w.room ? w.room.split('___')[1] || '' : '').title)}</span>`
                    : w.reserved ? `<span class="badge cyan">Reservado para ${esc(w.reserved)}</span>` : '<span class="badge success">Livre</span>'}</td></tr>`).join('')}
            </tbody></table></div>` : `<div class="card empty">${icon('server', 'xl')}<h3>Nenhum servidor conectado</h3><p>Inicie um worker apontando para este coordinator.</p></div>`}`;
};

const renderAudit = () => audit.length ? `<div class="table-wrap"><table class="table">
    <thead><tr><th>Quando</th><th>Quem</th><th>Ação</th><th>Alvo</th><th>Detalhes</th></tr></thead><tbody>
    ${audit.map((a) => `<tr><td class="muted" style="white-space:nowrap">${esc(dateTime(a.created_at))}</td><td>${esc(a.actor || 'Sistema')}</td>
        <td>${esc(auditActions[a.action] || a.action)}</td><td>${esc(a.target || '—')}</td><td class="muted">${esc(a.details || '')}</td></tr>`).join('')}
    </tbody></table></div>` : `<div class="card empty">${icon('list', 'xl')}<h3>Sem registros</h3></div>`;

const render = () => {
    if (!root) return;
    const body = {pending: renderPending, users: renderUsers, rooms: renderRooms, servers: renderServers, audit: renderAudit}[tab];
    const searchFocused = document.activeElement?.matches?.('[data-usearch]');
    const caret = searchFocused ? document.activeElement.selectionStart : 0;
    root.querySelector('[data-tabs]').innerHTML = tabs();
    root.querySelector('[data-body]').innerHTML = body();
    if (searchFocused) {
        const s = root.querySelector('[data-usearch]');
        s?.focus();
        s?.setSelectionRange(caret, caret);
    }
};

const startAutoRefresh = () => {
    clearInterval(refreshTimer);
    refreshTimer = setInterval(() => {
        if (root && !root.hidden && (tab === 'rooms' || tab === 'servers')) refresh();
    }, 5000);
};

export const admin = {
    mount(el, {onPending} = {}) {
        root = el;
        onPendingCount = onPending || onPendingCount;
        root.innerHTML = `<div class="page admin-layout">
            <div class="page-head"><div><h1>Administração</h1><p>Aprove cadastros, gerencie pessoas e acompanhe os servidores.</p></div>
            <button class="btn outline" data-refresh>${icon('refresh')}Atualizar</button></div>
            <div data-tabs></div>
            <div class="admin-layout" data-body></div>
        </div>`;
        on(root, 'click', '[data-atab]', (e, b) => {
            tab = b.dataset.atab;
            render();
            refresh();
        });
        on(root, 'click', '[data-refresh]', () => refresh());
        on(root, 'click', '[data-act]', (e, b) => {
            const u = userById(b.dataset.id);
            if (u) actions[b.dataset.act]?.(u);
        });
        on(root, 'change', '[data-role]', (e, s) => {
            const u = userById(s.dataset.role);
            if (u) actions.role(u, s.value);
        });
        on(root, 'click', '[data-retry-covers]', () => actions.retryCovers());
        on(root, 'click', '[data-retry-meta]', () => actions.retryMeta());
        on(root, 'click', '[data-close-room]', (e, b) => actions.closeRoom(b.dataset.closeRoom, b.dataset.game));
        on(root, 'click', '[data-sfilter]', (e, b) => {
            statusFilter = b.dataset.sfilter;
            render();
        });
        on(root, 'input', '[data-usearch]', (e, i) => {
            search = i.value;
            render();
        });
        startAutoRefresh();
    },
    configure({user, moderators}) {
        me = user;
        moderatorsEnabled = !!moderators;
    },
    /** Called when the view becomes visible. */
    show() {
        render();
        refresh();
    },
    /** Called when the pending count changes in real time. */
    pendingChanged() {
        if (root && !root.hidden) refresh('users');
    },
};
