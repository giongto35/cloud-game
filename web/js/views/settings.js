import {
    pub,
    sub,
    GAMEPAD_BUTTON_CAPTURED,
    GAMEPAD_CONNECTED,
    GAMEPAD_DISCONNECTED,
    KEYBOARD_KEY_PRESSED,
    KEYBOARD_TOGGLE_FILTER_MODE,
} from 'event';
import {BINDABLE, KEY, defaultGamepadMap, defaultKeyboardMap, joystick, keyboard} from 'input';
import {log} from 'log';
import {opts, settings} from 'settings';
import {http} from '../http.js?v=4';
import {icon} from '../ui/icons.js?v=4';
import {avatar, bumpAvatar, esc, on} from '../ui/dom.js?v=4';
import {confirmDialog, modal} from '../ui/modal.js?v=4';
import {toast} from '../ui/toast.js?v=4';
import {buttons, keyName, padButtonName, roles} from '../i18n.js?v=4';

// game keys that make sense on a gamepad
const PAD_BINDABLE = [KEY.UP, KEY.DOWN, KEY.LEFT, KEY.RIGHT, KEY.A, KEY.B, KEY.X, KEY.Y,
    KEY.L, KEY.R, KEY.L2, KEY.R2, KEY.L3, KEY.R3, KEY.START, KEY.SELECT, KEY.MENU];

let ctx = {me: null, onLogout: () => ({}), onNameChanged: () => ({})};
let current = null; // open modal
let tab = 'controls';
let capture = null; // {kind: 'kb'|'pad', key, sub}

/** Crops the picture to a centered square and scales it down in the browser (smaller upload). */
const squareBlob = (file, side) => new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
        const s = Math.min(img.naturalWidth, img.naturalHeight);
        const c = document.createElement('canvas');
        c.width = c.height = Math.min(side, s);
        c.getContext('2d').drawImage(img, (img.naturalWidth - s) / 2, (img.naturalHeight - s) / 2, s, s, 0, 0, c.width, c.height);
        URL.revokeObjectURL(url);
        c.toBlob((b) => (b ? resolve(b) : reject(new Error('Não foi possível processar a imagem.'))), 'image/png');
    };
    img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('Esse arquivo não é uma imagem válida.'));
    };
    img.src = url;
});

const codesFor = (map, key) => Object.keys(map).filter((c) => map[c] === key);

const switchRow = (id, title, desc, checked) => `
    <div class="setting"><div class="label"><b>${title}</b><small>${desc}</small></div>
    <div class="control"><label class="switch"><input type="checkbox" data-opt="${id}" ${checked ? 'checked' : ''}><span></span></label></div></div>`;

const rangeRow = (id, title, desc, value, min, max, step, fmt) => `
    <div class="setting"><div class="label"><b>${title}</b><small>${desc}</small></div>
    <div class="control"><input type="range" data-opt="${id}" min="${min}" max="${max}" step="${step}" value="${value}" style="width:160px">
    <span class="badge" data-out="${id}" style="min-width:52px;justify-content:center">${fmt(value)}</span></div></div>`;

const selectRow = (id, title, desc, value, options) => `
    <div class="setting"><div class="label"><b>${title}</b><small>${desc}</small></div>
    <div class="control"><select class="input" data-opt="${id}" style="width:200px">
    ${options.map(([v, l]) => `<option value="${esc(v)}" ${String(v) === String(value) ? 'selected' : ''}>${esc(l)}</option>`).join('')}
    </select></div></div>`;

const formats = {
    [opts.VOLUME]: (v) => `${v}%`,
    [opts.INPUT_DEADZONE]: (v) => `${Math.round(v * 100)}%`,
    [opts.TOUCH_OPACITY]: (v) => `${Math.round(v * 100)}%`,
    [opts.TOUCH_SCALE]: (v) => `${Math.round(v * 100)}%`,
};

// Panes

const paneControls = () => {
    const kb = keyboard.map;
    const pad = joystick.map;
    const active = joystick.active();
    const type = active?.type || 'generic';
    const kbRows = BINDABLE.map((k) => {
        const codes = codesFor(kb, k);
        const busy = capture?.kind === 'kb' && capture.key === k;
        return `<div class="binding ${busy ? 'capturing' : ''}"><span>${esc(buttons[k] || k)}</span>
            <button class="btn sm outline key-value" data-bind-kb="${k}">${busy ? 'Pressione…' : codes.length ? codes.map((c) => esc(keyName(c))).join(' / ') : '—'}</button></div>`;
    }).join('');
    const padRows = PAD_BINDABLE.map((k) => {
        const idx = codesFor(pad, k);
        const busy = capture?.kind === 'pad' && capture.key === k;
        return `<div class="binding ${busy ? 'capturing' : ''}"><span>${esc(buttons[k] || k)}</span>
            <button class="btn sm outline key-value" data-bind-pad="${k}">${busy ? 'Pressione…' : idx.length ? idx.map((i) => esc(padButtonName(type, +i))).join(' / ') : '—'}</button></div>`;
    }).join('');

    return `
        <div class="pane-title"><h3>${icon('gamepad')} Controle</h3>
            <button class="btn ghost sm" data-reset="pad">${icon('rotate')}Restaurar padrão</button></div>
        <div class="gamepad-status ${active ? 'connected' : ''}">${icon('gamepad', 'lg')}
            <span>${active ? `<b>Controle conectado</b> · ${esc(active.id.replace(/\(.*?\)/g, '').trim() || 'Controle')}${active.standard ? '' : ' · <span class="badge warning">mapeamento não padrão</span>'}`
            : 'Nenhum controle detectado. Conecte um controle e aperte qualquer botão.'}</span></div>
        <div class="binding-grid">${padRows}</div>
        ${rangeRow(opts.INPUT_DEADZONE, 'Zona morta do analógico', 'Ignora pequenos movimentos do analógico.', settings.get(opts.INPUT_DEADZONE) ?? 0.15, 0.05, 0.4, 0.01, formats[opts.INPUT_DEADZONE])}
        ${switchRow(opts.INPUT_DPAD_MODE, 'Analógico esquerdo como direcional', 'Desative para jogos que usam o analógico (ex.: Nintendo 64).', settings.get(opts.INPUT_DPAD_MODE) ?? true)}

        <div class="pane-title"><h3>${icon('keyboard')} Teclado</h3>
            <button class="btn ghost sm" data-reset="kb">${icon('rotate')}Restaurar padrão</button></div>
        <div class="binding-grid">${kbRows}</div>

        <div class="pane-title"><h3>${icon('hand')} Controles na tela</h3></div>
        ${selectRow(opts.TOUCH_MODE, 'Exibir controles de toque', 'Automático mostra em celulares e tablets.', settings.get(opts.TOUCH_MODE) ?? 'auto',
            [['auto', 'Automático'], ['on', 'Sempre'], ['off', 'Nunca']])}
        ${rangeRow(opts.TOUCH_OPACITY, 'Opacidade', 'Transparência dos botões na tela.', settings.get(opts.TOUCH_OPACITY) ?? 0.55, 0.2, 1, 0.05, formats[opts.TOUCH_OPACITY])}
        ${rangeRow(opts.TOUCH_SCALE, 'Tamanho', 'Tamanho dos botões na tela.', settings.get(opts.TOUCH_SCALE) ?? 1, 0.7, 1.4, 0.05, formats[opts.TOUCH_SCALE])}`;
};

const paneVideo = () => `
    <div class="pane-title"><h3>${icon('monitor')} Vídeo</h3></div>
    ${selectRow(opts.VIDEO_PIXEL, 'Estilo da imagem', 'Pixelado deixa os jogos retrô mais nítidos.', settings.get(opts.VIDEO_PIXEL) ? 'pixel' : 'smooth',
        [['smooth', 'Suave'], ['pixel', 'Pixelado (nítido)']])}
    ${switchRow(opts.FORCE_FULLSCREEN, 'Tela cheia ao iniciar', 'Entra em tela cheia quando o jogo começa (exceto celulares).', settings.get(opts.FORCE_FULLSCREEN))}
    ${switchRow(opts.SHOW_STATS, 'Mostrar estatísticas', 'Exibe latência, quadros e taxa de dados durante o jogo (F8).', settings.get(opts.SHOW_STATS))}
    <div class="pane-title"><h3>${icon('volume')} Áudio</h3></div>
    ${rangeRow(opts.VOLUME, 'Volume', 'Volume do jogo.', settings.get(opts.VOLUME) ?? 50, 0, 100, 1, formats[opts.VOLUME])}`;

const paneAccount = () => {
    const me = ctx.me || {};
    return `
    <div class="pane-title"><h3>${icon('user')} Perfil</h3></div>
    <div class="setting avatar-setting">
        <div data-avatar-preview>${avatar(me.name || '?', 'xl', me.id)}</div>
        <div class="label"><b>Foto de perfil</b><small>Aparece na barra, nas salas e nos torneios. PNG, JPEG ou GIF; ela é cortada em quadrado.</small></div>
        <div class="control">
            <input type="file" accept="image/png,image/jpeg,image/gif" data-avatar-file hidden>
            <button class="btn outline" data-avatar-pick>${icon('upload')}Enviar foto</button>
            <button class="btn ghost" data-avatar-remove>${icon('trash')}Remover</button>
        </div>
    </div>
    <form class="setting" data-form="name" style="align-items:flex-end">
        <div class="field" style="flex:1"><label for="s-name">Nome de exibição</label>
        <input class="input" id="s-name" maxlength="40" value="${esc(me.name || '')}" required></div>
        <button class="btn primary" type="submit">Salvar</button>
    </form>
    <div class="setting"><div class="label"><b>E-mail</b><small>${esc(me.email || '')}</small></div>
        <div class="control"><span class="badge accent">${esc(roles[me.role] || me.role || '')}</span></div></div>
    <div class="pane-title"><h3>${icon('lock')} Alterar senha</h3></div>
    <form class="list" data-form="password">
        <div class="field"><label for="s-cur">Senha atual</label><input class="input" id="s-cur" type="password" autocomplete="current-password" required></div>
        <div class="field"><label for="s-new">Nova senha</label><input class="input" id="s-new" type="password" autocomplete="new-password" minlength="8" required placeholder="Mínimo de 8 caracteres"></div>
        <div class="field"><label for="s-conf">Confirme a nova senha</label><input class="input" id="s-conf" type="password" autocomplete="new-password" minlength="8" required></div>
        <div><button class="btn primary" type="submit">${icon('key')}Alterar senha</button></div>
        <small class="subtle">Ao alterar a senha, as suas outras sessões são encerradas.</small>
    </form>
    <div class="pane-title"><h3>${icon('logout')} Sessão</h3></div>
    <button class="btn danger" data-logout>${icon('logout')}Sair da conta</button>`;
};

const paneAdvanced = () => `
    <div class="pane-title"><h3>${icon('zap')} Avançado</h3></div>
    ${switchRow(opts.MIRROR_SCREEN, 'Renderizar em canvas', 'Desativa a suavização do navegador desenhando o vídeo num canvas. Exige mais do computador.', settings.get(opts.MIRROR_SCREEN) === 'mirror')}
    ${switchRow(opts.WEBRTC_INIT_OFFER, 'WebRTC: aguardar oferta do servidor', 'Mude apenas se tiver problemas de conexão. Vale após recarregar a página.', settings.get(opts.WEBRTC_INIT_OFFER))}
    ${selectRow(opts.LOG_LEVEL, 'Nível de registro (console)', 'Usado para diagnóstico de problemas.', settings.get(opts.LOG_LEVEL),
        [[log.ERROR, 'Erros'], [log.WARN, 'Avisos'], [log.INFO, 'Informações'], [log.DEBUG, 'Depuração'], [log.TRACE, 'Detalhado']])}
    <div class="pane-title"><h3>${icon('rotate')} Restaurar</h3></div>
    <button class="btn danger" data-reset-all>${icon('trash')}Restaurar todas as configurações</button>`;

const panes = {
    controls: ['Controles', 'gamepad', paneControls],
    video: ['Vídeo e áudio', 'monitor', paneVideo],
    account: ['Conta', 'user', paneAccount],
    advanced: ['Avançado', 'zap', paneAdvanced],
};

const render = () => {
    if (!current) return;
    const pane = current.el.querySelector('[data-pane]');
    const scroll = pane.scrollTop;
    current.el.querySelectorAll('[data-stab]').forEach((t) => t.classList.toggle('active', t.dataset.stab === tab));
    pane.innerHTML = panes[tab][2]();
    pane.scrollTop = scroll;
};

// Capture (remapping)

const stopCapture = () => {
    if (!capture) return;
    capture.sub?.unsub();
    if (capture.kind === 'kb') pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: true});
    if (capture.kind === 'pad') joystick.capture(false);
    capture = null;
};

const startKb = (key) => {
    stopCapture();
    capture = {kind: 'kb', key};
    pub(KEYBOARD_TOGGLE_FILTER_MODE, {mode: false});
    capture.sub = sub(KEYBOARD_KEY_PRESSED, ({key: code}) => {
        const target = capture.key;
        stopCapture();
        if (code !== 'Escape' || target === KEY.MENU) {
            const map = keyboard.map;
            const prev = map[code];
            codesFor(map, target).forEach((c) => delete map[c]);
            map[code] = target;
            keyboard.remap(map);
            if (prev && prev !== target) toast.warning(`${keyName(code)} saiu de “${buttons[prev] || prev}”.`);
        }
        render();
    });
    render();
};

const startPad = (key) => {
    stopCapture();
    if (!joystick.active()) {
        toast.warning('Conecte um controle e aperte um botão antes de remapear.');
        return;
    }
    capture = {kind: 'pad', key};
    joystick.capture(true);
    capture.sub = sub(GAMEPAD_BUTTON_CAPTURED, ({index, type}) => {
        const target = capture.key;
        stopCapture();
        const map = joystick.map;
        const prev = map[index];
        codesFor(map, target).forEach((c) => delete map[c]);
        map[index] = target;
        joystick.remap(map);
        if (prev && prev !== target) toast.warning(`${padButtonName(type, index)} saiu de “${buttons[prev] || prev}”.`);
        render();
    });
    render();
};

// Change handlers

const onOptChange = (input) => {
    const id = input.dataset.opt;
    let v;
    if (input.type === 'checkbox') v = input.checked;
    else if (input.type === 'range') v = parseFloat(input.value);
    else v = input.value;

    switch (id) {
        case opts.MIRROR_SCREEN:
            v = v ? 'mirror' : 'none';
            break;
        case opts.VIDEO_PIXEL:
            v = v === 'pixel';
            break;
        case opts.LOG_LEVEL:
            v = +v;
            break;
    }
    settings.set(id, v);
    const out = current?.el.querySelector(`[data-out="${CSS.escape(id)}"]`);
    if (out) out.textContent = formats[id] ? formats[id](v) : v;
};

const wire = (root) => {
    on(root, 'click', '[data-stab]', (e, b) => {
        stopCapture();
        tab = b.dataset.stab;
        render();
    });
    on(root, 'click', '[data-bind-kb]', (e, b) => startKb(b.dataset.bindKb));
    on(root, 'click', '[data-bind-pad]', (e, b) => startPad(b.dataset.bindPad));
    on(root, 'change', '[data-opt]', (e, i) => onOptChange(i));
    on(root, 'input', 'input[type=range][data-opt]', (e, i) => onOptChange(i));
    on(root, 'click', '[data-reset]', (e, b) => {
        if (b.dataset.reset === 'kb') keyboard.remap({...defaultKeyboardMap});
        else joystick.remap({...defaultGamepadMap});
        toast.success('Controles restaurados para o padrão.');
        render();
    });
    on(root, 'click', '[data-reset-all]', async () => {
        if (await confirmDialog({title: 'Restaurar configurações?', text: 'Controles, vídeo e áudio voltarão ao padrão.', confirm: 'Restaurar', danger: true, iconName: 'rotate'})) {
            settings.resetAll();
            toast.success('Configurações restauradas.');
            render();
        }
    });
    on(root, 'click', '[data-logout]', () => ctx.onLogout());
    on(root, 'click', '[data-avatar-pick]', () => root.querySelector('[data-avatar-file]').click());
    on(root, 'change', '[data-avatar-file]', async (e, input) => {
        const file = input.files?.[0];
        input.value = '';
        if (!file) return;
        try {
            const blob = await squareBlob(file, 256);
            await fetch('/api/account/avatar', {method: 'PUT', credentials: 'same-origin', headers: {'Content-Type': 'image/png'}, body: blob})
                .then(async (r) => {
                    if (!r.ok) throw new Error((await r.json().catch(() => ({})))?.error?.message || 'Não foi possível enviar a foto.');
                });
            bumpAvatar(ctx.me.id);
            document.dispatchEvent(new Event('avatar-changed'));
            toast.success('Foto de perfil atualizada!');
            render();
        } catch (err) {
            toast.error(err.message || 'Não foi possível usar essa imagem.');
        }
    });
    on(root, 'click', '[data-avatar-remove]', async () => {
        try {
            await http.del('/api/account/avatar');
            bumpAvatar(ctx.me.id);
            document.dispatchEvent(new Event('avatar-changed'));
            toast.success('Foto removida.');
            render();
        } catch (err) {
            toast.error(err.message);
        }
    });
    on(root, 'submit', '[data-form]', async (e, form) => {
        e.preventDefault();
        const btn = form.querySelector('[type=submit]');
        btn.classList.add('is-loading');
        try {
            if (form.dataset.form === 'name') {
                const {name} = await http.put('/api/account/profile', {name: form.querySelector('#s-name').value});
                ctx.me.name = name;
                ctx.onNameChanged(name);
                toast.success('Nome atualizado.');
            } else {
                const cur = form.querySelector('#s-cur').value;
                const next = form.querySelector('#s-new').value;
                if (next !== form.querySelector('#s-conf').value) {
                    toast.error('A confirmação não confere com a nova senha.');
                    return;
                }
                await http.put('/api/account/password', {current: cur, new: next});
                form.reset();
                toast.success('Senha alterada com sucesso.');
            }
        } catch (err) {
            toast.error(err.message);
        } finally {
            btn.classList.remove('is-loading');
        }
    });
};

sub(GAMEPAD_CONNECTED, () => current && tab === 'controls' && !capture && render());
sub(GAMEPAD_DISCONNECTED, () => current && tab === 'controls' && render());

export const settingsView = {
    configure(c) {
        ctx = {...ctx, ...c};
    },
    open(startTab = tab) {
        if (current) return;
        tab = panes[startTab] ? startTab : 'controls';
        current = modal({
            title: 'Configurações',
            wide: true,
            cls: 'settings-modal',
            body: `<div class="settings-layout" style="margin:0 -24px -24px">
                <nav class="settings-nav">${Object.entries(panes).map(([k, [t, i]]) =>
                    `<button class="tab" data-stab="${k}">${icon(i)}${t}</button>`).join('')}</nav>
                <div class="settings-pane" data-pane></div></div>`,
            onClose: () => {
                stopCapture();
                current = null;
            },
        });
        wire(current.el);
        render();
    },
    get isOpen() {
        return !!current;
    },
    get capturing() {
        return !!capture;
    },
};

/** Keyboard and gamepad shortcuts sheet. */
export const showHelp = () => {
    const kb = keyboard.map;
    const pad = joystick.map;
    const type = joystick.active()?.type || 'xbox';
    const row = (k) => {
        const codes = codesFor(kb, k).map((c) => `<kbd>${esc(keyName(c))}</kbd>`).join('');
        const btns = codesFor(pad, k).map((i) => `<span class="pad-btn">${esc(padButtonName(type, +i))}</span>`).join('');
        return `<div class="shortcut"><span>${esc(buttons[k] || k)}</span><span>${codes || '—'}${btns}</span></div>`;
    };
    modal({
        title: 'Atalhos e controles',
        wide: true,
        body: `<div class="section-title">${icon('gamepad')}Jogo</div>
            <div class="shortcuts">${[KEY.UP, KEY.DOWN, KEY.LEFT, KEY.RIGHT, KEY.A, KEY.B, KEY.X, KEY.Y, KEY.L, KEY.R, KEY.L2, KEY.R2, KEY.START, KEY.SELECT].map(row).join('')}</div>
            <div class="section-title" style="margin-top:24px">${icon('zap')}Atalhos</div>
            <div class="shortcuts">${[KEY.MENU, KEY.SAVE, KEY.LOAD, KEY.FULL, KEY.STATS, KEY.CHAT].map(row).join('')}
                <div class="shortcut"><span>Menu do jogo (controle)</span><span><span class="pad-btn">${esc(padButtonName(type, 9))}</span>+<span class="pad-btn">${esc(padButtonName(type, 8))}</span></span></div>
                <div class="shortcut"><span>Buscar na biblioteca</span><span><kbd>/</kbd></span></div>
            </div>
            <p class="subtle" style="margin-top:16px">Na interface, use o direcional do controle para navegar, ${esc(padButtonName(type, 0))} para confirmar e ${esc(padButtonName(type, 1))} para voltar. Você pode mudar tudo em Configurações › Controles.</p>`,
        footer: `<button class="btn outline" data-open-settings>${icon('settings')}Personalizar</button><button class="btn primary" data-close>Fechar</button>`,
    }).el.querySelector('[data-open-settings]').addEventListener('click', (e) => {
        e.target.closest('.modal-backdrop').querySelector('[data-close]')?.click();
        settingsView.open('controls');
    });
};
