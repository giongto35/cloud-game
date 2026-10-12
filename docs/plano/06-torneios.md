# 06 — Torneios automatizados e replays

Módulo de torneios **eliminatória simples** com chaves sorteadas, **no máximo 2 partidas simultâneas**,
espectadores via WebRTC, **detecção automática do vencedor pela RAM emulada** e **replays determinísticos**
(savestate do frame zero + log de inputs por frame).

Jogos iniciais: Ultimate Mortal Kombat 3, Super Bomberman 2 e Super Mario Kart (SNES, core snes9x).

## 1. Mapas de RAM — verificados no emulador

Nenhum endereço foi copiado às cegas. Os candidatos vieram de códigos Pro Action Replay e do
[Data Crystal](https://datacrystal.tcrf.net/wiki/Super_Mario_Kart/RAM_map); os que faltavam foram
**descobertos e confirmados** com o laboratório de RAM (`pkg/worker/caged/libretro/ramlab_test.go`):
o jogo roda sem interface, com inputs programados, capturas de tela e gravação da RAM inteira
(WRAM, `$7E0000–$7FFFFF`) a cada 4 frames. A análise (`lab/analyze.py`) procura bytes que mudam
**apenas** logo depois de cada evento (nocaute, morte, chegada), com +1, e que ficam constantes no
cenário inverso (o outro jogador vencendo).

| Jogo | P1 | P2 | Vitória | Como foi confirmado |
|---|---|---|---|---|
| Ultimate Mortal Kombat 3 | `7E3AA6` | `7E3AA8` | rounds vencidos ≥ 2 | 3 cenários (P1 vence, P2 vence, misto 2–1); casa com os medalhões na tela |
| Super Bomberman 2 (Batalha) | `7E5E66` | `7E5E68` | troféus ≥ meta (padrão 3) | P1 e P2 se explodem 3×; casa com o quadro de troféus |
| Super Mario Kart (Match Race) | `7E10C1` | `7E11C1` | primeiro a chegar a **133** | escrita forçada na volta final + cruzar a linha → o jogo mostra *WON/LOST* no mesmo frame |

Notas de calibração:
- UMK3: energia em `7E36D4`/`7E3898` (166 = cheia) é zerada entre rounds — não serve como sinal de vitória.
- SMK: a volta usa a escala 128 = volta 1 … 132 = volta final, 133 = terminou; o kart 2 é a estrutura do kart 1 + `0x100`.
- SB2: a bomba é o botão **A**; os contadores voltam a 0 quando o jogo retorna ao menu.
- **Bug corrigido**: o SNES usava multitap na porta 2 para todos os jogos, o que impede o jogador 2 de
  jogar Super Mario Kart. Agora o multitap só é ligado para os jogos que o suportam (`hid4rom`).

Regras de robustez do detector (`pkg/tourney/watcher.go`):
- **Armar**: só passa a valer depois de ver os dois contadores abaixo da meta (ignora lixo de menus).
- **Estabilidade**: o valor vencedor precisa ficar 30 frames estável (evita leituras transitórias).
- **Desempate**: se os dois atingirem a meta no mesmo frame, nenhum vence (o admin decide).
- Toda regra fica numa tabela de dados — um jogo novo é uma linha, não código novo.

## 2. Arquitetura

```
Coordinator                                         Worker (1 sala por worker)
┌────────────────────────────────────────┐         ┌──────────────────────────────────────┐
│ pkg/tournament                          │         │ pkg/tourney                   │
│  bracket.go   sorteio + chaves + byes   │  start  │  watcher.go  regra → vencedor        │
│  scheduler.go semáforo (2) + fila FIFO  │ ──────▶ │  replay.go   gravação / reprodução   │
│  manager.go   avanço automático         │         │  rules.go    tabela de endereços      │
│ store (SQLite) torneios/partidas/replays│ ◀────── │ Frontend: ganchos por frame          │
└────────────────────────────────────────┘ result+  └──────────────────────────────────────┘
                                            replay
```

### Modelos (SQLite, migração 3)

| Tabela | Campos principais |
|---|---|
| `tournaments` | id, name, game, system, status (`registering`/`running`/`finished`/`canceled`), target, created_by, winner_id, datas |
| `tournament_players` | tournament_id, user_id, seed |
| `matches` | id, tournament_id, round, position, p1_id, p2_id, status (`pending`/`ready`/`starting`/`running`/`finished`), winner_id, source (`ram`/`admin`/`walkover`), score, room_id, replay_id, datas |
| `replays` | id, match_id, game, frames, bytes, file, created_at |

### Chaves (single elimination)

- Participantes embaralhados com `crypto/rand`; tamanho = próxima potência de 2.
- **Byes**: quem não tem adversário avança sozinho (`walkover`).
- Partida `(round r, posição p)` alimenta `(r+1, p/2)`, lado `p%2`.
- Nomes das fases: Final, Semifinal, Quartas de final, Oitavas de final, Rodada de 32…

### Fila de execução e concorrência

- `Scheduler` com **semáforo de capacidade 2** (`coordinator.tournaments.maxConcurrent`).
- Partidas `ready` entram numa fila FIFO ordenada por (rodada, posição).
- Um laço único consome a fila: adquire uma vaga do semáforo → reserva um worker livre → avisa os dois
  jogadores. A vaga só é devolvida quando a partida termina (resultado, W.O. ou cancelamento).
- O resultado avança a chave na mesma transação; a próxima partida que ficou pronta entra na fila.
- Tempo para entrar: 3 min. Quem não aparecer perde por W.O.; se nenhum aparecer, o admin decide.

### Salas de torneio

- O worker reservado só aceita os dois participantes como jogadores: P1 = vaga 0, P2 = vaga 1.
  Todos os outros entram como **espectadores** (sem input — validado no coordinator e no worker).
- A sala é criada com gravação ligada e a regra do jogo.

## 3. Replays determinísticos

Formato binário (`.brpl`), compacto:

```
"BRPL" | versão u8 | cabeçalho JSON (jogo, sistema, core, CRC32 da ROM, fps, frame inicial)
savestate do frame zero (comprimido) | eventos
evento = varint(delta de frame) · u8(porta) · u16(máscara de botões)   ← só quando o input muda
```

- **Gravação**: o gancho do frame recebe exatamente a máscara que foi aplicada ao core naquele frame
  (capturada dentro de `syncInputToCache`, sem corrida com o WebRTC). Só mudanças são gravadas
  (uma luta de 3 min com dois jogadores gera poucos KB).
- **Reprodução**: o worker carrega a ROM, restaura o savestate do frame zero e, a cada frame, injeta as
  máscaras do log no cache de input (`SetInputOverride`) — os inputs ao vivo são ignorados.
  O vídeo segue pelo WebRTC normal; todos assistem como espectadores.
- Garantia de determinismo verificada por teste: gravar uma luta, reproduzir e comparar o hash da RAM
  final e o vencedor detectado.

## 4. Plano de implementação

1. `nanoarch`: `SystemRAM()`, máscara de input aplicada por frame, override de input. ☑
2. `libretro.Frontend`: ganchos por frame (`SetFrameHook`, `SetInputOverride`). ☑
3. `config`: `hid4rom` (multitap por jogo) e correção do Super Mario Kart. ☑
4. `pkg/tourney`: regras, watcher, replay (formato, gravador, reprodutor) + testes. ☑
5. Worker: sala de torneio (gravação + detecção + envio do resultado/replay) e sala de replay. ☑
6. `pkg/api`: pacotes `MatchResult`, `ReplayUpload`; `StartGame` com dados de torneio/replay. ☑
7. `pkg/store`: migração 3. ☑
8. `pkg/tournament`: chaves, scheduler com semáforo, gerente com avanço automático + testes. ☑
9. Coordinator: rotas HTTP, reserva de workers para partidas, vagas travadas, avisos aos jogadores. ☑
10. Frontend: lista/criação de torneios, chaveamento ao vivo, "sua partida está pronta", replays. ☑
11. Teste de ponta a ponta: torneio de 4 jogadores com resultado detectado pela RAM. ☑

## 5. Verificação (12/10/2026)

- **Laboratório de RAM**: endereços dos 3 jogos confirmados no snes9x real (seção 1).
- **Determinismo**: `TestReplayDeterminism` grava uma luta de UMK3 e a reproduz só pelo log —
  mesmo vencedor, mesmo frame de detecção e hash idêntico da RAM após 1872 frames.
- **Unidade**: chaves com byes, semáforo (nunca mais de 2 partidas), fila FIFO esperando servidor,
  W.O. por ausência, decisão do admin, gravação e leitura de replays.
- **Ponta a ponta (4 navegadores, 2 workers)**: torneio de UMK3 com 4 inscritos; as duas semifinais
  rodaram ao mesmo tempo em servidores diferentes; os vencedores foram detectados pela RAM (2-0) no
  mesmo instante em que o jogo mostrou "RAIN WINS"; a final foi chamada sozinha quando os servidores
  liberaram; o torneio terminou com campeão; o replay abriu numa sala própria e reproduz a partida.

Problemas reais encontrados e corrigidos durante o teste:
- O worker vazava a porta UDP do WebRTC a cada reconexão com o coordinator (a mídia ia para uma porta
  não publicada) — a fábrica WebRTC agora vive enquanto o worker viver.
- Corrida: o segundo jogador que chegava enquanto o primeiro criava a sala recebia "sem vaga" — agora espera.
- O fim de partida desconectava os jogadores atrás de um diálogo e eles perdiam a chamada da próxima
  partida — agora voltam ao chaveamento conectados, e toda chamada pendente é reenviada ao (re)conectar.

## 6. Operação

- Cada partida simultânea precisa de um worker livre: para 2 partidas ao mesmo tempo, rode 2 workers.
- Configuração em `coordinator.tournaments` (`maxConcurrent`, `joinTimeoutSec`, `replayDir`).
- Replays ficam em `replayDir` (no Docker, `/data/replays`, no volume persistente).
