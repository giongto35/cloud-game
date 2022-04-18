/**
 * App UI elements module.
 *
 * @version 1
 */
const gui = (() => {

    const _create = (name = 'div', modFn) => {
        const el = document.createElement(name);
        if (modFn) {
            modFn(el);
        }
        return el;
    }

    const _option = (text = '', selected = false, label) => {
        const el = _create('option');
        if (label) {
            el.textContent = label;
            el.value = text;
        } else {
            el.textContent = text;
        }
        if (selected) el.selected = true;

        return el;
    }

    const select = (key = '', callback = function () {
    }, values = {values: [], labels: []}, current = '') => {
        const el = _create();
        const select = _create('select');
        select.onchange = event => {
            callback(key, event.target.value);
        };
        el.append(select);

        select.append(_option('none', current === ''));
        values.values.forEach((value, index) => {
            select.append(_option(value, current === value, values.labels?.[index]));
        });

        return el;
    }

    const panel = (root, title = '', cc = '', content, buttons = []) => {
        const state = {
            shown: false,
            loading: false,
            title: title,
        }

        const _root = root || _create('div');
        _root.classList.add('panel');
        const header = _create('div', (el) => el.classList.add('panel__header'));
        const _content = _create('div', (el) => {
            if (cc) {
                el.classList.add(cc);
            }
            el.classList.add('panel__content')
        });

        const _title = _create('span', (el) => {
            el.classList.add('panel__header__title');
            el.innerText = title;
        });
        header.append(_title);

        header.append(_create('div', (el) => {
            el.classList.add('panel__header__controls');

            buttons.forEach((b => el.append(_create('span', (el) => {
                el.classList.add('panel__button');
                if (b.cl) b.cl.forEach(class_ => el.classList.add(class_));
                if (b.title) el.title = b.title;
                el.innerText = b.caption;
                el.addEventListener('click', b.handler)
            }))))

            el.append(_create('span', (el) => {
                el.classList.add('panel__button');
                el.innerText = 'X';
                el.title = 'Close';
                el.addEventListener('click', () => toggle(false))
            }))
        }))

        root.append(header, _content);
        if (content) {
            _content.append(content);
        }

        const setContent = (content) => _content.replaceChildren(content)

        const setLoad = (load = true) => {
            state.loading = load;
            _title.innerText = state.loading ? `${state.title}...` : state.title;
        }

        function toggle(show) {
            state.shown = show;
            if (state.shown) {
                gui.show(_root);
            } else {
                gui.hide(_root);
            }
        }

        return {
            isHidden: () => !state.shown,
            setContent,
            setLoad,
            toggle,
        }
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
        el.classList.remove('hidden');
    }

    const hide = (el) => {
        el.classList.add('hidden');
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
                const proceed = ((val += speed) <= 1);
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

    const fragment = () => document.createDocumentFragment();

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
        binding,
        create: _create,
        fragment,
        hide,
        panel,
        select,
        show,
        toggle,
    }
})(document);
