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
    let _statsRendererId = 0;
    let tempHide = false;

    // !to add connection drop notice

    // UI
    const statsOverlayEl = document.getElementById('stats-overlay');

    /**
     *
     * @returns {{render: render}}
     */
    const graph = (options = {
        historySize: 25,
        width: 120,
        height: 20,
        topGap: 2,
        style: {
            fillColor: '#f6f6f6',
            barColor: 'red',
            leadBarColor: 'rgba(17,144,213,0.34)'
        }
    }) => {
        const _canvas = document.createElement('canvas'),
            _context = _canvas.getContext('2d');

        let i = 0;
        let data = [];

        // viewport size
        _canvas.style.height = '2em';
        _canvas.style.width = '100%';

        // scale for Retina stuff
        const scale = 1 // window.devicePixelRatio * 2;

        // internal size
        _canvas.width = options.width * scale;
        _canvas.height = options.height * scale;

        _context.scale(scale, scale);
        _context.imageSmoothingEnabled = false;
        _context.fillStyle = options.fillStyle;

        // bar size
        const barWidth = Math.round(_canvas.width / scale / options.historySize),
            barHeight = Math.round(_canvas.height / scale);
        let maxHeight = 0,
            prevMaxHeight = 0;

        const max = () => maxHeight

        const get = () => _canvas

        const add = (value) => {
            if (i > options.historySize - 1) i = 0;
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
            setFillColor(options.style.fillColor);
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

            for (let j = 0; j < stats.length; j++) {
                let x0 = j * barWidth,
                    // normalize y with maxHeight = canvas.height
                    // the range [0 + gap; canvas.height]
                    y0 = Math.round(barHeight - (barHeight * (stats[j] / maxHeight)) + options.topGap),
                    x1 = barWidth,
                    y1 = barHeight;

                // draw something if the normalized value is too low
                if (y0 + 1 >= barHeight) y0 -= 4;

                const isLeadingBar = j === index;
                if (isLeadingBar) {
                    y0 = 0;
                }

                // a really expensive color switching
                setFillColor(!isLeadingBar ? options.style.barColor : options.style.leadBarColor);
                _context.fillRect(x0, y0, x1, y1);
            }
        }

        function setFillColor(color = options.style.fillColor) {
            if (_context.fillStyle !== color) _context.fillStyle = color;
        }

        return {add, get, max, render, data}
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
     * @returns {{el: HTMLDivElement, update: function}}
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

        const update = (value, callback) => {
            if (_graph) _graph.add(value);

            if (callback) {
                callback({el: ui, label: _label, value: _value, newValue: value, graph: _graph});
                return;
            }

            _value.textContent = `${value < 1 ? '<1' : value} (${_graph.max()}) ms`;
        }

        return {el: ui, update}
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
        let previous = Date.now();
        const window = 5;

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

        const render = () => ui.update(mean);

        const get = () => ui.el;

        return {get, enable, disable, render}
    })(event, moduleUi);

    /**
     * Random numbers submodule.
     *
     *
     * ?Interface:
     *  void enable()
     *  void disable()
     *  void render()
     *
     * @version 1
     */
    const random = (() => {
        let _rendererId = 0;
        const frequencyMs = 1000;

        const ui = moduleUi('Magic', true);

        const getSome = (min, max) => Math.round(Math.random() * (max - min) + min);

        const enable = () => {
            renderItself();
            _rendererId = window.setInterval(renderItself, frequencyMs);
        }

        const disable = () => {
            if (_rendererId > 0) {
                window.clearInterval(_rendererId);
                _rendererId = 0;
            }
        }

        // dummy
        const render = () => {
        }

        const customText = (_ui) => {
            console.info(_ui.graph.data);
            _ui.value.textContent = `${_ui.newValue} (${_ui.graph.max()}) x`;
        }

        const renderItself = () => ui.update(getSome(42, 999), customText);

        const get = () => ui.el;

        return {get, enable, disable, render}
    })(event, moduleUi, window);

    // !to use requestAnimationFrame instead of intervals
    const enable = () => {
        modules.forEach(m => m.enable());
        render();
        _statsRendererId = window.setInterval(render, snapshotPeriodMSec);
        statsOverlayEl.hidden = false;
    };

    const disable = () => {
        modules.forEach(m => m.disable());
        if (_statsRendererId) {
            window.clearInterval(_statsRendererId);
            _statsRendererId = 0;
        }
        statsOverlayEl.hidden = true;
    }

    const onToggle = () => _statsRendererId ? disable() : enable();

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

    const render = () => modules.forEach(m => m.render());

    // add submodules
    modules.push(latency);
    modules.push(random);
    modules.forEach(m => statsOverlayEl.append(m.get()));

    event.sub(STATS_TOGGLE, onToggle);
    event.sub(HELP_OVERLAY_TOGGLED, onHelpOverlayToggle)

    return {enable, disable,}
})
(document, event, log, window);
