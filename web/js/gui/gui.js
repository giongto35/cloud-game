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

    const panel = (root, title = '', cc = '', content) => {
        const _root = root || _create('div');
        _root.classList.add('panel');
        const header = _create('div', (el) => el.classList.add('panel__header'));
        const _content = _create('div', (el) => {
            if (cc) {
                el.classList.add(cc);
            }
            el.classList.add('panel__content')
        });

        header.append(_create('span', (el) => {
            el.classList.add('panel__header__title');
            el.innerText = title;
        }));
        header.append(_create('div', (el) => {
            el.innerHTML = "<div style=\"color: rgba(0, 0, 0, 0.7);\"><div style=\"background-color: rgba(0, 0, 0, 0.3); border-radius: 9999px; height: 16px; width: 16px;\"></div></div>";
            el.addEventListener('click', () => {
                root.style.display = 'none';
                //document.addEventListener('keydown', (e) => {
                //   if (e.key === 'Escape') {

                // }
                //})
            })
        }))

        root.append(header, _content);
        if (content) {
            _content.append(content);
        }


        return root;
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
