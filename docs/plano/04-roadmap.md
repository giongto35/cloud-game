# 04 — Roadmap de execução

Legenda: ☐ pendente · ☑ concluído

## Fase 0 — Base
- ☑ 0.1 Documentar análise, arquitetura, design e roadmap (`docs/plano/`)
- ☑ 0.2 Ambiente de build/teste via Docker (`golang:1.27`)
- ☑ 0.3 Corrigir *panic* por índice de jogador inválido no worker (S2)

## Fase 1 — Contas e aprovação
- ☑ 1.1 `pkg/store`: SQLite, migrações, usuários, sessões, preferências, auditoria
- ☑ 1.2 `pkg/auth`: Argon2id, sessões, papéis/permissões, limitador, admin inicial
- ☑ 1.3 Configuração `coordinator.auth` (caminho do banco, cookie seguro, moderadores)
- ☑ 1.4 Rotas `/api/auth/*`, `/api/account/*`
- ☑ 1.5 `/ws` exige sessão aprovada; `/` redireciona para `/entrar`
- ☑ 1.6 Testes de store e auth

## Fase 2 — Novo visual e experiência
- ☑ 2.1 Tokens, base CSS, ícones SVG, componentes (botões, campos, modais, toasts, abas)
- ☑ 2.2 Página Entrar/Criar conta/Aguardando aprovação
- ☑ 2.3 Shell do app: barra superior, navegação, menu da conta
- ☑ 2.4 Biblioteca: busca, filtros, capas geradas, recentes
- ☑ 2.5 Tela de jogo: palco, HUD, etapas de carregamento, menu do jogo
- ☑ 2.6 Configurações em abas
- ☑ 2.7 Todos os textos em pt-BR (`i18n.js`)

## Fase 3 — Controles e jogabilidade
- ☑ 3.1 Controle no mapeamento padrão W3C, troca automática para o controle ativo
- ☑ 3.2 Remapeamento de teclado e controle por captura; zona morta; modo analógico
- ☑ 3.3 Navegação da interface pelo controle (foco espacial)
- ☑ 3.4 Teclado/mouse (pointer lock) para cores que pedem; atalhos seguros
- ☑ 3.5 Controle virtual de toque redesenhado
- ☑ 3.6 Indicador de conexão e estatísticas (F8)

## Fase 4 — Multijogador
- ☑ 4.1 Papéis na sala (anfitrião, jogadores 1–4, espectador) validados no coordinator
- ☑ 4.2 Worker ignora input de espectadores
- ☑ 4.3 Painel de jogadores: ocupar vaga, mover, expulsar
- ☑ 4.4 Ações restritas ao anfitrião (salvar, carregar, reiniciar)
- ☑ 4.5 Chat da sala com mensagens de sistema
- ☑ 4.6 Convite por link

## Fase 5 — Fila com transmissão
- ☑ 5.1 Fila por conta com tolerância a reconexão
- ☑ 5.2 Reserva de worker por 30 s para quem é chamado
- ☑ 5.3 Estado do lobby (salas ao vivo, servidores livres) em tempo real
- ☑ 5.4 Assistir enquanto espera; modal "Sua vez chegou!"
- ☑ 5.5 Tela "Ao vivo"

## Fase 6 — Administração
- ☑ 6.1 API admin: usuários, aprovação, papéis, bloqueio, redefinição de senha
- ☑ 6.2 Visão geral: servidores, salas, fila, online; encerrar sala
- ☑ 6.3 Auditoria
- ☑ 6.4 Aviso em tempo real de novos cadastros para quem pode aprovar
- ☑ 6.5 Painel admin no frontend

## Fase 7 — Produção
- ☑ 7.1 Docker: volume de dados e variáveis de ambiente
- ☑ 7.2 Testes Go e smoke test HTTP/WS
- ☑ 7.3 Guia de operação (`05-operacao.md`) e README atualizado

## Fase 8 — Capas dos jogos
- ☑ 8.1 `pkg/covers`: busca no libretro-thumbnails (gratuito, sem chave) com correspondência tolerante de nomes
- ☑ 8.2 Imagens salvas no servidor (disco + tabela `covers`), resultado negativo também em cache
- ☑ 8.3 `GET /api/covers` (contas aprovadas), pré-carregamento ao conectar worker
- ☑ 8.4 Capas na biblioteca e no "Ao vivo"; estatísticas e "buscar faltantes" no admin

## Fase 9 — Visual retrô e catálogo
- ☑ 9.1 Visual synthwave / Outrun 1984 igual ao belko.com.br: paleta, Orbitron + Rajdhani + Share Tech Mono, grade em perspectiva, scanlines, brilho neon (`tokens.css`, `retro.css`)
- ☑ 9.2 `pkg/meta`: gênero, ano, desenvolvedora, publicadora e nº de jogadores vindos do banco público libretro-database (sem chave de API), guardados na tabela `game_meta`
- ☑ 9.3 Categorias em pt-BR (Ação, RPG, Luta, Corrida…), palpite por título e dados manuais para homebrew e DOS
- ☑ 9.4 `GET /api/meta`; carga em segundo plano ao conectar um worker; "Atualizar catálogo" no painel admin
- ☑ 9.5 Biblioteca: filtros por sistema, categoria, década e jogadores; ordenação; busca por nome, gênero, ano e desenvolvedora; visão em grade ou por categoria; filtros lembrados no navegador
- ☑ 9.6 PS1: pasta de BIOS (`assets/bios`) e overscan; correção do ajuste de tela ao sair da tela cheia

## Fase 10 — Redesenho pixel-art e catálogo estilo streaming
- ☑ 10.1 Nome do produto passa a ser apenas **Arcade**
- ☑ 10.2 Novo visual de fliperama em pixel-art (Press Start 2P + VT323, bordas grossas, sombras duras, sem cantos arredondados) — `tokens.css`, `retro.css`
- ☑ 10.3 Catálogo estilo streaming: destaque do dia, carrosséis (continue jogando, minha lista, multijogador, um por gênero e por console), menu enxuto com Gêneros e Consoles, busca e página de resultados — `catalog.css`, `views/library.js`
- ☑ 10.4 Torneios em árvore: rodadas em colunas ligadas por linhas, placar por jogador, vencedor destacado e campeão no fim
- ☑ 10.5 Foto de perfil por conta (`PUT/DELETE /api/account/avatar`, `GET /api/avatars/{id}`); imagens são validadas, cortadas em quadrado e recodificadas em PNG (`pkg/imgutil`)
- ☑ 10.6 Administração troca ou adiciona a capa de qualquer jogo (`PUT/DELETE /api/admin/covers`) e volta à capa automática quando quiser
- ☑ 10.7 Correção: o número de jogadores do catálogo vinha sem aspas nos bancos de SNES e era ignorado

## Verificação (11/10/2026)

- Testes Go: `pkg/auth`, `pkg/coordinator` (fila, reservas, vagas, anfitrião, fim de sala sem jogadores — também com `-race`), `pkg/config`, `pkg/com`, `pkg/games`, `pkg/network`.
- Imagem Docker completa (coordinator + worker com GStreamer) compilada.
- Testes de ponta a ponta com dois navegadores reais contra a stack completa:
  cadastro → pendente → aprovação → acesso; admin joga; segunda conta entra na fila, assiste ao vivo,
  conversa no chat, ocupa a vaga de Jogador 2, volta a assistir, é chamada quando o servidor libera e joga;
  anfitrião salva (F2), convidado entra pelo link, é impedido de salvar, é removido e não consegue voltar;
  controles de toque em celular na horizontal.

## Próximos passos sugeridos

- HTTPS + domínio próprio e servidor TURN para jogar fora da rede local.
- Mais workers para partidas simultâneas.
- Capas reais dos jogos (opcional: arquivo de imagem com o mesmo nome da ROM).
- Envio de e-mail (aviso de aprovação e recuperação de senha) quando houver um provedor SMTP.
