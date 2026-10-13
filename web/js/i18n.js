/**
 * Português (Brasil) — textos compartilhados da interface.
 * Os textos específicos de cada tela ficam junto da própria tela.
 */

export const systems = {
    gba: {short: 'GBA', name: 'Game Boy Advance'},
    gbc: {short: 'GBC', name: 'Game Boy Color'},
    nes: {short: 'NES', name: 'Nintendo (NES)'},
    snes: {short: 'SNES', name: 'Super Nintendo'},
    n64: {short: 'N64', name: 'Nintendo 64'},
    pcsx: {short: 'PS1', name: 'PlayStation'},
    mame: {short: 'ARCADE', name: 'Arcade'},
    dos: {short: 'DOS', name: 'MS-DOS'},
};

export const systemShort = (s = '') => systems[s]?.short || s.toUpperCase() || '—';
export const systemName = (s = '') => systems[s]?.name || s.toUpperCase() || 'Outro';

export const roles = {
    admin: 'Administrador',
    moderator: 'Moderador',
    user: 'Jogador',
};

export const statuses = {
    pending: 'Pendente',
    approved: 'Aprovado',
    rejected: 'Recusado',
    blocked: 'Bloqueado',
};

export const auditActions = {
    'admin.created': 'Conta de administrador criada',
    'user.registered': 'Novo cadastro',
    'user.approved': 'Cadastro aprovado',
    'user.rejected': 'Cadastro recusado',
    'user.blocked': 'Conta bloqueada',
    'user.unblocked': 'Conta desbloqueada',
    'user.role_changed': 'Papel alterado',
    'user.password_reset': 'Senha redefinida',
    'user.password_changed': 'Senha alterada',
    'room.closed': 'Sala encerrada',
    'covers.retry': 'Nova busca de capas',
    'meta.retry': 'Atualização do catálogo',
    'cover.set': 'Capa trocada',
    'cover.reset': 'Capa automática restaurada',
    'tournament.created': 'Torneio criado',
    'tournament.start': 'Torneio iniciado',
    'tournament.cancel': 'Torneio cancelado',
    'match.winner_set': 'Vencedor definido pela administração',
};

/** Names of the virtual controller buttons. */
export const buttons = {
    up: 'Cima', down: 'Baixo', left: 'Esquerda', right: 'Direita',
    a: 'A', b: 'B', x: 'X', y: 'Y',
    l: 'L', r: 'R', l2: 'L2', r2: 'R2', l3: 'L3', r3: 'R3',
    start: 'Start', select: 'Select',
    menu: 'Menu do jogo', save: 'Salvar', load: 'Carregar', full: 'Tela cheia',
    stats: 'Estatísticas', chat: 'Chat',
};

/** Friendly names for keyboard codes. */
export const keyName = (code = '') => {
    const map = {
        ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→',
        Enter: 'Enter', Escape: 'Esc', Space: 'Espaço', Backspace: 'Backspace', Tab: 'Tab',
        ShiftLeft: 'Shift esq.', ShiftRight: 'Shift dir.', ControlLeft: 'Ctrl esq.', ControlRight: 'Ctrl dir.',
        AltLeft: 'Alt', AltRight: 'AltGr', Semicolon: 'Ç', Quote: '´', Period: '.', Comma: ',',
        Slash: ';', Backslash: ']', BracketLeft: '´', BracketRight: '[', Minus: '-', Equal: '=',
    };
    if (map[code]) return map[code];
    if (code.startsWith('Key')) return code.slice(3);
    if (code.startsWith('Digit')) return code.slice(5);
    if (code.startsWith('Numpad')) return 'Num ' + code.slice(6);
    return code;
};

/** Labels for the standard gamepad buttons by controller family. */
export const padLabels = {
    xbox: ['A', 'B', 'X', 'Y', 'LB', 'RB', 'LT', 'RT', 'View', 'Menu', 'LS', 'RS', '↑', '↓', '←', '→', 'Xbox'],
    ps: ['✕', '○', '□', '△', 'L1', 'R1', 'L2', 'R2', 'Create', 'Options', 'L3', 'R3', '↑', '↓', '←', '→', 'PS'],
    nintendo: ['B', 'A', 'Y', 'X', 'L', 'R', 'ZL', 'ZR', '−', '+', 'LS', 'RS', '↑', '↓', '←', '→', 'Home'],
    generic: ['B1', 'B2', 'B3', 'B4', 'LB', 'RB', 'LT', 'RT', 'Select', 'Start', 'LS', 'RS', '↑', '↓', '←', '→', 'Home'],
};

export const padButtonName = (type, index) => (padLabels[type] || padLabels.generic)[index] ?? `B${index}`;

/** Splits "Title (Demo) (U)" into a clean title and tags. */
export const gameTitle = (raw = '') => {
    const tags = [];
    let title = raw.replace(/\s*[([]([^)\]]+)[)\]]/g, (_, t) => {
        tags.push(t.trim());
        return '';
    }).trim();
    if (!title) title = raw;
    return {title, tags};
};

const tagNames = {U: 'EUA', E: 'Europa', J: 'Japão', USA: 'EUA', Europe: 'Europa', Japan: 'Japão', PD: 'Domínio público', Demo: 'Demo', Beta: 'Beta', Hack: 'Hack'};
export const tagName = (t) => tagNames[t] || t;
