/**
 * Server list module.
 * @version 1
 */
const serverList = (() => {
    const id = 'server-list',
        root = document.getElementById(id),
        // cap -- is a caption of the field
        // mut -- ia a transformation function for the field value
        list = {
            'n': {},
            'id': {cap: 'ID'},
            'addr': {
                cap: 'Address',
                mut: (v) => {
                    try {
                        return new URL(v).host
                    } catch (_) {
                        return v
                    }
                }
            },
            'is_busy': {cap: 'Reserved', mut: (v) => v === true ? '◍' : ''}
        },
        fields = Object.keys(list);

    // root.classList.add("hidden");

    const state = {
        servers: [],
        shown: true,
    }

    // waiting for the server connection when it's ready
    const onReady = () => socket.getServerList()

    const handleGetServerList = (data) => {
        state.servers = data?.servers ? data.servers : [];
        _render();
    }

    const _index = (i = 0) => () => i++

    function _render() {
        root.innerHTML = '';
        if (!state.shown) {
            gui.hide(root);
            return;
        }
        gui.show(root);

        if (state.servers.length === 0) {
            root.append(gui.create('span', (el) => el.innerText = 'No data :('));
            return;
        }

        const header = gui.create('div', (el) => {
            el.classList.add(`${id}__header`);
            fields.forEach(field => el.append(gui.create('span', (f) => f.innerText = list[field]?.cap || '')))
        });
        root.append(header);

        const renderRow = (server, i) => (row) => fields.forEach(field => {
            const val = server.hasOwnProperty(field) ? server[field] :
                // do row index
                field === 'n' ? i
                    : '';
            const mut = list[field]?.mut;
            row.append(gui.create('span', (f) => f.innerText = mut ? mut(val) : val));
        })
        const index = _index(1);
        state.servers.forEach(server => root.append(gui.create('div', renderRow(server, index()))))
    }

    event.sub(SOCKET_READY, onReady);
    event.sub(GET_SERVER_LIST, handleGetServerList);

    return {}
})(document, event, gui, log, socket);
