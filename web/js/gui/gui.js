/**
 * App UI elements module.
 *
 * @version 1
 */
const gui = (() => {

    const _create = (name = 'div') => document.createElement(name);

    const _option = (text = '', selected = false) => {
        const el = _create('option');
        el.innerText = text;
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
    }, name = '') => {
        const el = _create('button');
        el.onclick = () => callback(name, 'lol');

        el.innerText = name;

        return el;
    }

    const binding = (key = '', value = '', callback = function () {
    }) => {
        const el = _create();
        el.setAttribute('class', 'binding-element');

        const k = _bind(callback, key);

        el.append(k);

        const v = _create();
        v.innerText = value;
        el.append(v);


        return el;
    }

    return {
        select,
        binding,
    }
})(document);
