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
    const graph = (parent, opts = {
        historySize: 60,
        width: 60 * 2,
        height: 20,
        pad: 4,
        scale: 1,
        style: {
            barColor: 'red',
            leadBarColor: 'white'
        }
    }) => {
        const _canvas = document.createElement('canvas'),
            _context = _canvas.getContext('2d');

        let data = [];

        _canvas.setAttribute('class', 'graph');
        // _canvas.style.width = '' + (opts.width / 2);
        // _canvas.style.height = '' + opts.height;

        // internal size
        _canvas.width = opts.width * opts.scale;
        _canvas.height = opts.height * opts.scale;

        _context.scale(opts.scale, opts.scale);
        _context.imageSmoothingEnabled = false;
        _context.fillStyle = opts.fillStyle;

        if (parent) parent.append(_canvas);

        // bar size
        const barWidth = Math.round(_canvas.width / opts.scale / opts.historySize);
        const barHeight = Math.round(_canvas.height / opts.scale);

        let maxN = 0,
            minN = 0;

        const max = () => maxN

        const get = () => _canvas

        const add = (value) => {
            if (data.length > opts.historySize) data.shift();
            data.push(value);
            render();
        }

        /**
         *  Draws a bar graph on the canvas.
         */
        const render = () => {

            // 0,0   w,0   0,0   w,0   0,0     w,0
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

            maxN = data[0];
            minN = 0;
            for (let k = 1; k < data.length; k++) {
                if (data[k] > maxN) maxN = data[k];
                if (data[k] < minN) minN = data[k];
            }

            for (let j = 0; j < data.length; j++) {
                let x = j * barWidth,
                    y = (barHeight - opts.pad * 2) * (data[j] - minN) / (maxN - minN) + opts.pad;

                const color = data[j - 1] !== undefined &&
                data[j] > data[j - 1] ? '#c12604' : '#9bd914';

                drawRect(x, barHeight - Math.round(y), barWidth, barHeight, color);
            }
        }

        const drawRect = (x, y, w, h, color = opts.style.barColor) => {
            _context.fillStyle = color;
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
            ui.append(_container);
            _graph = graph(_container);
        }

        _label.innerHTML = label;

        const update = (value) => {
            if (_graph) _graph.add(value);
            // 203 (333) ms
            _value.textContent = `${value < 1 ? '<1' : value} ${_graph ? `(${_graph.max()}) ` : ''}${postfix}`;
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
        statsOverlayEl.style.visibility = 'visible';
    };

    const disable = () => {
        modules.forEach(m => m.disable());
        if (_statsRendererId) {
            window.clearInterval(_statsRendererId);
            _statsRendererId = 0;
        }
        statsOverlayEl.style.visibility = 'hidden';
    }

    const onToggle = () => _statsRendererId ? disable() : enable();

    /**
     * Handles help overlay toggle event.
     * Workaround for a not normal app layout layering.
     *
     * !to make it more declarative
     * !to remove when app layering is fixed
     *
     * @param {Object} overlay Overlay data.
     * @param {boolean} overlay.shown A flag if the overlay is being currently showed.
     */
    const onHelpOverlayToggle = (overlay) => {
        if (statsOverlayEl.style.visibility !== 'hidden' && overlay.shown && !tempHide) {
            statsOverlayEl.style.visibility = 'hidden';
            tempHide = true;
        } else {
            if (tempHide) {
                statsOverlayEl.style.visibility = 'visible';
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
