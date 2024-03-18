/**
 * App UI elements module.
 */

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

const select = (key = '', callback = () => ({}), values = {values: [], labels: []}, current = '') => {
    const el = _create();
    const select = _create('select');
    select.onchange = event => {
        callback(key, event.target.value);
    };
    el.append(select);

    select.append(_option(0, current === '', 'none'));
    values.values.forEach((value, index) => {
        select.append(_option(value, current === value, values.labels?.[index]));
    });

    return el;
}

const checkbox = (id, cb = () => ({}), checked = false, label = '', cc = '') => {
    const el = _create();
    cc !== '' && el.classList.add(cc);

    let parent = el;

    if (label) {
        const _label = _create('label', (el) => {
            el.setAttribute('htmlFor', id);
        })
        _label.innerText = label;
        el.append(_label)
        parent = _label;
    }

    const input = _create('input', (el) => {
        el.setAttribute('id', id);
        el.setAttribute('name', id);
        el.setAttribute('type', 'checkbox');
        el.onclick = ((e) => {
            checked = e.target.checked
            cb(id, checked)
        })
        checked && el.setAttribute('checked', '');
    });
    parent.prepend(input);

    return el;
}

const panel = (root, title = '', cc = '', content, buttons = [], onToggle) => {
    const state = {
        shown: false,
        loading: false,
        title: title,
    }

    const tHandlers = [];
    onToggle && tHandlers.push(onToggle);

    const _root = root || _create('div');
    _root.classList.add('panel');
    gui.hide(_root);

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
            if (Object.keys(b).length === 0) {
                el.classList.add('panel__button_separator');
                return
            }
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

    const toggle = (() => {
        let br = window.getComputedStyle(_root.parentElement).borderRadius;
        return (force) => {
            state.shown = force !== undefined ? force : !state.shown;
            // hack for not transparent jpeg corners :_;
            _root.parentElement.style.borderRadius = state.shown ? '0px' : br;
            tHandlers.forEach(h => h?.(state.shown, _root));
            state.shown ? gui.show(_root) : gui.hide(_root)
        }
    })()

    return {
        contentEl: _content,
        isHidden: () => !state.shown,
        onToggle: (fn) => tHandlers.push(fn),
        setContent,
        setLoad,
        toggle,
    }
}

const _bind = (cb = () => ({}), name = '', oldValue) => {
    const el = _create('button');
    el.onclick = () => cb(name, oldValue);
    el.textContent = name;
    return el;
}

const binding = (key = '', value = '', cb = () => ({})) => {
    const el = _create();
    el.setAttribute('class', 'binding-element');

    const k = _bind(cb, key, value);

    el.append(k);

    const v = _create();
    v.textContent = value;
    el.append(v);

    return el;
}

const show = (el) => {
    el.classList.remove('hidden');
}

const inputN = (key = '', cb = () => ({}), current = 0) => {
    const el = _create();
    const input = _create('input');
    input.type = 'number';
    input.value = current;
    input.onchange = event => cb(key, event.target.value);
    el.append(input);
    return el;
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

export const gui = {
    anim: {
        fadeIn,
        fadeOut,
        fadeInOut,
    },
    binding,
    checkbox,
    create: _create,
    fragment,
    hide,
    inputN,
    panel,
    select,
    show,
    toggle,
}
