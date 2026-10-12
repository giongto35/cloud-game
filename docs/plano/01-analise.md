# 01 — Análise do código existente

## Visão geral

O projeto é um fork do **CloudRetro** (`giongto35/cloud-game`). Emuladores Libretro rodam no
servidor; vídeo e áudio são codificados com GStreamer e transmitidos ao navegador por WebRTC.
Os comandos do jogador voltam por um *data channel* WebRTC.

```
Navegador ──WS /ws──▶ Coordinator ◀──WS /wso── Worker (emulador + GStreamer)
    ▲                                              │
    └──────────── WebRTC (vídeo/áudio/input) ──────┘
```

| Componente | Local | Responsabilidade |
|---|---|---|
| Coordinator | `cmd/coordinator`, `pkg/coordinator` | Serve o site, recebe usuários e workers, escolhe worker, faz a sinalização WebRTC |
| Worker | `cmd/worker`, `pkg/worker` | Executa o emulador, cria a sala, transmite mídia, recebe input |
| Sala | `pkg/worker/room` | 1 jogo por worker; envia mídia para **todos** os peers conectados |
| Protocolo | `pkg/api` | Pacotes JSON `{t, id, p}` com códigos numéricos |
| Frontend | `web/` | ~4,3 mil linhas JS puro; visual skeuomórfico (Game Boy) com imagens PNG |

## Como o fluxo funciona hoje

1. O navegador abre `/ws`. O coordinator escolhe imediatamente um worker livre (ou o worker da sala do link).
2. Sem worker livre → envia `ErrNoFreeSlots` e **encerra** a conexão. Não há fila.
3. O worker cria um peer WebRTC por usuário. O usuário escolhe um jogo e envia `StartGame`.
4. A vaga do worker é reservada no início do jogo (`TryReserve`) e liberada quando a sala fecha (`CloseRoom`).
5. Qualquer pessoa que abre o link `?id=<sala>` entra na mesma sala e recebe a mesma transmissão.

## Problemas encontrados

### Segurança
| # | Problema | Impacto |
|---|---|---|
| S1 | Nenhuma autenticação: qualquer pessoa com a URL joga e entra em qualquer sala | Crítico |
| S2 | `player_index` vem do cliente sem validação; `InputState[port]` é um array de 4 posições → índice ≥ 4 ou negativo **derruba o worker** (panic) | Crítico |
| S3 | Qualquer pessoa na sala pode salvar, carregar, reiniciar e trocar de jogador | Alto |
| S4 | Não existe papel de espectador: todos os presentes controlam o jogo | Alto |
| S5 | `HandleWebrtcSignal` usa `u.w` sem checar `nil` | Médio |

### Experiência
| # | Problema |
|---|---|
| U1 | Interface toda em inglês, com textos técnicos ("No free slots :(") |
| U2 | Visual de console portátil escalado por `transform`, ruim em telas largas e em celulares |
| U3 | Lista de jogos estilo "rolo", sem busca, sem filtro e sem capas |
| U4 | Número do jogador escolhido num *slider* sem indicar quem está em cada vaga |
| U5 | Sem fila, sem aviso de servidor ocupado além de um *toast* de 1 segundo |
| U6 | Atalhos perigosos: `Q` sai do jogo imediatamente |
| U7 | Mapeamento de controle por sistema operacional/navegador "na mão", sem tela de remapeamento |
| U8 | Configurações em painel técnico (log level, wait offer) misturadas às do jogador |

### Manutenção
| # | Observação |
|---|---|
| M1 | Testes desativados nos últimos commits (`TestStateConcurrency`, consistência de savestate) |
| M2 | Mudanças grandes tornam difícil acompanhar o projeto original: o fork passa a ser produto próprio |
| M3 | O coordinator não guarda nenhum estado persistente (sem banco) |

## Pontos fortes a preservar

- Pipeline de mídia (GStreamer + Pion WebRTC) maduro e de baixa latência.
- A sala já transmite para vários peers → base pronta para **espectadores** e **fila com stream**.
- Input compacto e eficiente (`retropad.js`: 10 bytes por atualização, via `requestAnimationFrame`).
- Suporte a teclado+mouse para cores como DOSBox (pointer lock, keyboard lock).
- Configuração flexível via YAML + variáveis `CLOUD_GAME_*`.
