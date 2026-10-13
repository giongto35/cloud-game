import {icon} from '../ui/icons.js?v=4';
import {esc} from '../ui/dom.js?v=4';
import {modal} from '../ui/modal.js?v=4';
import {toast} from '../ui/toast.js?v=4';
import {gameTitle} from '../i18n.js?v=4';

/**
 * Waiting queue UI: top bar pill, details dialog and the "your turn" call.
 */

let pill;
let h = {onAccept: () => ({}), onLeave: () => ({}), onWatch: () => ({}), onRejoin: () => ({}), rooms: () => [], rumble: () => ({})};
let state = {status: 'none'};
let callModal = null;
let infoModal = null;
let countdown = null;
let lastGame = '';
let expecting = false;

const chime = () => {
    try {
        const ctx = new (window.AudioContext || window.webkitAudioContext)();
        const notes = [660, 880, 1320];
        notes.forEach((f, i) => {
            const o = ctx.createOscillator();
            const g = ctx.createGain();
            o.type = 'triangle';
            o.frequency.value = f;
            g.gain.setValueAtTime(0.0001, ctx.currentTime + i * 0.12);
            g.gain.exponentialRampToValueAtTime(0.15, ctx.currentTime + i * 0.12 + 0.02);
            g.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + i * 0.12 + 0.25);
            o.connect(g).connect(ctx.destination);
            o.start(ctx.currentTime + i * 0.12);
            o.stop(ctx.currentTime + i * 0.12 + 0.3);
        });
        setTimeout(() => ctx.close(), 1000);
    } catch {
        // audio is optional
    }
};

const renderPill = () => {
    if (!pill) return;
    const s = state.status;
    pill.hidden = s !== 'waiting' && s !== 'called';
    pill.classList.toggle('called', s === 'called');
    pill.innerHTML = s === 'called'
        ? `${icon('play')}Sua vez!`
        : `${icon('hourglass')}Fila: ${state.position}º de ${state.size}`;
};

const closeCall = () => {
    clearInterval(countdown);
    countdown = null;
    callModal?.close();
    callModal = null;
};

const openCall = () => {
    if (callModal) return;
    infoModal?.close();
    const {title} = gameTitle(state.game);
    let left = Math.max(1, state.expires_in || 30);
    let accepted = false;
    callModal = modal({
        dismissible: false,
        body: `<div class="modal-hero">
            <div class="hero-icon success">${icon('gamepad', 'xl')}</div>
            <h2>Sua vez chegou!</h2>
            <p class="muted">Um servidor foi reservado para você jogar <b>${esc(title)}</b>.</p>
            <p class="badge warning" style="height:28px;font-size:13px">Reservado por <b data-left>${left}s</b></p>
        </div>`,
        footer: `<button class="btn ghost" data-decline>Agora não</button>
            <button class="btn primary lg" data-accept autofocus>${icon('play')}Jogar agora</button>`,
        onClose: () => {
            clearInterval(countdown);
            callModal = null;
        },
    });
    const leftEl = callModal.el.querySelector('[data-left]');
    countdown = setInterval(() => {
        left = Math.max(0, left - 1);
        leftEl.textContent = `${left}s`;
        if (left === 0) clearInterval(countdown);
    }, 1000);
    callModal.el.querySelector('[data-accept]').addEventListener('click', () => {
        accepted = true;
        const game = state.game;
        closeCall();
        h.onAccept(game);
    });
    callModal.el.querySelector('[data-decline]').addEventListener('click', () => {
        closeCall();
        if (!accepted) h.onLeave();
    });
    chime();
    h.rumble();
    if (document.hidden && 'Notification' in window && Notification.permission === 'granted') {
        try {
            new Notification('Sua vez chegou!', {body: `Volte ao Arcade para jogar ${title}.`});
        } catch {
            // ignore
        }
    }
};

const openInfo = () => {
    if (state.status === 'called') return openCall();
    if (infoModal) return;
    const {title} = gameTitle(state.game);
    const rooms = h.rooms();
    infoModal = modal({
        title: 'Fila de espera',
        body: `<div class="modal-hero" style="padding-top:8px">
            <div class="hero-icon">${icon('hourglass', 'xl')}</div>
            <h2>${state.position}º lugar na fila</h2>
            <p class="muted">${state.size} ${state.size === 1 ? 'pessoa' : 'pessoas'} esperando · você vai jogar <b>${esc(title)}</b>.</p>
            <p class="subtle">Você pode fechar esta janela e continuar navegando. Avisaremos quando chegar a sua vez — mantenha a página aberta.</p>
        </div>`,
        footer: `<button class="btn danger" data-qleave>${icon('x')}Sair da fila</button>
            ${rooms.length ? `<button class="btn primary" data-qwatch>${icon('tv')}Assistir enquanto espera</button>` : '<button class="btn primary" data-close>Ok</button>'}`,
        onClose: () => (infoModal = null),
    });
    infoModal.el.querySelector('[data-qleave]').addEventListener('click', () => {
        infoModal.close();
        h.onLeave();
    });
    infoModal.el.querySelector('[data-qwatch]')?.addEventListener('click', () => {
        infoModal.close();
        h.onWatch(rooms[0].id);
    });
    if ('Notification' in window && Notification.permission === 'default') Notification.requestPermission().catch(() => ({}));
};

export const queueView = {
    mount(pillEl, handlers) {
        pill = pillEl;
        h = {...h, ...handlers};
        pill.addEventListener('click', openInfo);
        renderPill();
    },
    /** Applies a QueueState message from the server. */
    set(q = {status: 'none'}) {
        const prev = state.status;
        if (q.status === 'expired') {
            closeCall();
            state = {status: 'none'};
            renderPill();
            toast.warning('O tempo para aceitar acabou e você saiu da fila.', 6000);
            return;
        }
        state = q;
        if (q.game) lastGame = q.game;
        renderPill();
        if (q.status === 'called' && prev !== 'called') openCall();
        if (q.status !== 'called' && prev === 'called') closeCall();
        if (q.status === 'waiting' && expecting) {
            expecting = false;
            toast(`Servidores ocupados — você entrou na fila (${q.position}º lugar).`, 'info', 5000);
        }
        if (q.status !== 'waiting') infoModal?.close();
    },
    get state() {
        return state;
    },
    get inQueue() {
        return state.status === 'waiting' || state.status === 'called';
    },
    openInfo,
    lastGame: () => lastGame,
    /** The next "waiting" state comes from a join made by this user. */
    expect() {
        expecting = true;
    },
};
