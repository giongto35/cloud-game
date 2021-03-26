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

    const show = (el) => {
        el.style.display = 'block';
    }

    const hide = (el) => {
        el.style.display = 'none';
    }

    const toggle = (el, what) => {
        if (what) {
            show(el)
        } else {
            hide(el)
        }
    }

    const fadeIn = async (el, speed = .1) => {
        el.style.opacity = '0';
        el.style.display = 'block';
        return new Promise((done) => (function fade() {
                let val = parseFloat(el.style.opacity);
                const proceed = ((val += 0.1) <= 1);
                if (proceed) {
                    el.style.opacity = '' + val;
                    requestAnimationFrame(fade);
                } else {
                    done();
                }
            })()
        );
    }

    const fadeOut = async (el, speed = .1) => {
        el.style.opacity = '1';
        return new Promise((done) => (function fade() {
                if ((el.style.opacity -= speed) < 0) {
                    el.style.display = "none";
                    done();
                } else {
                    requestAnimationFrame(fade);
                }
            })()
        )
    }

    const sleep = async (ms) => new Promise(resolve => setTimeout(resolve, ms));

    const fadeInOut = async (el, wait = 1000, speed = .1) => {
        await fadeIn(el, speed)
        await sleep(wait);
        await fadeOut(el, speed)
    }

    return {
        anim: {
            fadeIn,
            fadeOut,
            fadeInOut,
        },
        create: _create,
        select,
        binding,
        show,
        hide,
        toggle,
    }
})(document);
