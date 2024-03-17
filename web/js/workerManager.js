import {api} from 'api';
import {
    sub,
    WORKER_LIST_FETCHED
} from 'event'
import {gui} from 'gui';
import {log} from 'log';
import {ajax} from 'network';
import {debounce} from 'utils';

const id = 'servers',
    _class = 'server-list',
    trigger = document.getElementById('w'),
    panel = gui.panel(document.getElementById(id), 'WORKERS', 'server-list', null, [
        {
            caption: '⟳',
            cl: ['bold'],
            handler: debounce(handleReload, 1000),
            title: 'Reload server data',
        }
    ]),
    index = ((i = 1) => ({v: () => i++, r: () => i = 1}))(),
    // caption -- the field caption
    // renderer -- an arbitrary DOM output for the field
    list = {
        'n': {
            renderer: renderIdEl
        },
        'id': {
            caption: 'ID',
            renderer: (data) => data?.in_group ? `${data.id} x ${data.replicas}` : data.id
        },
        'addr': {
            caption: 'Address',
            renderer: (data) => data?.port ? `${data.addr}:${data.port}` : data.addr
        },
        'is_busy': {
            caption: 'State',
            renderer: renderStateEl
        },
        'use': {
            caption: 'Use',
            renderer: renderServerChangeEl
        }
    },
    fields = Object.keys(list);

let state = {
    lastId: null,
    workers: [],
}

const onNewData = (dat = {servers: []}) => {
    panel.setLoad(false);
    index.r();
    state.workers = dat?.servers || [];
    _render(state.workers);
}

function _render(servers = []) {
    if (panel.isHidden()) return;

    const content = gui.fragment();

    if (servers.length === 0) {
        content.append(gui.create('span', (el) => el.innerText = 'No data :('));
        panel.setContent(content);
        return;
    }

    const header = gui.create('div', (el) => {
        el.classList.add(`${_class}__header`);
        fields.forEach(field => el.append(gui.create('span', (f) => f.innerHTML = list[field]?.caption || '')))
    });
    content.append(header)

    const renderRow = (server) => (row) => {
        if (server?.id && state.lastId && state.lastId === server?.id) {
            row.classList.add('active');
        }
        return fields.forEach(field => {
            const val = server.hasOwnProperty(field) ? server[field] : '';
            const renderer = list[field]?.renderer;
            row.append(gui.create('span', (f) => f.append(renderer ? renderer(server) : val)));
        })
    }
    servers.forEach(server => content.append(gui.create('div', renderRow(server))))
    panel.setContent(content);
}

function handleReload() {
    panel.setLoad(true);
    api.server.getWorkerList();
}

function renderIdEl(server) {
    const id = String(index.v()).padStart(2, '0');
    const isActive = server?.id && state.lastId && state.lastId === server?.id
    return `${(isActive ? '>' : '')}${id}`
}

function renderServerChangeEl(server) {
    const handleServerChange = (e) => {
        e.preventDefault();
        window.location.search = `wid=${server.id}`
    }
    return gui.create('a', (el) => {
        el.innerText = '>>';
        el.href = "#";
        el.addEventListener('click', handleServerChange);
    })
}

function renderStateEl(server) {
    const state = server?.is_busy === true ? 'R' : ''
    if (server.room) {
        return gui.create('a', (el) => {
            el.innerText = state;
            el.href = "/?id=" + server.room;
        })
    }
    return state
}

panel.toggle(false);

trigger.addEventListener('click', () => {
    handleReload();
    panel.toggle(true);
})

const checkLatencies = (data) => {
    const timeoutMs = 1111;
    // deduplicate
    const addresses = [...new Set(data.addresses || [])];

    return Promise.all(addresses.map(address => {
        const start = Date.now();
        return ajax.fetch(`${address}?_=${start}`, {method: "GET", redirect: "follow"}, timeoutMs)
            .then(() => ({[address]: Date.now() - start}))
            .catch(() => ({[address]: 9999}));
    }))
};

const whoami = (id) => {
    state.lastId = id;
    _render(state.workers);
}

sub(WORKER_LIST_FETCHED, onNewData);

/**
 * Worker manager module.
 */
export const workerManager = {
    checkLatencies,
    whoami,
}
