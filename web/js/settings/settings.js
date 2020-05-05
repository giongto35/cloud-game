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
 * Uses ES6.
 *
 * @version 1
 */
const settings = (() => {
    const revision = 1;

    // the main store with settings passed around by reference
    // (because of that we need a wrapper object)
    // don't do this at work (it's faster to write than immutable code)
    let store = {
        settings: {
            _version: revision
        }
    };
    let provider;

    const exportFileName = `cg.settings.v${revision}.txt`;

    const ui = document.getElementById('settings');
    const close = document.getElementById('modal-close');
    const data = document.getElementById('settings-data');

    /**
     * The NullObject provider if everything else fails.
     */
    const voidProvider = (store_ = {settings: {}}) => {
        const nil = () => {
        }

        return {
            get: key => store_.settings[key],
            set: nil,
            save: nil,
            loadSettings: nil,
        }
    }

    /**
     * The LocalStorage backend for our settings (store).
     *
     * For simplicity it will rewrite all the settings on every store change.
     * If you want to roll your own, then use its "interface".
     */
    const localStorageProvider = ((store_ = {settings: {}}) => {
        if (!_isSupported()) return undefined;

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
                return false;
            }
        }

        const get = key => JSON.parse(localStorage.getItem(key));

        const set = (key, value) => save();

        const loadSettings = () => {
            if (!localStorage.getItem(root)) save();
            store_.settings = JSON.parse(localStorage.getItem(root));
        }

        return {
            get,
            set,
            save,
            loadSettings,
        }
    });

    /**
     * Nuke existing settings with provided data.
     * @param text The text to extract data from.
     * @private
     */
    const _import = text => {
        if (!text) return;

        try {
            for (const property of Object.getOwnPropertyNames(store.settings)) delete store.settings[property];
            Object.assign(store.settings, JSON.parse(text).settings);
            provider.save();
            event.pub(SETTINGS_CHANGED);
        } catch (e) {
            log.error(`Your import file is broken!`);
        }
    }

    const _export = () => {
        let el = document.createElement('a');
        el.setAttribute(
            'href',
            `data:text/plain;charset=utf-8,${encodeURIComponent(JSON.stringify(store))}`
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

    /**
     * Tries to load settings by some key.
     *
     * @param key A key to find values with.
     * @param default_ The default values to set if none exist.
     * @returns A slice of the settings with the given key.
     */
    const loadOr = (key, default_) => {
        if (!store.settings.hasOwnProperty(key)) {
            store.settings[key] = {};
            set(key, default_);
        } else {
            // !to check if settings doesn't have new properties from default & update
            // or it have one which defaults doesn't have
        }

        return store.settings[key];
    }

    const set = (key, value) => {
        // mutate existing settings
        // without changing the reference
        if (Array.isArray(value)) {
            store.settings[key].splice(0, Infinity, ...value);
        } else if (typeof value === 'object' && value !== null) {
            for (const k of Object.keys(value)) {
                log.debug(`Change ${k}: ${store.settings[key][k]} -> ${value[k]}`);
                store.settings[key][k] = value[k];
            }
        } else {
            store.settings[key] = value;
        }

        provider.set(key, value);
        event.pub(SETTINGS_CHANGED);
    }

    // oh, wow!
    const _render = () => {
        const els = [];
        Object.keys(store.settings).forEach(k => {
            const value = store.settings[k];

            if (typeof value === 'object' && value !== null) {
                els.push(`<div>${k} → ...</div>`);

                els.push('<div>');
                Object.keys(value).forEach(kk => {
                    els.push(`<div>${kk} → ${value[kk]}</div>`);
                })
                els.push('</div>')
            } else
                els.push(`<div>${k} → ${store.settings[k]}</div>`);
        })

        data.innerHTML = '';
        data.innerHTML = els.join('');
    }

    const toggle = () => {
        const what = ui.classList.toggle('modal-visible');

        if (what) {
            _render();
        }

        return what;
    }

    // init
    close.addEventListener('click', () => {
        toggle();
    })

    return {
        init,
        loadOr,
        get,
        set,
        import: _import,
        export: _export,
        ui: {
            toggle,
        }
    }
})(document, event, JSON, localStorage, log);
