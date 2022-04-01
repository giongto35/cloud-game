/**
 * Server list module.
 * @version 1
 */
const serverList = (() => {
    const id = 'servers',
        _class = 'server-list',
        trigger = document.getElementById('w'),
        panel = gui.panel(document.getElementById(id), 'WORKERS', 'server-list', null, [
            {
                caption: '⟳',
                cl: ['bold'],
                handler: handleReload,
                title: 'Reload server data',
            }
        ]),
        index = ((i = 1) => ({v: () => i++, r: () => i = 1}))(),
        // caption -- the field caption
        // renderer -- an arbitrary DOM output for the field
        list = {
            'n': {
                renderer: () => String(index.v()).padStart(2, '0')
            },
            'id': {
                caption: 'ID',
                renderer: (data) => data?.id ? data.id : `??? [replicated] x ${data['replicas']}`
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

    // root.classList.add("hidden");

    // waiting for the socket to request data
    const onReady = () => handleReload()

    const onNewData = (dat = {servers: []}) => {
        panel.setLoad(false);
        index.r();
        _render(dat?.servers);
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

        const renderRow = (server) => (row) => fields.forEach(field => {
            const val = server.hasOwnProperty(field) ? server[field] : '';
            const renderer = list[field]?.renderer;
            row.append(gui.create('span', (f) => f.append(renderer ? renderer(server) : val)));
        })
        servers.forEach(server => content.append(gui.create('div', renderRow(server))))
        panel.setContent(content);
    }

    function handleReload() {
        panel.setLoad(true);
        socket.getServerList();
    }

    function renderServerChangeEl(server) {
        const handleServerChange = (e) => {
            e.preventDefault();
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

    event.sub(SOCKET_READY, onReady);
    event.sub(GET_SERVER_LIST, onNewData);

    return {}
})(document, event, gui, log, socket);
