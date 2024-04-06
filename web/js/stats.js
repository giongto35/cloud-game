import {
    sub,
    HELP_OVERLAY_TOGGLED
} from 'event';

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
     *
     * @example
     *  +-------+   +-------+   +---------+
     *  |       |   |+---+  |   |+---+    |
     *  |       |   ||||||  |   ||||||+---+
     *  |       |   ||||||  |   |||||||||||
     *  +-------+   +----+--+   +---------+
     *  []          [3]         [3, 2]
     */
    const render = () => {
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

    const clear = () => {
        data = [];
        render();
    }

    return {add, get, max, render, clear}
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

    const withPostfix = (value) => (postfix_ = value);

    const update = (value) => {
        if (_graph) _graph.add(value);
        // 203 (333) ms
        _value.textContent = `${value < 1 ? '<1' : value} ${_graph ? `(${_graph.max()}) ` : ''}${postfix_(value)}`;
    }

    const clear = () => {
        _graph && _graph.clear();
    }

    return {el: ui, update, withPostfix, clear}
}

const modules = (fn, force = true) => _modules.forEach(m => (force || m.get) && fn(m))

const module = (mod) => {
    mod = {
        val: 0,
        enable: () => ({}),
        ...mod,
        _disable: function () {
            mod.val = 0;
            mod.disable && mod.disable();
            mod.mui && mod.mui.clear();
        },
        ...(mod.mui && {
            get: () => mod.mui.el,
            render: () => mod.mui.update(mod.val)
        })
    }
    mod.init?.();
    _modules.push(mod);
    modules(m => m.get && statsOverlayEl.append(m.get()), false);
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
    modules(m => m._disable());
    _hide();
}

const _show = () => (statsOverlayEl.style.visibility = 'visible');
const _hide = () => (statsOverlayEl.style.visibility = 'hidden');

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

sub(HELP_OVERLAY_TOGGLED, onHelpOverlayToggle)

/**
 * App statistics module.
 */
export const stats = {
    toggle: () => active ? disable() : enable(),
    set modules(m) {
        m && m.forEach(mod => module(mod))
    },
    mui: moduleUi,
}
