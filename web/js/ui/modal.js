import {icon} from './icons.js?v=4';
import {esc, html} from './dom.js?v=4';
import {nav} from './nav.js?v=4';

const stack = [];

/**
 * Opens a modal dialog.
 * @param {Object} o
 * @param {string} [o.title]
 * @param {string|HTMLElement} o.body
 * @param {string} [o.footer] footer markup
 * @param {boolean} [o.wide]
 * @param {boolean} [o.dismissible=true]
 * @param {Function} [o.onClose]
 * @returns {{el: HTMLElement, close: Function}}
 */
export const modal = ({title = '', body = '', footer = '', wide = false, dismissible = true, onClose, cls = ''} = {}) => {
    const backdrop = html(`<div class="modal-backdrop">
        <div class="modal ${wide ? 'wide' : ''} ${cls}" role="dialog" aria-modal="true" ${title ? `aria-label="${esc(title)}"` : ''}>
            ${title ? `<div class="modal-head"><h2>${esc(title)}</h2>
                ${dismissible ? `<button class="btn ghost icon-only sm" data-close aria-label="Fechar">${icon('x')}</button>` : ''}</div>` : ''}
            <div class="modal-body"></div>
            ${footer ? `<div class="modal-foot">${footer}</div>` : ''}
        </div></div>`);
    const bodyEl = backdrop.querySelector('.modal-body');
    typeof body === 'string' ? (bodyEl.innerHTML = body) : bodyEl.append(body);

    const prevFocus = document.activeElement;
    let closed = false;
    const close = (result) => {
        if (closed) return;
        closed = true;
        backdrop.remove();
        const i = stack.indexOf(api);
        if (i >= 0) stack.splice(i, 1);
        nav.popScope(backdrop);
        prevFocus?.focus?.({preventScroll: true});
        onClose?.(result);
    };
    const api = {el: backdrop, close, dismissible};

    backdrop.addEventListener('click', (e) => {
        if (e.target.closest('[data-close]') || (dismissible && e.target === backdrop)) close();
    });

    document.body.append(backdrop);
    stack.push(api);
    nav.pushScope(backdrop);
    requestAnimationFrame(() => {
        const first = backdrop.querySelector('[autofocus]') ||
            backdrop.querySelector('.modal-foot .btn.primary') ||
            backdrop.querySelector('input, button:not([data-close]), [tabindex]');
        first?.focus({preventScroll: true});
    });
    return api;
};

/** Closes the top-most dismissible modal. Returns true if one was closed. */
modal.closeTop = () => {
    const top = stack[stack.length - 1];
    if (top && top.dismissible) {
        top.close();
        return true;
    }
    return false;
};

modal.isOpen = () => stack.length > 0;

/** Confirmation dialog. Resolves to true/false. */
export const confirmDialog = ({title, text, confirm = 'Confirmar', cancel = 'Cancelar', danger = false, iconName = 'help'}) =>
    new Promise((resolve) => {
        let result = false;
        const m = modal({
            body: `<div class="modal-hero"><div class="hero-icon ${danger ? 'danger' : ''}">${icon(iconName, 'xl')}</div>
                <h2>${esc(title)}</h2><p class="muted">${esc(text)}</p></div>`,
            footer: `<button class="btn ghost" data-close>${esc(cancel)}</button>
                <button class="btn ${danger ? 'danger' : 'primary'}" data-ok>${esc(confirm)}</button>`,
            onClose: () => resolve(result),
        });
        m.el.querySelector('[data-ok]').addEventListener('click', () => {
            result = true;
            m.close();
        });
    });

/** Simple information dialog. */
export const infoDialog = ({title, text, iconName = 'info', kind = '', button = 'Entendi', onClose}) => {
    const m = modal({
        body: `<div class="modal-hero"><div class="hero-icon ${kind}">${icon(iconName, 'xl')}</div>
            <h2>${esc(title)}</h2><p class="muted">${esc(text)}</p></div>`,
        footer: `<button class="btn primary" data-close>${esc(button)}</button>`,
        onClose,
    });
    return m;
};
