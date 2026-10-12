# 02 — Arquitetura e decisões técnicas

## 1. Visão geral da nova arquitetura

```
                ┌──────────────────────────── Coordinator ────────────────────────────┐
Navegador ─HTTP─▶ /entrar, /api/auth/*, /api/account/*, /api/admin/*   (pkg/auth + pkg/store/SQLite)
          ─WS───▶ /ws  (exige sessão aprovada) ─▶ Hub: usuários, lobby, fila, salas, chat
                └──────────────────────────────────────▲──────────────────────────────┘
                                                       │ WS /wso
                                                    Worker(s) ── WebRTC ──▶ Navegador
```

Novos pacotes Go:

| Pacote | Função |
|---|---|
| `pkg/store` | Acesso ao SQLite: migrações, usuários, sessões, preferências, auditoria |
| `pkg/auth` | Regras de negócio: cadastro, login, sessões, papéis/permissões, limitador de tentativas, admin inicial |
| `pkg/coordinator/httpapi.go` | Rotas HTTP JSON de autenticação, conta e administração |
| `pkg/coordinator/lobby.go` | Fila, reservas de worker, estado do lobby e das salas, chat |
| `pkg/covers` | Capas dos jogos: busca no libretro-thumbnails e cache permanente em disco + SQLite |

## 2. Banco de dados (SQLite)

Arquivo único (padrão `./data/belko.db`, configurável). WAL ativado, `foreign_keys=ON`.

```sql
users(id INTEGER PK, email TEXT UNIQUE COLLATE NOCASE, name TEXT, password_hash TEXT,
      role TEXT  -- 'admin' | 'moderator' | 'user'
      status TEXT -- 'pending' | 'approved' | 'rejected' | 'blocked'
      created_at, updated_at, approved_at, approved_by, last_login_at)
sessions(token_hash TEXT PK, user_id FK, created_at, expires_at, last_seen_at, ip, user_agent)
preferences(user_id PK FK, data TEXT /* JSON: perfis de controle, vídeo, áudio */, updated_at)
audit_log(id PK, actor_id, action TEXT, target_id, details TEXT, created_at)
schema_migrations(version INTEGER PK)
```

## 3. Autenticação e autorização

- **Senha**: Argon2id (m=64 MiB, t=1, p=4, sal 16 bytes), formato PHC (`$argon2id$v=19$...`). Mínimo 8 caracteres.
- **Sessão**: token de 32 bytes aleatórios no cookie `belko_session`; no banco fica apenas o SHA-256.
  Validade 30 dias, renovada a cada uso (com escrita no máximo a cada 5 min).
- **CSRF**: `SameSite=Lax` + toda rota que altera estado exige `Content-Type: application/json`
  e `Origin` igual ao host.
- **Limite de tentativas**: login e cadastro limitados por IP (balde de fichas em memória: 10/min).
- **Admin inicial**: na inicialização, se `wenderson@belko.com.br` não existir, é criado com papel
  `admin`/`approved`. Senha: `CLOUD_GAME_COORDINATOR_AUTH_ADMINPASSWORD`; se ausente, gera senha
  aleatória e imprime **uma única vez** no log. O admin principal não pode ser rebaixado, bloqueado
  nem excluído.
- **Permissões** (por papel, definidas em `pkg/auth/roles.go`):

| Permissão | admin | moderator | user |
|---|:-:|:-:|:-:|
| `play` (jogar, assistir, entrar na fila) | ✔ | ✔ | ✔ |
| `users.approve` (aprovar/recusar cadastros) | ✔ | ✔¹ | |
| `users.manage` (bloquear, papéis, redefinir senha) | ✔ | | |
| `rooms.moderate` (encerrar salas, expulsar, ver servidores) | ✔ | ✔¹ | |
| `admin.view` (painel administrativo) | ✔ | ✔¹ | |

¹ Somente se `coordinator.auth.moderators: true`. Desligado por padrão, conforme o pedido
("de início apenas o admin, mas deixe preparado o terreno").

O papel `admin` é exclusivo da conta principal: a API recusa promover qualquer outra conta a
administrador (`admin_reserved`).

- **WebSocket**: `/ws` valida o cookie **antes** do upgrade; contas não aprovadas recebem 401/403.
- **Páginas**: `/` sem sessão → redireciona para `/entrar`. Conta pendente vê a tela de espera.

## 4. Rotas HTTP

| Método | Rota | Descrição |
|---|---|---|
| POST | `/api/auth/register` | `{name,email,password}` → cria conta pendente |
| POST | `/api/auth/login` | `{email,password}` → define cookie |
| POST | `/api/auth/logout` | encerra a sessão |
| GET | `/api/auth/me` | dados da conta atual + permissões |
| PUT | `/api/account/profile` | altera o nome |
| PUT | `/api/account/password` | `{current,new}` |
| GET/PUT | `/api/account/preferences` | preferências (controles, vídeo, áudio) |
| GET | `/api/admin/users?status=` | lista usuários |
| POST | `/api/admin/users/{id}/{approve,reject,block,unblock}` | moderação |
| PUT | `/api/admin/users/{id}/role` | `{role}` |
| POST | `/api/admin/users/{id}/reset-password` | gera senha temporária |
| GET | `/api/admin/overview` | servidores, salas, fila, usuários online |
| POST | `/api/admin/rooms/{rid}/close` | encerra uma sala |
| GET | `/api/admin/audit` | histórico de ações |

## 5. Protocolo WebSocket — novos pacotes

Códigos existentes mantidos. Novos (faixa 12x–13x, usuário ⇄ coordinator):

| Código | Nome | Direção | Payload |
|---|---|---|---|
| 4 | `InitSession` (estendido) | S→C | `{wid, ice, games, you:{id,name,role,perms}, busy}` |
| 104 | `StartGame` (estendido) | C→S | `+ spectator: bool` |
| 108 | `ChangePlayer` (validado) | C→S | `slot` (0–3) ou `-1` para espectador |
| 120 | `LobbyState` | S→C | `{rooms:[{id,game,system,players,spectators,slots_free}], workers:{total,free}, queue:{size}}` |
| 121 | `QueueJoin` | C→S | `{game}` |
| 122 | `QueueLeave` | C→S | — |
| 123 | `QueueState` | S→C | `{status:'none'|'waiting'|'called', position, size, game, expires_in}` |
| 125 | `RoomState` | S→C | `{room_id, game, host, members:[{id,name,slot,host,you}]}` |
| 126 | `ChatSend` | C→S | `{text}` |
| 127 | `ChatMessage` | S→C | `{from,name,text,ts,system}` |
| 128 | `RoomKick` | C→S | `{user}` |
| 129 | `RoomAssign` | C→S | `{user, slot}` (anfitrião move alguém de vaga) |
| 130 | `Kicked` | S→C | `{reason}` |
| 131 | `Notice` | S→C | `{kind, text, count}` (ex.: novos cadastros para o admin) |

## 6. Multijogador

- Cada sala tem até **4 vagas de jogador** (portas Libretro 0–3) e **espectadores ilimitados**.
- O **anfitrião** é quem criou a sala; se sair, o jogador mais antigo assume.
- Ao entrar por link: ocupa a primeira vaga livre; se não houver, entra como espectador.
- O coordinator valida: vaga ocupada por outra pessoa → recusa; espectador não envia input.
- O worker ignora input de quem tem índice fora de `0..3` (espectador = `-1`) — corrige o *panic* S2.
- **Salvar / carregar / reiniciar**: só anfitrião (ou admin). Expulsar e mover jogadores: anfitrião/admin.
- **Chat** da sala: até 300 caracteres, limite de 1 mensagem/0,5 s por pessoa; mensagens de sistema
  para entradas, saídas e trocas de vaga.

## 7. Fila de espera com transmissão

```
[Sem worker livre] ─▶ QueueJoin(jogo) ─▶ posição N ──(assiste uma sala ao vivo enquanto espera)
        worker liberado ─▶ cabeça da fila recebe QueueState{called, 30 s}
            ├─ Aceitar ─▶ reconecta; coordinator entrega o worker reservado ─▶ jogo inicia
            └─ Recusar/expirar ─▶ reserva cancelada ─▶ próximo da fila é chamado
```

- A fila é **por conta**, não por conexão: recarregar a página não perde a posição.
  A entrada só é removida se a conta ficar desconectada por mais de 30 s.
- Reserva: o worker liberado fica reservado para a conta chamada; nenhuma outra pessoa
  consegue iniciar jogo nele enquanto a reserva vale.
- Espera assistindo: o cliente se conecta à sala ao vivo como **espectador** (sem input),
  usando a mesma transmissão que a sala já envia a todos os peers.

## 8. Frontend

Módulos novos em `web/js/`:

| Módulo | Função |
|---|---|
| `i18n.js` | Dicionário pt-BR e função `t()` |
| `ui/dom.js`, `ui/toast.js`, `ui/modal.js`, `ui/icons.js` | Componentes base |
| `ui/nav.js` | Navegação espacial por controle/teclado na interface |
| `ui/covers.js` | Capas geradas por jogo |
| `views/library.js` | Biblioteca com busca, filtros e capas geradas |
| `views/live.js` | Salas ao vivo (assistir / entrar) |
| `views/game.js` | Palco do jogo, HUD, painel de jogadores e chat, menu do jogo |
| `views/queue.js` | Indicador e modal da fila |
| `views/settings.js` | Configurações (controles com remapeamento, vídeo, áudio, conta) |
| `views/admin.js` | Painel administrativo |
| `app.js` | Orquestrador: ciclo de vida da conexão (WS + WebRTC), reconexão, troca de worker, roteamento de entradas |
| `auth.js` | Página de entrar/criar conta |
| `http.js` | Cliente JSON da API com erros em pt-BR |

Reaproveitados com ajustes: `api.js`, `event.js`, `network/*`, `input/*` (teclado, controle
padrão W3C, toque redesenhado), `stream.js`, `screen.js`, `settings.js` (armazenamento), `log.js`.

## 9. Testes e qualidade

- Go: testes de `pkg/store`, `pkg/auth` (hash, sessão, permissões, admin inicial),
  fila/reserva e papéis da sala no coordinator, guarda de índice no worker.
- `go vet` e `go build ./...` via Docker (`golang:1.27`).
- Smoke test HTTP/WS com Node contra o coordinator real.
