# 03 — Design e experiência do usuário

## Identidade

- **Nome**: Belko Arcade — "Seus clássicos, na nuvem, com os amigos."
- **Tom**: direto, amigável, sem jargão técnico. Erros dizem o que aconteceu **e o que fazer**.
- **Estética**: escuro, elegante, com acentos neon contidos (violeta → ciano) e textura sutil de
  grade/scanline remetendo ao arcade, sem cair no caricato.

## Design system (tokens CSS em `web/css/tokens.css`)

| Token | Valor | Uso |
|---|---|---|
| `--bg` | `#0b0d14` | fundo da página |
| `--surface-1/2/3` | `#121521` / `#181c2b` / `#20253a` | cartões, painéis, campos |
| `--border` | `#2a3048` | divisórias |
| `--text` / `--text-2` / `--text-3` | `#eef0f8` / `#a7adc4` / `#6e7591` | hierarquia de texto |
| `--accent` | `#7c5cff` | ação principal |
| `--accent-2` | `#22d3ee` | destaque secundário, foco |
| `--success` / `--warning` / `--danger` | `#22c55e` / `#f59e0b` / `#ef4444` | estados |
| `--radius-s/m/l` | 8 / 12 / 20 px | cantos |
| Fonte UI | Inter (fallback system-ui) | texto |
| Fonte de títulos | Space Grotesk | títulos e logotipo |

Escala de espaçamento 4/8/12/16/24/32/48. Sombras suaves; foco sempre visível (anel ciano 2px).
Movimento: 150–250 ms, `prefers-reduced-motion` respeitado. Contraste AA em todo texto.

## Telas

1. **Entrar / Criar conta** (`/entrar`)
   Duas colunas: arte gerada em CSS com o logotipo e frases; cartão com abas *Entrar* / *Criar conta*.
   Validação em linha, mostrar/ocultar senha, medidor de força, mensagens claras.
2. **Aguardando aprovação** — ilustração, explicação ("O administrador foi avisado"), botão *Verificar
   novamente* (checagem automática a cada 30 s) e *Sair*. Variações para conta recusada/bloqueada.
3. **Biblioteca** (início)
   Barra superior fixa: logotipo · Biblioteca · Ao vivo · Admin (se permitido) · indicador da fila ·
   status do servidor · menu da conta.
   Faixa "Jogados recentemente"; busca instantânea (atalho `/`); filtros por sistema em *chips*;
   grade de cartões com **capa gerada** (gradiente único por jogo + sigla do sistema + título).
   Cartão: *Jogar* e *Criar sala*; foco/hover com elevação.
   Aviso quando todos os servidores estão ocupados, com *Entrar na fila*.
4. **Ao vivo** — cartões das salas em andamento: jogo, vagas livres, jogadores, espectadores;
   botões *Assistir* e *Entrar* (se houver vaga).
5. **Jogo**
   Palco preto ocupando a tela, vídeo centralizado mantendo proporção.
   HUD superior que some após 3 s sem movimento: voltar, título, código da sala + *Convidar*,
   avatares dos jogadores, indicador de conexão (verde/amarelo/vermelho + ms), tela cheia, menu.
   Painel lateral recolhível com abas **Jogadores** (4 vagas com nome, anfitrião ⭐, botão
   *Ocupar*, menu do anfitrião: mover/expulsar) e **Chat**.
   **Menu do jogo** (`Esc`, `F1`, botão Home/Start+Select no controle): Continuar, Salvar,
   Carregar, Reiniciar, Trocar vaga, Convidar, Configurações, Sair da sala.
   Etapas de carregamento ("Conectando ao servidor…", "Preparando o emulador…") em vez de GIF.
   Selo **ASSISTINDO** para espectadores.
6. **Fila** — pílula fixa na barra ("Fila: 2º de 5"); painel com posição, tempo de espera e botão
   *Assistir enquanto espera*; modal **"Sua vez chegou!"** com contagem regressiva de 30 s, som
   curto e vibração do controle.
7. **Configurações** (modal com abas) — Controles (teclado e controle com remapeamento por captura,
   zona morta, modo direcional/analógico, controles na tela), Vídeo (suavização, tela cheia
   automática), Áudio (volume), Conta (nome, senha), Avançado (diagnóstico).
8. **Admin** — abas: **Pendentes** (com contador), **Usuários** (busca, papel, bloquear, redefinir
   senha), **Salas e fila** (encerrar sala), **Servidores**, **Auditoria**.

## Controles

| Ação | Teclado | Controle (padrão W3C) |
|---|---|---|
| Direcional | Setas | D-pad / analógico esquerdo |
| A / B / X / Y | X / Z / S / A | B1(direita) / B0(baixo) / B3(cima) / B2(esquerda) |
| L / R | Q / W | LB / RB |
| L2 / R2 | E / R | LT / RT |
| L3 / R3 | C / V | L3 / R3 |
| Start / Select | Enter / Shift direito | Start / Back |
| Menu do jogo | Esc ou F1 | Home, ou Start+Select |
| Salvar / Carregar | F2 / F4 | — (pelo menu) |
| Tela cheia | F | — |
| Estatísticas | F8 | — |
| Chat | T (abre e foca) | — |

- Na interface (biblioteca, menus, modais) o controle navega por **foco espacial**: direcional move,
  A confirma, B volta/fecha, Start abre o menu.
- Ícones de botões se adaptam ao controle detectado (Xbox, PlayStation ou genérico).
- Remapeamento por captura: clique em "Mudar", aperte a tecla/botão; conflito é avisado.
- Perfis salvos localmente **e** na conta (`/api/account/preferences`).
- Celular/tablet: controle virtual sobreposto ao vídeo, com D-pad/analógico, ABXY, L/R,
  Start/Select e botão de menu; opacidade e tamanho ajustáveis.

## Acessibilidade e responsividade

- Layout fluido de 360 px a 4K; grade da biblioteca com `auto-fill`.
- Navegação completa por teclado e por controle; `aria-live` para avisos e fila.
- Textos alternativos e rótulos em todos os botões de ícone.
