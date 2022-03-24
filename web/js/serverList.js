/**
 * Server list module.
 * @version 1
 */
const serverList = (() => {
    const id = 'server-list',
        root = document.getElementById(id),
        index = ((i = 1) => () => i++)(),
        // caption -- the field caption
        // renderer -- is an arbitrary mutation of the field
        list = {
            'n': {
                renderer: () => String(index()).padStart(2, '0')
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
                renderer: (data) => data.is_busy === true ? 'X' : ''
            },
            'use': {
                renderer: renderServerChangeEl
            }
        },
        fields = Object.keys(list);

    // root.classList.add("hidden");

    const state = {
        shown: true,
    }

    // waiting for the socket to request data
    const onReady = () => socket.getServerList()

    const onNewData = (dat = {servers: []}) => _render(dat?.servers)

    function _render(servers = []) {
        if (!state.shown) {
            gui.hide(root);
            return;
        }
        root.innerHTML = '';
        gui.show(root);

        if (servers.length === 0) {
            root.append(gui.create('span', (el) => el.innerText = 'No data :('));
            return;
        }

        const frag = gui.fragment();
        const header = gui.create('div', (el) => {
            el.classList.add(`${id}__header`);
            fields.forEach(field => el.append(gui.create('span', (f) => f.innerHTML = list[field]?.caption || '')))
        });
        frag.append(header)

        const renderRow = (server) => (row) => fields.forEach(field => {
            const val = server.hasOwnProperty(field) ? server[field] : '';
            const renderer = list[field]?.renderer;
            row.append(gui.create('span', (f) => f.append(renderer ? renderer(server) : val)));
        })
        servers.forEach(server => frag.append(gui.create('div', renderRow(server))))
        root.append(frag);
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

    event.sub(SOCKET_READY, onReady);
    event.sub(GET_SERVER_LIST, onNewData);

    return {}
})(document, event, gui, log, socket);
