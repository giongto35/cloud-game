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
    const snapshotPeriodSec = 1;
    let _renderer;
    let tempHide = false;

    // UI
    const statsOverlayEl = document.getElementById('stats-overlay');

    /**
     * Get cached module UI.
     *
     * HTML:
     * <div><div>LABEL</div><span>VALUE</span>
     *
     * Return exposed ui sub-tree and the _value as only changing node.
     */
    const moduleUi = (label = '') => {
        const ui = document.createElement('div'),
            _label = document.createElement('div'),
            _value = document.createElement('span');
        ui.append(_label, _value);

        _label.innerHTML = label;

        return {node: ui, value: _value};
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
        const ui = moduleUi('Ping');

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
            ui.value.innerText = `${mean < 1 ? '<1' : mean} ms`;

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
        _renderer = window.setInterval(() => {
            render();
        }, snapshotPeriodSec * 1000);
        statsOverlayEl.hidden = false;
    };

    const disable = () => {
        modules.forEach(m => m.disable());
        if (_renderer) window.clearInterval(_renderer);
        _renderer = undefined;
        statsOverlayEl.hidden = true;
    }

    const onToggle = () => {
        if (_renderer) {
            disable();
        } else {
            enable();
        }
    }

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
})(document, event, log, window);
