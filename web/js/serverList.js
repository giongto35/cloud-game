/**
 * Server list module.
 * @version 1
 */
const serverList = (() => {
    const blockName = 'server-list',
        container = document.getElementById(blockName),
        fields = ['addr', 'id', 'is_busy'],
        field_caps = ['Address', 'ID', 'Use'];


    // container.classList.add("hidden");

    const state = {
        servers: [],
        shown: true,
    }

    const onReady = () => {
        socket.getServerList()
    }

    const handleGetServerList = (data) => {
        if (data && data['servers']) {
            state.servers = data['servers'];
            _render();
        }
    }

    function _render() {
        container.innerHTML = '';
        if (!state.shown) {
            container.classList.add('hidden');
            return;
        }

        container.classList.remove('hidden');

        if (state.servers.length > 0) {
            const h = gui.create();
            h.classList.add(`${blockName}__header`);
            container.append(h);
            field_caps.forEach(field => {
                const f = gui.create('span');
                f.innerText = field;
                h.append(f);
            })
        }
        state.servers.forEach(server => {
            const row = gui.create();
            fields.forEach(field => {
                if (server.hasOwnProperty(field)) {
                    const f = gui.create('span');
                    f.innerText = server[field];
                    row.append(f);
                }
            })
            container.append(row);
        })
    }

    event.sub(SOCKET_READY, onReady);
    event.sub(GET_SERVER_LIST, handleGetServerList);

    return {}
})(document, event, gui, log, socket);
