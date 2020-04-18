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
        pad: 4,
        style: {
            barColor: 'red',
            leadBarColor: 'white'
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

        /**
         *  Draws a bar graph on the canvas.
         *
         * @param stats A list of values to graph.
         * @param index The index of the last updated value in the list.
         */
        const render = (stats = [], index = 0) => {

            // 0,0   w,0   0,0   w,0   0,0   w,0
            // +-------+   +-------+   +---------+
            // |       |   |+-1-+  |   |+-1-+    |
            // |       |   ||||||  |   ||||||+-2-+
            // |       |   ||||||  |   |||||||||||
            // +-------+   +----+--+   +---------+
            // 0,h   w,h   0,h   w,h   0,h     w,h
            // []          [3]         [3, 2]
            //
            // O(N+N) :( can be O(1) without visual scale

            _context.clearRect(0, 0, _canvas.width, _canvas.height);

            maxHeight = stats[0];
            let minHeight = 0;
            for (let k = 1; k < stats.length; k++) {
                if (stats[k] > maxHeight) maxHeight = stats[k];
                if (stats[k] < minHeight) minHeight = stats[k];
            }

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
                let x = j * barWidth,
                    y = (barHeight - options.pad * 2) * (stats[j] - minHeight) / (maxHeight - minHeight) + options.pad;

                drawRect(x, barHeight - y, barWidth, barHeight);

                // draw bar pointer
                if (j === index) {
                    drawRect(x, barHeight - 1, barWidth, barHeight, options.style.leadBarColor);
                }
            }
        }

        const drawRect = (x, y, w, h, color = options.style.barColor) => {
            if (_context.fillStyle !== color) _context.fillStyle = color;
            _context.fillRect(x, y, w, h);
        }

        return {add, get, max, render}
    }

    /**
     * Get cached module UI.
     *
     * HTML:
     * <div><div>LABEL</div><span>VALUE</span>[<span><canvas/><span>]</div>
     *
     * Returns exposed ui sub-tree and the _value as only changing node.
     *
     * @param label The name of the stat to show.
     * @param withGraph True if to draw a graph.
     * @param postfix The name of dimension of the stat.
     * @returns {{el: HTMLDivElement, update: function}}
     */
    const moduleUi = (label = '', withGraph = false, postfix = 'ms') => {
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

        const update = (value) => {
            if (_graph) _graph.add(value);
            _value.textContent = `${value < 1 ? '<1' : value} (${_graph.max()}) ${postfix}`;
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
     *  HTMLElement get()
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
     * Renders itself without external calls.
     *
     * ?Interface:
     *  HTMLElement get()
     *  void enable()
     *  void disable()
     *  void render()
     *
     * @version 1
     */
    const random = (() => {
        let _rendererId = 0;
        const frequencyMs = 1000;

        const ui = moduleUi('Magic', true, 'x');

        const getSome = (min, max) => Math.round(Math.random() * (max - min) + min);

        const enable = () => {
            _render();
            _rendererId = window.setInterval(_render, frequencyMs);
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

        const _render = () => ui.update(getSome(42, 999));

        const get = () => ui.el;

        return {get, enable, disable, render}
    })(moduleUi, window);

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

    return {enable, disable}
})(document, event, log, window);
