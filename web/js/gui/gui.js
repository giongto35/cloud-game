/**
 * App UI elements module.
 *
 * @version 1
 */
const gui = (() => {

    const _create = (name = 'div') => document.createElement(name);

    const _option = (text = '', selected = false) => {
        const el = _create('option');
        el.textContent = text;
        if (selected) el.selected = true;

        return el;
    }

    const select = (key = '', callback = function () {
    }, values = [], current = '') => {
        const el = _create();
        const select = _create('select');
        select.onchange = event => {
            callback(key, event.target.value);
        };
        el.append(select);

        select.append(_option('none', current === ''));
        for (let value of values) select.append(_option(value, current === value));

        return el;
    }

    const _bind = (callback = function () {
    }, name = '', oldValue) => {
        const el = _create('button');
        el.onclick = () => callback(name, oldValue);

        el.textContent = name;

        return el;
    }

    const binding = (key = '', value = '', callback = function () {
    }) => {
        const el = _create();
        el.setAttribute('class', 'binding-element');

        const k = _bind(callback, key, value);

        el.append(k);

        const v = _create();
        v.textContent = value;
        el.append(v);

        return el;
    }

    return {
        select,
        binding,
    }
})(document);
