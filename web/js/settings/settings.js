/**
 * App settings module.
 *
 * So the basic idea is to let app modules request their settings
 * from an abstract store first, and if the store doesn't contain such settings yet,
 * then let the store to take default values from the module to save them before that.
 * The return value with the settings is gonna be a slice of in-memory structure
 * backed by a data provider (localStorage).
 * Doing it this way allows us to considerably simplify the code and make sure that
 * exposed settings will have the latest values without additional update/get calls.
 *
 * Uses ES8.
 *
 * @version 1
 */
const settings = (() => {
    // internal structure version
    const revision = 1;

    // default settings
    // keep them for revert to defaults option
    const _defaults = Object.create(null);
    _defaults[opts._VERSION] = revision;

    /**
     * The main store with settings passed around by reference
     * (because of that we need a wrapper object)
     * don't do this at work (it's faster to write than immutable code).
     *
     * @type {{settings: {_version: number}}}
     */
    let store = {
        settings: {
            ..._defaults
        }
    };
    let provider;

    /**
     * Enum for settings types (the explicit type of a key-value pair).
     *
     * @readonly
     * @enum {number}
     */
    const option = Object.freeze({undefined: 0, string: 1, number: 2, object: 3, list: 4});

    const exportFileName = `cloud-game.settings.v${revision}.txt`;

    // ui references
    const ui = document.getElementById('app-settings'),
        closeEl = document.getElementById('settings__controls__close'),
        loadEl = document.getElementById('settings__controls__load'),
        saveEl = document.getElementById('settings__controls__save'),
        resetEl = document.getElementById('settings__controls__reset');

    this._renderrer = this._renderrer || {
        render: () => {
        }
    };

    const getStore = () => store.settings;

    /**
     * The NullObject provider if everything else fails.
     */
    const voidProvider = (store_ = {settings: {}}) => {
        const nil = () => {
        }

        return {
            get: key => store_.settings[key],
            set: nil,
            remove: nil,
            save: nil,
            loadSettings: nil,
            reset: nil,
        }
    }

    /**
     * The LocalStorage backend for our settings (store).
     *
     * For simplicity it will rewrite all the settings on every store change.
     * If you want to roll your own, then use its "interface".
     */
    const localStorageProvider = ((store_ = {settings: {}}) => {
        if (!_isSupported()) return;

        const root = 'settings';

        const _serialize = data => JSON.stringify(data, null, 2);

        const save = () => localStorage.setItem(root, _serialize(store_.settings));

        function _isSupported() {
            const testKey = '_test_42';
            try {
                // check if it's writable and isn't full
                localStorage.setItem(testKey, testKey);
                localStorage.removeItem(testKey);
                return true;
            } catch (e) {
                log.error(e);
                return false;
            }
        }

        const get = key => JSON.parse(localStorage.getItem(key));

        const set = (key, value) => save();

        const remove = () => save();

        const loadSettings = () => {
            if (!localStorage.getItem(root)) save();
            store_.settings = JSON.parse(localStorage.getItem(root));
        }

        const reset = () => {
            localStorage.removeItem(root);
            localStorage.setItem(root, _serialize(store_.settings));
        }

        return {
            get,
            set,
            remove,
            save,
            loadSettings,
            reset,
        }
    });

    /**
     * Nuke existing settings with provided data.
     * @param text The text to extract data from.
     * @private
     */
    const _import = text => {
        try {
            for (const property of Object.getOwnPropertyNames(store.settings)) delete store.settings[property];
            Object.assign(store.settings, JSON.parse(text).settings);
            provider.save();
            event.pub(SETTINGS_CHANGED);
        } catch (e) {
            log.error(`Your import file is broken!`);
        }

        _render();
    }

    const _export = () => {
        let el = document.createElement('a');
        el.setAttribute(
            'href',
            `data:text/plain;charset=utf-8,${encodeURIComponent(JSON.stringify(store, null, 2))}`
        );
        el.setAttribute('download', exportFileName);
        el.style.display = 'none';
        document.body.appendChild(el);
        el.click();
        document.body.removeChild(el);
        el = undefined;
    }

    const init = () => {
        provider = localStorageProvider(store) || voidProvider(store);
        provider.loadSettings();

        if (revision > store.settings._version) {
            // !to handle this with migrations
            log.warning(`Your settings are in older format (v${store.settings._version})`);
        }
    }

    const get = () => store.settings;

    const _isLoaded = key => store.settings.hasOwnProperty(key);

    /**
     * Tries to load settings by some key.
     *
     * @param key A key to find values with.
     * @param default_ The default values to set if none exist.
     * @returns A slice of the settings with the given key or a copy of the value.
     */
    const loadOr = (key, default_) => {
        // preserve defaults
        _defaults[key] = default_;

        if (!_isLoaded(key)) {
            store.settings[key] = {};
            set(key, default_);
        } else {
            // !to check if settings do have new properties from default & update
            // or it have ones that defaults doesn't
        }

        return store.settings[key];
    }

    const set = (key, value, updateProvider = true) => {
        const type = _getType(value);

        // mutate settings w/o changing the reference
        switch (type) {
            case option.list:
                store.settings[key].splice(0, Infinity, ...value);
                break;
            case option.object:
                for (let option of Object.keys(value)) {
                    log.debug(`Change key [${option}] from ${store.settings[key][option]} to ${value[option]}`);
                    store.settings[key][option] = value[option];
                }
                break;
            case option.string:
            case option.number:
            case option.undefined:
            default:
                store.settings[key] = value;
        }

        if (updateProvider) {
            provider.set(key, value);
            event.pub(SETTINGS_CHANGED);
        }
    }

    const _reset = () => {
        for (let _option of Object.keys(_defaults)) {
            const value = _defaults[_option];

            // delete all sub-options not in defaults
            if (_getType(value) === option.object) {
                for (let opt of Object.keys(store.settings[_option])) {
                    const prev = store.settings[_option][opt];
                    const isDeleted = delete store.settings[_option][opt];
                    log.debug(`User option [${opt}=${prev}] has been deleted (${isDeleted}) from the [${_option}]`);
                }
            }

            set(_option, value, false);
        }

        provider.reset();
        event.pub(SETTINGS_CHANGED);
    }

    const remove = (key, subKey) => {
        const isRemoved = subKey !== undefined ? delete store.settings[key][subKey] : delete store.settings[key];
        if (!isRemoved) log.warning(`The key: ${key + (subKey ? '.' + subKey : '')} wasn't deleted!`);
        provider.remove(key, subKey);
    }

    const _render = () => settings._renderrer.render()

    /**
     * Settings modal window toggle handler.
     * @returns {boolean} True in case if it's opened.
     */
    const toggle = () => ui.classList.toggle('modal-visible') && !_render();

    function _getType(value) {
        if (value === undefined) return option.undefined
        else if (Array.isArray(value)) return option.list
        else if (typeof value === 'object' && value !== null) return option.object
        else if (typeof value === 'string') return option.string
        else if (typeof value === 'number') return option.number
        else return option.undefined;
    }

    /**
     * File reader submodule (FileReader API).
     *
     * @type {{read: read}} Tries to read a file.
     * @private
     */
    const _fileReader = (() => {
        let callback_ = () => {
        }

        const el = document.createElement('input');
        const reader = new FileReader();

        el.type = 'file';
        el.accept = '.txt';
        el.onchange = event => event.target.files.length && reader.readAsBinaryString(event.target.files[0]);
        reader.onload = event => callback_(event.target.result);

        return {
            read: callback => {
                callback_ = callback;
                el.click();
            },
        }
    })();

    const onFileLoad = text => {
        try {
            _import(text);
        } catch (e) {
            log.error(`Couldn't read your settings!`, e);
        }
    }

    event.sub(SETTINGS_CHANGED, _render);

    // internal init section
    closeEl.addEventListener('click', () => {
        event.pub(SETTINGS_CLOSED);
        // to make sure it's disabled, but it's a tad verbose
        event.pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true});
    });
    saveEl.addEventListener('click', () => _export());
    loadEl.addEventListener('click', () => _fileReader.read(onFileLoad));
    resetEl.addEventListener('click', () => {
        if (window.confirm("Are you sure want to reset your settings?")) {
            _reset();
            event.pub(SETTINGS_CHANGED);
        }
    });

    return {
        init,
        loadOr,
        getStore,
        get,
        set,
        remove,
        import: _import,
        export: _export,
        ui: {
            toggle,
        }
    }
})(document, event, JSON, localStorage, log, window);

// hardcoded ui stuff
settings._renderrer = (() => {
    // options to ignore (i.e. ignored = {'_version': 1})
    const ignored = {};

    // the main display data holder element
    const data = document.getElementById('settings-data');

    // a fast way to clear data holder.
    const clearData = () => {
        while (data.firstChild) data.removeChild(data.firstChild)
    };

    const _option = (holderEl) => {
        const wrapperEl = document.createElement('div');
        wrapperEl.classList.add('settings__option');

        const nameEl = document.createElement('div');
        nameEl.classList.add('settings__option-name');
        wrapperEl.append(nameEl);

        const valueEl = document.createElement('div');
        valueEl.classList.add('settings__option-value');
        wrapperEl.append(valueEl);

        return {
            withName: function (name = '') {
                nameEl.textContent = name;
                return this;
            },
            withClass: function (name = '') {
                wrapperEl.classList.add(name);
                return this;
            },
            readOnly: function () {
                // reserved
            },
            restartNeeded: function () {
                nameEl.classList.add('restart-needed-asterisk');
                return this;
            },
            add: function (...elements) {
                if (elements.length) for (let _el of elements.flat()) valueEl.append(_el);
                return this;
            },
            build: () => holderEl.append(wrapperEl),
        };
    }

    const onKeyChange = (key, oldValue, newValue, handler) => {

        if (newValue !== 'Escape') {
            const _settings = settings.get()[opts.INPUT_KEYBOARD_MAP];

            if (_settings[newValue] !== undefined) {
                log.warning(`There are old settings for key: ${_settings[newValue]}, won't change!`);
            } else {
                settings.remove(opts.INPUT_KEYBOARD_MAP, oldValue);
                settings.set(opts.INPUT_KEYBOARD_MAP, {[newValue]: key});
            }
        }

        // !to check leaks
        if (handler) {
            handler.unsub();
            handler = undefined;
        }

        event.pub(KEYBOARD_TOGGLE_FILTER_MODE);
        event.pub(SETTINGS_CHANGED);
    }

    const _keyChangeOverlay = (keyName, oldValue) => {
        const wrapperEl = document.createElement('div');
        wrapperEl.classList.add('settings__key-wait');
        wrapperEl.textContent = `Let's choose a ${keyName} key...`;

        let handler = event.sub(KEYBOARD_KEY_PRESSED, button => onKeyChange(keyName, oldValue, button.key, handler));

        return wrapperEl;
    }

    /**
     * Handles a normal option change.
     *
     * @param key The name (id) of an option.
     * @param newValue A new value to set.
     * @param oldValue An old value to use somehow if needed.
     */
    const onChange = (key, newValue, oldValue) => settings.set(key, newValue);

    const onKeyBindingChange = (key, oldValue) => {
        clearData();
        data.append(_keyChangeOverlay(key, oldValue));
        event.pub(KEYBOARD_TOGGLE_FILTER_MODE);
    }

    const render = function () {
        const _settings = settings.getStore();

        clearData();
        for (let k of Object.keys(_settings).sort()) {
            if (ignored[k]) continue;

            const value = _settings[k];
            switch (k) {
                case opts._VERSION:
                    _option(data).withName('Format version').add(value).build();
                    break;
                case opts.LOG_LEVEL:
                    _option(data).withName('Log level')
                        .restartNeeded()
                        .add(gui.select(k, onChange, ['trace', 'debug', 'warning', 'info'], value))
                        .build();
                    break;
                case opts.INPUT_KEYBOARD_MAP:
                    _option(data).withName('Keyboard bindings')
                        .withClass('keyboard-bindings')
                        .add(Object.keys(value).map(k => gui.binding(value[k], k, onKeyBindingChange)))
                        .build();
                    break;
                default:
                    _option(data).withName(k).add(value).build();
            }
        }
    }

    return {
        render,
    }
})(document, log, opts, settings);
