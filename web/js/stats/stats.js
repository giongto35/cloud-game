/**
 * App statistics module.
 *
 * Events:
 * <- STATS_TOGGLE
 * <- HELP_OVERLAY_TOGGLED
 *
 * @version 1
 */
const stats = (() => {
    const modules = [];
    const snapshotPeriodMSec = 200;
    let _renderer;
    let tempHide = false;

    // !to add connection drop noticing

    // UI
    const statsOverlayEl = document.getElementById('stats-overlay');

    /**
     *
     * @returns {{render: render}}
     */
    const graph = () => {
        const _canvas = document.createElement('canvas'),
            _context = _canvas.getContext('2d');

        const size = 25;
        let i = 0;
        let data = [];

        // viewport size
        _canvas.style.height = '2em';
        _canvas.style.width = '100%';

        // scale
        const scale = 1 // window.devicePixelRatio * 2;

        // internal size
        _canvas.width = 100 * scale;
        _canvas.height = 20 * scale;

        _context.scale(scale, scale);
        _context.imageSmoothingEnabled = false;
        _context.fillStyle = '#f6f6f6';

        // bar size
        const barWidth = Math.round(_canvas.width / scale / size),
            barHeight = Math.round(_canvas.height / scale);
        let maxHeight = 0,
            prevMaxHeight = 0;

        const max = () => maxHeight

        const get = () => _canvas

        const add = (value) => {
            if (i > size - 1) i = 0;
            data.splice(i, 1, value);
            render(data, i);
            i++;
        }

        // 0,0   w,0   0,0   w,0   0,0   w,0
        // +-------+   +-------+   +-------+
        // |       |   |+1-+   |   |+1-+   |
        // |       |   |||||   |   |||||+2-+
        // |       |   |||||   |   |||||||||
        // +-------+   +----+--+   +-------+
        // 0,h   w,h   0,h   w,h   0,h   w,h
        // []          [3]         [3, 2]
        //
        // O(N+N) :( can be O(1) without visual scale
        const render = (stats = [], index = 0) => {
            _context.fillRect(0, 0, _canvas.width, _canvas.height);

            // !to move outside maybe?
            maxHeight = stats[0];
            for (let i = 1; i < stats.length; i++) if (stats[i] > maxHeight) maxHeight = stats[i];

            // keep scale grow but
            // reset the max height only at the start of the new cycle
            if (index > 0) {
                if (maxHeight > prevMaxHeight) {
                    prevMaxHeight = maxHeight;
                } else {
                    maxHeight = prevMaxHeight;
                }
            } else {
                prevMaxHeight = maxHeight;
            }

            _context.fillStyle = 'red';
            const gap = 2;
            let wasLeadingBar = false;
            let barIndex = 0;
            stats.forEach(value => {
                let x0 = barIndex * barWidth,
                    // normalize y with maxHeight = canvas.height
                    // the range [0 + gap; canvas.height]
                    y0 = barHeight - (barHeight * (value / maxHeight)) + gap,
                    x1 = barWidth,
                    y1 = barHeight;

                // draw something if value is 0
                if (y0 >= barHeight) y0 -= 5;

                // whether it normal or leading bar
                if (barIndex === index) {
                    y0 = 0;
                    _context.fillStyle = 'rgba(17,144,213,0.34)';
                    wasLeadingBar = true;
                } else {
                    // because context style switching is kinda expensive
                    if (wasLeadingBar) {
                        _context.fillStyle = 'red';
                        wasLeadingBar = false;
                    }
                }

                _context.fillRect(x0, y0, x1, y1);
                barIndex++;
            });

            _context.fillStyle = '#f6f6f6';
        }

        return {
            add,
            get,
            max,
            render
        }
    }

    /**
     * Get cached module UI.
     *
     * HTML:
     * <div><div>LABEL</div><span>VALUE</span>
     *
     * Returns exposed ui sub-tree and the _value as only changing node.
     *
     * @param label
     * @param withGraph
     * @returns {{node: HTMLElement, value: HTMLElement, graph: Object}}
     */
    const moduleUi = (label = '', withGraph = false) => {
        const ui = document.createElement('div'),
            _label = document.createElement('div'),
            _value = document.createElement('span');
        ui.append(_label, _value);

        let _graph;
        if (withGraph) {
            const _container = document.createElement('span');
            _graph = graph();
            _container.append(_graph.get());
            ui.append(_container);
        }

        _label.innerHTML = label;

        return {node: ui, value: _value, graph: _graph};
    }

    function getRandomArbitrary(min, max) {
        // x -= 10;
        return Math.round(Math.random() * (max - min) + min);
    }

    /**
     * Latency stats submodule.
     *
     * Accumulates the simple rolling delta mean value
     * between a server request and a following server response values.
     *
     *      window
     *   _____________
     *  |            |
     * [1, 1, 3, 4, 1, 4, 3, 1, 2, 1, 1, 1, 2, ... n]
     *              |
     *    stats_snapshot_period
     *    mean = round(next - mean / length % window)
     *
     * Events:
     * <- PING_RESPONSE
     * <- PING_REQUEST
     *
     * ?Interface:
     *  void enable()
     *  void disable()
     *  void render()
     *
     * @version 1
     */
    const latency = (() => {
        let listeners = [];

        let mean = 0;
        let length = 0;
        let window = 5;
        let previous = Date.now();

        // UI
        const ui = moduleUi('Ping', true);

        const onPingRequest = (data) => previous = data.time;

        const onPingResponse = () => {
            length++;
            const delta = Date.now() - previous;
            mean += Math.round((delta - mean) / length);

            if (length % window === 0) {
                length = 1;
                mean = delta;
            }
        }

        const enable = () => {
            listeners.push(
                event.sub(PING_RESPONSE, onPingResponse),
                event.sub(PING_REQUEST, onPingRequest)
            );
        }

        const disable = () => {
            listeners.forEach(listener => listener.unsub())
            listeners = [];
        }

        const render = () => {
            // const v = getRandomArbitrary(50, 300);

            // const val = !Math.round(Math.random()) ? v : mean
            const val = mean;

            ui.graph.add(val);
            ui.value.innerText = `${val < 1 ? '<1' : val} (${ui.graph.max()}) ms`;

            return ui.node;
        }

        return {
            enable,
            disable,
            render,
        }
    })(event, moduleUi);

    const enable = () => {
        modules.forEach(m => m.enable());
        render();
        _renderer = window.setInterval(render, snapshotPeriodMSec);
        statsOverlayEl.hidden = false;
    };

    const disable = () => {
        modules.forEach(m => m.disable());
        if (_renderer) window.clearInterval(_renderer);
        _renderer = undefined;
        statsOverlayEl.hidden = true;
    }

    const onToggle = () => _renderer ? disable() : enable();

    /**
     * Handles help overlay toggle event.
     *
     * !to make it more declarative
     *
     * @param {Object} overlay Overlay data.
     * @param {boolean} overlay.shown A flag if the overlay is being currently showed.
     */
    const onHelpOverlayToggle = (overlay) => {
        if (!statsOverlayEl.hidden && overlay.shown && !tempHide) {
            statsOverlayEl.hidden = true;
            tempHide = true;
        } else {
            if (tempHide) {
                statsOverlayEl.hidden = false;
                tempHide = false;
            }
        }
    }

    const render = () => {
        modules.forEach(m => m.render(statsOverlayEl));
    }

    // add submodules
    modules.push(latency);
    modules
        .map(m => m.render(statsOverlayEl))
        .forEach(m => statsOverlayEl.append(m));

    event.sub(STATS_TOGGLE, onToggle);
    event.sub(HELP_OVERLAY_TOGGLED, onHelpOverlayToggle)

    return {
        enable,
        disable,
    }
})
(document, event, log, window);
