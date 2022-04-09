/**
 * Worker manager module.
 * @version 1
 */
const workerManager = (() => {
    const id = 'servers',
        _class = 'server-list',
        trigger = document.getElementById('w'),
        panel = gui.panel(document.getElementById(id), 'WORKERS', 'server-list', null, [
            {
                caption: '⟳',
                cl: ['bold'],
                handler: utils.debounce(handleReload, 1000),
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
                renderer: (data) => data?.id ? data.xid : `${data.xid} x ${data['replicas']}`
            },
            'addr': {
                caption: 'Address',
                renderer: (data) => data?.port ? `${data.addr}:${data.port}` : data.addr
            },
            'is_busy': {
                caption: 'State',
                renderer: (data) => data?.is_busy === true ? 'R' : ''
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
            if (server?.id && state.lastId && state.lastId === server?.xid) {
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
        socket.getServerList();
    }

    function renderIdEl(server) {
        const id = String(index.v()).padStart(2, '0');
        const isActive = server?.id && state.lastId && state.lastId === server?.xid
        return `${(isActive ? '>' : '')}${id}`
    }

    function renderServerChangeEl(server) {
        const handleServerChange = (e) => {
            e.preventDefault();
            window.location.search = `wid=${server.xid}`
            // window.location = window.location.pathname;
            console.log(server.addr, server.id);
        }
        return gui.create('a', (el) => {
            el.innerText = '>>';
            el.href = "#";
            el.addEventListener('click', handleServerChange);
        })
    }

    panel.toggle(false);

    trigger.addEventListener('click', () => {
        handleReload();
        panel.toggle(true);
    })

    const checkLatencies = (data) => {
        const _addresses = data.addresses?.split(',') || [];
        const timeoutMs = 1111;
        // deduplicate
        const addresses = [...new Set(_addresses)];

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

    event.sub(GET_SERVER_LIST, onNewData);

    return {
        checkLatencies,
        whoami,
    }
})(ajax, document, event, gui, log, socket, utils);
