# Belko Arcade — Plano de Atualização

Este diretório contém o plano completo da transformação do CloudRetro em **Belko Arcade**:
uma plataforma privada de jogos retrô na nuvem, em português, com contas aprovadas pelo
administrador, multijogador online com papéis, fila de espera com transmissão ao vivo e
suporte completo a controle, teclado, mouse e toque.

| Documento | Conteúdo |
|---|---|
| [01-analise.md](01-analise.md) | Análise do código existente, problemas e riscos encontrados |
| [02-arquitetura.md](02-arquitetura.md) | Decisões técnicas, modelo de dados, protocolo, segurança |
| [03-design-ux.md](03-design-ux.md) | Identidade visual, design system, telas e fluxos |
| [04-roadmap.md](04-roadmap.md) | Fases e passos de execução, com checklist de status |
| [05-operacao.md](05-operacao.md) | Como rodar, implantar, administrar e fazer backup |

## Requisitos do cliente (fonte da verdade)

1. Toda a interface em **português (pt-BR)**.
2. Design **profissional**: visual, experiência de uso e funcionalidade.
3. **Registro de contas e login**.
4. **Somente `wenderson@belko.com.br` é administrador** e é ele quem aprova novas pessoas.
5. **Terreno preparado** para que, no futuro, outras pessoas também possam aprovar.
6. Foco em **design, UX, jogabilidade, integração com controles, mouse e teclado**.
7. **Multijogador online**.
8. **Fila de espera com transmissão (stream) do jogo** enquanto a pessoa aguarda.

## Decisões tomadas (resumo)

| Tema | Decisão |
|---|---|
| Nome do produto | **Belko Arcade** |
| Frontend | JavaScript puro (ES Modules, sem etapa de build), reorganizado em módulos de UI |
| Banco de dados | **SQLite** embarcado (driver `modernc.org/sqlite`, Go puro, sem CGO) |
| Senhas | **Argon2id** (`golang.org/x/crypto/argon2`) |
| Sessão | Cookie `HttpOnly` + `SameSite=Lax` (+ `Secure` com HTTPS), token aleatório guardado como hash |
| Cadastro | Aberto, porém toda conta nova nasce **pendente** até aprovação |
| Admin | `wenderson@belko.com.br`, criado automaticamente na 1ª inicialização; senha via variável de ambiente ou gerada e exibida uma única vez no log |
| Aprovadores futuros | Papel **Moderador** já existe no código com permissão de aprovar, desligado por configuração (`coordinator.auth.moderators: false`) |
| Multijogador | Papéis por sala: **Anfitrião**, **Jogador 1–4**, **Espectador**; validação no servidor |
| Fila | Fila global no coordinator, por conta; vaga reservada por 30 s para quem é chamado; espera assistindo uma sala ao vivo |
| Hospedagem | Docker (imagem única coordinator + worker) em VPS Linux com GPU opcional |
