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

    /**
     * The NullObject provider if everything else fails.
     */
    const voidProvider = (store_ = {settings: {}}) => {
        const nil = () => {
        }

        return {
            get: key => store_.settings[key],
            set: nil,
            loadSettings: nil,
            isSupported: true,
        }
    }

    /**
     * The LocalStorage backend for our settings (store).
     *
     * For simplicity it will rewrite all the settings on every store change.
     * If you want to roll your own, then use its "interface".
     */
    const localStorageProvider = ((store_ = {settings: {}}) => {
        const root = 'settings';
        const isSupported = _isSupported();

        const _serialize = data => JSON.stringify(data, null, 2);

        const _save = () => localStorage.setItem(root, _serialize(store_.settings));

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

        const set = (key, value) => _save();

        const loadSettings = () => {
            if (!localStorage.getItem(root)) _save();
            store_.settings = JSON.parse(localStorage.getItem(root));
        }

        return {
            get,
            set,
            loadSettings,
            isSupported,
        }
    });

    const _import = (data = {}) => {
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
        provider = localStorageProvider(store);
        if (!provider.isSupported) provider = voidProvider(store);

        provider.loadSettings();

        if (revision > store.settings._version) {
            // !to handle this as migrations
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
        // replace or set object's values directly
        // instead of changing the whole reference
        // that way we can access new values right away
        if (typeof value === 'object' && value !== null) {
            for (let k in Object.keys(value)) {
                const old = store.settings[key][k];
                store.settings[key][k] = value[k];
                log.debug(`${k} was set from ${old} to ${value[k]}`)
            }
        } else {
            store.settings[key] = value;
        }
        // !to add arrays

        provider.set(key, value);
        event.pub(SETTINGS_CHANGED);
    }

    return {
        init,
        loadOr,
        get,
        set,
        import: _import,
        export: _export
    }
})(event, JSON, localStorage, log);
