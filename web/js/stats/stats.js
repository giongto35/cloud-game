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
    const _modules = [];
    let tempHide = false;

    // internal rendering stuff
    const fps = 30;
    let time = 0;
    let active = false;

    // !to add connection drop notice

    const statsOverlayEl = document.getElementById('stats-overlay');

    /**
     * The graph element.
     */
    const graph = (parent, opts = {
        historySize: 60,
        width: 60 * 2 + 2,
        height: 20,
        pad: 4,
        scale: 1,
        style: {
            barColor: '#9bd914',
            barFallColor: '#c12604'
        }
    }) => {
        const _canvas = document.createElement('canvas');
        const _context = _canvas.getContext('2d');

        let data = [];

        _canvas.setAttribute('class', 'graph');

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
            // |       |   |+---+  |   |+---+    |
            // |       |   ||||||  |   ||||||+---+
            // |       |   ||||||  |   |||||||||||
            // +-------+   +----+--+   +---------+
            // 0,h   w,h   0,h   w,h   0,h     w,h
            // []          [3]         [3, 2]
            //

            _context.clearRect(0, 0, _canvas.width, _canvas.height);

            maxN = data[0] || 1;
            minN = 0;
            for (let k = 1; k < data.length; k++) {
                if (data[k] > maxN) maxN = data[k];
                if (data[k] < minN) minN = data[k];
            }

            for (let j = 0; j < data.length; j++) {
                let x = j * barWidth,
                    y = (barHeight - opts.pad * 2) * (data[j] - minN) / (maxN - minN) + opts.pad;

                const color = j > 0 && data[j] > data[j - 1] ? opts.style.barFallColor : opts.style.barColor;

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
     * @param label The name of the stat to show.
     * @param withGraph True if to draw a graph.
     * @param postfix Supposed to be the name of the stat passed as a function.
     * @returns {{el: HTMLDivElement, update: function}}
     */
    const moduleUi = (label = '', withGraph = false, postfix = () => 'ms') => {
        const ui = document.createElement('div'),
            _label = document.createElement('div'),
            _value = document.createElement('span');
        ui.append(_label, _value);

        let postfix_ = postfix;

        let _graph;
        if (withGraph) {
            const _container = document.createElement('span');
            ui.append(_container);
            _graph = graph(_container);
        }

        _label.innerHTML = label;

        const withPostfix = (value) => postfix_ = value;

        const update = (value) => {
            if (_graph) _graph.add(value);
            // 203 (333) ms
            _value.textContent = `${value < 1 ? '<1' : value} ${_graph ? `(${_graph.max()}) ` : ''}${postfix_(value)}`;
        }

        return {el: ui, update, withPostfix}
    }

    /**
     * Latency stats submodule.
     *
     * Accumulates the simple rolling mean value
     * between the next server request and following server response values.
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
        let previous = 0;
        const window = 5;

        const ui = moduleUi('Ping(c)', true);

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
            while (listeners.length) listeners.shift().unsub();
        }

        const render = () => ui.update(mean);

        const get = () => ui.el;

        return {get, enable, disable, render}
    })(event, moduleUi);

    /**
     * User agent memory stats.
     *
     * ?Interface:
     *  HTMLElement get()
     *  void enable()
     *  void disable()
     *  void render()
     *
     * @version 1
     */
    const clientMemory = (() => {
        let active = false;

        const measures = ['B', 'KB', 'MB', 'GB'];
        const precision = 1;
        let mLog = 0;

        const ui = moduleUi('Memory', false, (x) => (x > 0) ? measures[mLog] : '');

        const get = () => ui.el;

        const enable = () => {
            active = true;
            render();
        }

        const disable = () => active = false;

        const render = () => {
            if (!active) return;

            const m = performance.memory.usedJSHeapSize;
            let newValue = 'N/A';

            if (m > 0) {
                mLog = Math.floor(Math.log(m) / Math.log(1000));
                newValue = Math.round(m * precision / Math.pow(1000, mLog)) / precision;
            }

            ui.update(newValue);
        }

        if (window.performance && !performance.memory) performance.memory = {usedJSHeapSize: 0, totalJSHeapSize: 0};

        return {get, enable, disable, render}
    })(moduleUi, performance, window);


    const webRTCStats_ = (() => {
        let interval = null

        function getStats() {
            if (!rtcp.isConnected()) return;
            rtcp.getConnection().getStats(null).then(stats => {
                let frameStatValue = '?';
                stats.forEach(report => {
                    if (report["framesReceived"] !== undefined && report["framesDecoded"] !== undefined && report["framesDropped"] !== undefined) {
                        frameStatValue = report["framesReceived"] - report["framesDecoded"] - report["framesDropped"];
                        event.pub('STATS_WEBRTC_FRAME_STATS', frameStatValue)
                    } else if (report["framerateMean"] !== undefined) {
                        frameStatValue = Math.round(report["framerateMean"] * 100) / 100;
                        event.pub('STATS_WEBRTC_FRAME_STATS', frameStatValue)
                    }

                    if (report["nominated"] && report["currentRoundTripTime"] !== undefined) {
                        event.pub('STATS_WEBRTC_ICE_RTT', report["currentRoundTripTime"] * 1000);
                    }
                });
            });
        }

        const enable = () => {
            interval = window.setInterval(getStats, 1000);
        }

        const disable = () => window.clearInterval(interval);

        return {enable, disable, internal: true}
    })(event, rtcp, window);

    /**
     * User agent frame stats.
     *
     * ?Interface:
     *  HTMLElement get()
     *  void enable()
     *  void disable()
     *  void render()
     *
     * @version 1
     */
    const webRTCFrameStats = (() => {
        let value = 0;
        let listener;

        const label = env.getBrowser() === 'firefox' ? 'FramerateMean' : 'FrameDelay';
        const ui = moduleUi(label, false, () => '');

        const get = () => ui.el;

        const enable = () => {
            listener = event.sub('STATS_WEBRTC_FRAME_STATS', onStats);
        }

        const disable = () => {
            value = 0;
            if (listener) listener.unsub();
        }

        const render = () => ui.update(value);

        function onStats(val) {
            value = val;
        }

        return {get, enable, disable, render}
    })(moduleUi, rtcp, window);

    const webRTCRttStats = (() => {
        let value = 0;
        let listener;

        const ui = moduleUi('RTT(w)', true, () => 'ms');

        const get = () => ui.el;

        const enable = () => {
            listener = event.sub('STATS_WEBRTC_ICE_RTT', onStats);
        }

        const disable = () => {
            value = 0;
            if (listener) listener.unsub();
        }

        const render = () => ui.update(value);

        function onStats(val) {
            value = val;
        }

        return {get, enable, disable, render}
    })(moduleUi, rtcp, window);

    const modules = (fn, force = true) => {
        _modules.forEach(m => {
                if (force || !m.internal) {
                    fn(m);
                }
            }
        )
    }

    const enable = () => {
        active = true;
        modules(m => m.enable())
        render();
        draw();
        _show();
    };

    function draw(timestamp) {
        if (!active) return;

        const time_ = time + 1000 / fps;

        if (timestamp > time_) {
            time = timestamp;
            render();
        }

        requestAnimationFrame(draw);
    }

    const disable = () => {
        active = false;
        modules(m => m.disable());
        _hide();
    }

    const _show = () => statsOverlayEl.style.visibility = 'visible';
    const _hide = () => statsOverlayEl.style.visibility = 'hidden';

    const onToggle = () => active ? disable() : enable();

    /**
     * Handles help overlay toggle event.
     * Workaround for a not normal app layout layering.
     *
     * !to remove when app layering is fixed
     *
     * @param {Object} overlay Overlay data.
     * @param {boolean} overlay.shown A flag if the overlay is being currently showed.
     */
    const onHelpOverlayToggle = (overlay) => {
        if (statsOverlayEl.style.visibility === 'visible' && overlay.shown && !tempHide) {
            _hide();
            tempHide = true;
        } else {
            if (tempHide) {
                _show();
                tempHide = false;
            }
        }
    }

    const render = () => modules(m => m.render(), false);

    // add submodules
    _modules.push(
        webRTCRttStats,
        latency,
        clientMemory,
        webRTCStats_,
        webRTCFrameStats
    );
    modules(m => statsOverlayEl.append(m.get()), false);

    event.sub(STATS_TOGGLE, onToggle);
    event.sub(HELP_OVERLAY_TOGGLED, onHelpOverlayToggle)

    return {enable, disable}
})(document, env, event, log, rtcp, window);
