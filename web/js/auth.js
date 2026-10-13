import {http} from './http.js?v=4';
import {icon} from './ui/icons.js?v=4';
import {esc} from './ui/dom.js?v=4';
import {toast} from './ui/toast.js?v=4';

const card = document.getElementById('card');

document.getElementById('art-logo').innerHTML =
    `<span class="logo-mark">${icon('gamepad')}</span><b>Arcade</b>`;
document.getElementById('features').innerHTML = [
    ['zap', 'Baixa latência'], ['users', 'Até 4 jogadores'], ['tv', 'Assista ao vivo'], ['gamepad', 'Controle, teclado e toque'],
].map(([i, t]) => `<span>${icon(i)}${t}</span>`).join('');

// only same-site relative paths are accepted as a destination
const nextUrl = () => {
    const n = new URLSearchParams(location.search).get('next') || '/';
    return n.startsWith('/') && !n.startsWith('//') ? n : '/';
};

const goApp = () => location.replace(nextUrl());

let pollTimer = null;

const passwordField = (id, label, autocomplete, extra = '') => `
    <div class="field">
        <label for="${id}">${label}</label>
        <div class="input-wrap">
            ${icon('lock')}
            <input class="input with-icon" id="${id}" name="${id}" type="password" autocomplete="${autocomplete}" required minlength="8" maxlength="128" ${extra}>
            <button type="button" class="btn ghost icon-only sm input-action" data-toggle="${id}" aria-label="Mostrar senha">${icon('eye')}</button>
        </div>
        <div class="field-error" data-error="${id}"></div>
    </div>`;

const wireToggles = (root) => root.querySelectorAll('[data-toggle]').forEach((b) => b.addEventListener('click', () => {
    const input = root.querySelector('#' + b.dataset.toggle);
    const show = input.type === 'password';
    input.type = show ? 'text' : 'password';
    b.innerHTML = icon(show ? 'eyeOff' : 'eye');
    b.setAttribute('aria-label', show ? 'Ocultar senha' : 'Mostrar senha');
}));

const setBusy = (btn, busy) => {
    btn.classList.toggle('is-loading', busy);
    btn.disabled = busy;
};

const showFormError = (form, message) => {
    const el = form.querySelector('.form-error');
    el.hidden = !message;
    el.innerHTML = message ? `${icon('alert')}<span>${esc(message)}</span>` : '';
};

const fieldError = (form, id, message) => {
    const input = form.querySelector('#' + id);
    const el = form.querySelector(`[data-error="${id}"]`);
    input?.classList.toggle('invalid', !!message);
    if (el) el.textContent = message || '';
    return !message;
};

const handleUser = (user) => {
    if (user.status === 'approved') return goApp();
    renderStatus(user);
};

// Sign in

const renderLogin = (email = '') => {
    stopPolling();
    card.innerHTML = `
        <div class="head"><h2>Bem-vindo de volta</h2><p>Entre com a sua conta para continuar jogando.</p></div>
        <form novalidate>
            <div class="form-error" hidden></div>
            <div class="field">
                <label for="email">E-mail</label>
                <div class="input-wrap">${icon('mail')}
                    <input class="input with-icon" id="email" type="email" autocomplete="email" required value="${esc(email)}" placeholder="voce@exemplo.com">
                </div>
                <div class="field-error" data-error="email"></div>
            </div>
            ${passwordField('password', 'Senha', 'current-password')}
            <button class="btn primary lg block" type="submit">${icon('login')}Entrar</button>
        </form>
        <p class="auth-switch">Ainda não tem conta? <button type="button" data-go="register">Criar conta</button></p>`;
    const form = card.querySelector('form');
    wireToggles(card);
    card.querySelector('[data-go]').addEventListener('click', () => renderRegister(form.email.value));
    (email ? form.password : form.email).focus();

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        showFormError(form, '');
        const ok = fieldError(form, 'email', form.email.value.trim() ? '' : 'Informe seu e-mail.') &
            fieldError(form, 'password', form.password.value ? '' : 'Informe sua senha.');
        if (!ok) return;
        const btn = form.querySelector('[type=submit]');
        setBusy(btn, true);
        try {
            const {user} = await http.post('/api/auth/login', {email: form.email.value.trim(), password: form.password.value});
            handleUser(user);
        } catch (err) {
            showFormError(form, err.message);
            form.password.select();
        } finally {
            setBusy(btn, false);
        }
    });
};

// Sign up

const strength = (p) => {
    if (!p) return 0;
    let s = 0;
    if (p.length >= 8) s++;
    if (p.length >= 12) s++;
    if (/[A-Z]/.test(p) && /[a-z]/.test(p)) s++;
    if (/\d/.test(p) && /[^A-Za-z0-9]/.test(p)) s++;
    return Math.max(1, Math.min(4, s));
};
const strengthLabel = ['', 'Fraca', 'Razoável', 'Boa', 'Forte'];

const renderRegister = (email = '') => {
    stopPolling();
    card.innerHTML = `
        <div class="head"><h2>Criar conta</h2><p>Depois do cadastro, o administrador aprova o seu acesso.</p></div>
        <form novalidate>
            <div class="form-error" hidden></div>
            <div class="field">
                <label for="name">Nome</label>
                <div class="input-wrap">${icon('user')}
                    <input class="input with-icon" id="name" autocomplete="name" required maxlength="40" placeholder="Como você quer ser chamado">
                </div>
                <div class="field-error" data-error="name"></div>
            </div>
            <div class="field">
                <label for="email">E-mail</label>
                <div class="input-wrap">${icon('mail')}
                    <input class="input with-icon" id="email" type="email" autocomplete="email" required value="${esc(email)}" placeholder="voce@exemplo.com">
                </div>
                <div class="field-error" data-error="email"></div>
            </div>
            ${passwordField('password', 'Senha', 'new-password', 'placeholder="Mínimo de 8 caracteres"')}
            <div class="field" style="margin-top:-8px">
                <div class="strength" data-level="0"><i></i><i></i><i></i><i></i></div>
                <small class="subtle" data-strength>Use letras, números e símbolos.</small>
            </div>
            ${passwordField('confirm', 'Confirme a senha', 'new-password')}
            <button class="btn primary lg block" type="submit">${icon('userPlus')}Criar conta</button>
        </form>
        <p class="auth-switch">Já tem conta? <button type="button" data-go="login">Entrar</button></p>`;
    const form = card.querySelector('form');
    wireToggles(card);
    card.querySelector('[data-go]').addEventListener('click', () => renderLogin(form.email.value));
    form.name.focus();

    const meter = card.querySelector('.strength');
    const meterText = card.querySelector('[data-strength]');
    form.password.addEventListener('input', () => {
        const lvl = strength(form.password.value);
        meter.dataset.level = lvl;
        meterText.textContent = lvl ? `Força da senha: ${strengthLabel[lvl]}` : 'Use letras, números e símbolos.';
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        showFormError(form, '');
        const name = form.name.value.trim(), mail = form.email.value.trim(), pass = form.password.value;
        const ok = fieldError(form, 'name', name.length >= 2 ? '' : 'Informe um nome com pelo menos 2 letras.') &
            fieldError(form, 'email', /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(mail) ? '' : 'Informe um e-mail válido.') &
            fieldError(form, 'password', pass.length >= 8 ? '' : 'A senha precisa ter pelo menos 8 caracteres.') &
            fieldError(form, 'confirm', form.confirm.value === pass ? '' : 'As senhas não são iguais.');
        if (!ok) return;
        const btn = form.querySelector('[type=submit]');
        setBusy(btn, true);
        try {
            const {user} = await http.post('/api/auth/register', {name, email: mail, password: pass});
            toast.success('Conta criada! Agora é só aguardar a aprovação.');
            handleUser(user);
        } catch (err) {
            if (err.code === 'email_taken') fieldError(form, 'email', err.message);
            else showFormError(form, err.message);
        } finally {
            setBusy(btn, false);
        }
    });
};

// Account status (pending / rejected / blocked)

const stopPolling = () => {
    clearInterval(pollTimer);
    pollTimer = null;
};

const check = async (manual = false) => {
    try {
        const {user} = await http.get('/api/auth/me');
        if (user.status === 'approved') {
            toast.success('Acesso aprovado! Boas-vindas ao Arcade.');
            setTimeout(goApp, 600);
        } else if (manual) {
            toast('Seu cadastro ainda está em análise.');
        }
        if (user.status !== 'pending') renderStatus(user);
    } catch (e) {
        if (e.status === 401) renderLogin();
    }
};

const logout = async () => {
    try {
        await http.post('/api/auth/logout');
    } catch {
        // ignore
    }
    renderLogin();
};

const renderStatus = (user) => {
    stopPolling();
    const first = esc(user.name.split(' ')[0]);
    if (user.status === 'pending') {
        card.innerHTML = `
            <div class="status-card">
                <div class="hero-icon"><span class="hourglass">${icon('hourglass')}</span></div>
                <div><h2>Quase lá, ${first}!</h2>
                <p class="muted" style="margin-top:8px">Seu cadastro foi recebido e está aguardando a aprovação do administrador. Esta página verifica sozinha a cada 30 segundos.</p></div>
                <div class="status-steps">
                    <div>${icon('check')}Conta criada com <b>${esc(user.email)}</b></div>
                    <div class="wait">${icon('clock')}Aguardando aprovação</div>
                    <div class="subtle">${icon('gamepad')}Liberação da biblioteca de jogos</div>
                </div>
                <div class="status-actions">
                    <button class="btn outline" data-logout>${icon('logout')}Sair</button>
                    <button class="btn primary" data-check>${icon('refresh')}Verificar agora</button>
                </div>
            </div>`;
        pollTimer = setInterval(check, 30000);
    } else {
        const blocked = user.status === 'blocked';
        card.innerHTML = `
            <div class="status-card">
                <div class="hero-icon danger">${icon(blocked ? 'ban' : 'x')}</div>
                <div><h2>${blocked ? 'Conta bloqueada' : 'Cadastro não aprovado'}</h2>
                <p class="muted" style="margin-top:8px">${blocked
                    ? 'O acesso desta conta foi suspenso pela administração. Se acha que isso é um engano, fale com o administrador.'
                    : 'O administrador não aprovou este cadastro. Se acha que isso é um engano, fale com o administrador.'}</p></div>
                <div class="status-actions"><button class="btn outline" data-logout>${icon('logout')}Sair</button></div>
            </div>`;
    }
    card.querySelector('[data-logout]')?.addEventListener('click', logout);
    card.querySelector('[data-check]')?.addEventListener('click', () => check(true));
};

// Boot

(async () => {
    try {
        const {user} = await http.get('/api/auth/me');
        handleUser(user);
    } catch {
        renderLogin();
    }
})();
