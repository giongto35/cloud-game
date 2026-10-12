# 05 — Operação: rodar, implantar e administrar

## Rodar localmente (Docker)

Pré-requisito: Docker com Compose.

```bash
# 1. defina a senha inicial do administrador (usada só na primeira inicialização)
echo "BELKO_ADMIN_PASSWORD=troque-esta-senha" > .env

# 2. suba tudo (coordinator + worker + Xvfb)
docker compose up --build
```

Acesse `http://localhost:8000` e entre com `wenderson@belko.com.br` e a senha definida.
Se `BELKO_ADMIN_PASSWORD` ficar vazio, uma senha aleatória é gerada e exibida **uma única vez**
no log do contêiner (`docker compose logs cloud-game | grep "Senha inicial"`).

Os jogos ficam em `assets/games/<sistema>/` e os cores Libretro são baixados automaticamente
para `assets/cores`.

## Configuração (`pkg/config/config.yaml` ou variáveis `CLOUD_GAME_*`)

| Chave | Variável de ambiente | Padrão | Descrição |
|---|---|---|---|
| `coordinator.auth.dbPath` | `CLOUD_GAME_COORDINATOR_AUTH_DBPATH` | `./data/belko.db` | Arquivo SQLite |
| `coordinator.auth.adminEmail` | `CLOUD_GAME_COORDINATOR_AUTH_ADMINEMAIL` | `wenderson@belko.com.br` | Administrador principal (protegido) |
| `coordinator.auth.adminPassword` | `CLOUD_GAME_COORDINATOR_AUTH_ADMINPASSWORD` | vazio | Senha inicial do admin (só na criação) |
| `coordinator.auth.moderators` | `CLOUD_GAME_COORDINATOR_AUTH_MODERATORS` | `false` | Ativa o papel Moderador (aprova cadastros) |
| `coordinator.auth.sessionDays` | `CLOUD_GAME_COORDINATOR_AUTH_SESSIONDAYS` | `30` | Validade da sessão |
| `coordinator.auth.secureCookie` | `CLOUD_GAME_COORDINATOR_AUTH_SECURECOOKIE` | `false` | Use `true` atrás de proxy HTTPS |
| `coordinator.auth.queueReserveSec` | `CLOUD_GAME_COORDINATOR_AUTH_QUEUERESERVESEC` | `30` | Tempo para aceitar a vez na fila |
| `coordinator.covers.enabled` | `CLOUD_GAME_COORDINATOR_COVERS_ENABLED` | `true` | Busca capas dos jogos |
| `coordinator.covers.dir` | `CLOUD_GAME_COORDINATOR_COVERS_DIR` | `./data/covers` | Onde as capas ficam salvas (no Docker: `/data/covers`) |
| `coordinator.covers.url` | `CLOUD_GAME_COORDINATOR_COVERS_URL` | `https://thumbnails.libretro.com` | Acervo gratuito libretro-thumbnails |
| `coordinator.covers.retryDays` | `CLOUD_GAME_COORDINATOR_COVERS_RETRYDAYS` | `7` | Dias até procurar de novo um jogo sem capa |

| `coordinator.tournaments.enabled` | `CLOUD_GAME_COORDINATOR_TOURNAMENTS_ENABLED` | `true` | Módulo de torneios |
| `coordinator.tournaments.maxConcurrent` | `CLOUD_GAME_COORDINATOR_TOURNAMENTS_MAXCONCURRENT` | `2` | Partidas de torneio ao mesmo tempo |
| `coordinator.tournaments.joinTimeoutSec` | `CLOUD_GAME_COORDINATOR_TOURNAMENTS_JOINTIMEOUTSEC` | `180` | Prazo para entrar (depois: W.O.) |
| `coordinator.tournaments.replayDir` | `CLOUD_GAME_COORDINATOR_TOURNAMENTS_REPLAYDIR` | `./data/replays` | Replays (no Docker: `/data/replays`) |

## Torneios

Veja [06-torneios.md](06-torneios.md). Cada partida simultânea ocupa um worker: para 2 partidas
ao mesmo tempo, rode 2 workers. Jogos suportados: Ultimate Mortal Kombat 3, Super Bomberman 2 e
Super Mario Kart (SNES) — a ROM precisa ter um desses nomes.

## Capas dos jogos

As capas vêm do **libretro-thumbnails** (`thumbnails.libretro.com`), acervo público e gratuito, sem
chave de API. Cada imagem é baixada **uma única vez** e guardada em `covers.dir`, com o registro na
tabela `covers` do SQLite; depois disso é sempre servida pelo próprio servidor (`GET /api/covers`,
somente para contas aprovadas). Quando um worker se conecta, as capas da biblioteca são baixadas em
segundo plano. A busca tolera diferenças de nome (etiquetas como `(PT-BR)`, "The" no fim, `&`) e
prefere caixa → tela de título → captura, e região EUA/Mundo/Europa. Jogos sem capa no acervo
mantêm a arte gerada e são procurados de novo após `retryDays` (ou na hora, pelo botão
**Admin › Servidores › Buscar capas faltantes**).

## Produção

1. **VPS Linux** (o worker precisa de CPU dedicada; cada worker roda 1 jogo por vez).
   Para mais partidas simultâneas, rode mais workers apontando para o mesmo coordinator.
2. **HTTPS obrigatório**: coloque um proxy (Caddy/Nginx) na frente do coordinator
   (`:8000`, incluindo WebSocket em `/ws`) e defina `secureCookie: true`.
3. **WebRTC**: libere a porta UDP do worker (`CLOUD_GAME_WEBRTC_SINGLEPORT`, ex.: 8443) e
   configure `webrtc.iceIpMap` com o IP público. Para redes restritas, adicione um servidor TURN
   em `webrtc.iceServers`.
4. **Origem do WebSocket**: em produção defina `coordinator.origin.userWs` com o seu domínio.
5. Desative o modo debug (`CLOUD_GAME_COORDINATOR_DEBUG=false`).

## Backup

Todo o estado de contas fica num único arquivo SQLite (volume `belko-data`, arquivo `belko.db`), e as
capas baixadas ficam em `/data/covers` no mesmo volume.

```bash
# cópia consistente com o serviço rodando
docker compose exec cloud-game sh -c 'cp /data/belko.db /data/belko-$(date +%F).db'
```

Os saves dos jogos ficam em `~/.cr/save` dentro do contêiner do worker (ou no S3, se configurado).

## Administração no dia a dia

- **Aprovar cadastros**: Admin › Pendentes. O administrador recebe aviso em tempo real a cada novo
  cadastro (contador vermelho no menu).
- **Bloquear/desbloquear**: Admin › Usuários. Bloquear desconecta a pessoa na hora.
- **Esqueceu a senha**: Admin › Usuários › ícone de chave gera uma senha temporária para enviar à pessoa.
- **Delegar aprovações (futuro)**: defina `coordinator.auth.moderators: true`, reinicie e, em
  Admin › Usuários, mude o papel da pessoa para **Moderador**. Moderadores aprovam/recusam cadastros
  pendentes e moderam salas, mas não bloqueiam contas nem mudam papéis. Somente
  `wenderson@belko.com.br` é administrador.
- **Salas e fila**: Admin › Salas e fila mostra quem está jogando, quem espera e permite encerrar salas.
- **Auditoria**: Admin › Auditoria registra aprovações, bloqueios, trocas de papel e redefinições de senha.

## Testes

```bash
# Go (via Docker, sem instalar nada)
docker run --rm -v "$PWD":/src -w /src golang:1.27 go test ./pkg/auth ./pkg/store ./pkg/coordinator
```
