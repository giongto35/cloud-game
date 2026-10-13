# 07 — Subir no Coolify com auto deploy

Guia para publicar o Belko Arcade num servidor com [Coolify](https://coolify.io), com deploy
automático a cada `git push`. O arquivo usado é `docker-compose.coolify.yml` (produção).

## O que você precisa

- Uma VPS Linux x86_64 (mínimo 2 vCPU / 4 GB; o 1º build compila o GStreamer e leva ~15–25 min).
- Coolify instalado nela (`curl -fsSL https://cdn.coollabs.io/coolify/install.sh | bash`).
- Um domínio com registro **A** apontando para o IP da VPS (ex.: `arcade.seudominio.com.br`).
- O repositório no GitHub (`wendersonbelko/claude-game`, branch `belko/v1` ou `master`).

## 1. Firewall / portas

| Porta | Protocolo | Para quê |
|---|---|---|
| 80, 443 | TCP | Proxy HTTPS do Coolify (site + WebSocket) |
| **8443** | **UDP** | WebRTC (vídeo/áudio do jogo) — **libere no firewall da VPS e do provedor** |

Sem a 8443/UDP o site abre, mas o jogo não aparece.

## 2. Criar o recurso

1. Coolify → **Projects → New → Resource → Public/Private Repository (with GitHub App)**.
   Para repositório privado, conecte o GitHub App (ou use *Deploy Key*).
2. Escolha o repositório e a branch (`belko/v1` ou `master`).
3. **Build Pack: Docker Compose**.
4. **Docker Compose Location:** `/docker-compose.coolify.yml`.
5. **Save** → o Coolify lê os serviços (`cloud-game`, `xvfb`).

## 3. Domínio

Na aba do serviço `cloud-game`, em **Domains**, coloque `https://arcade.seudominio.com.br`.
O Coolify gera o certificado Let's Encrypt e roteia para a porta 8000 (WebSocket incluso).
Não publique a 8000 manualmente.

## 4. Variáveis de ambiente

Em **Environment Variables**:

| Variável | Exemplo | Obrigatória |
|---|---|---|
| `PUBLIC_IP` | IP público da VPS | sim |
| `PUBLIC_DOMAIN` | `arcade.seudominio.com.br` (sem `https://`) | sim |
| `BELKO_ADMIN_PASSWORD` | senha forte | recomendada (usada só na 1ª subida; vazia = gerada no log) |

## 5. Deploy

Clique em **Deploy**. Acompanhe em *Deployments*. Ao terminar, abra o domínio e entre com
`wenderson@belko.com.br` e a senha definida.

## 6. Auto deploy (a cada push)

1. No recurso → **Configuration → General**, ative **Auto Deploy**.
2. Com **GitHub App** o webhook é criado sozinho. Com repositório público/deploy key:
   - Recurso → **Webhooks → Manual Git Webhooks (GitHub)**, copie a *URL* e o *Secret*.
   - GitHub → repo → **Settings → Webhooks → Add webhook**: *Payload URL* = URL copiada,
     *Content type* = `application/json`, *Secret* = o secret, evento **Just the push event**.
3. Faça um `git push` na branch configurada: o Coolify faz *build* e troca o contêiner sozinho.

> O build é cacheado: depois do primeiro (lento), os deploys seguintes levam poucos minutos,
> exceto se o `Dockerfile` mudar nas etapas do GStreamer.

## 7. Jogos, cores e BIOS (volumes persistentes)

As ROMs **não** vão no Git. Elas ficam em volumes Docker que sobrevivem aos deploys:

- `belko-games` → `/usr/local/share/cloud-game/assets/games`
- `belko-cores` → `.../assets/cores` (os cores Libretro são baixados sozinhos)
- `belko-bios` → `.../libretro/system` (BIOS do PS1 etc.)
- `belko-data` → `/data` (contas SQLite, capas, catálogo, replays) — **faça backup**

Para enviar ROMs: pelo painel admin (upload), ou pelo terminal do Coolify no contêiner
`cloud-game`, ou do seu PC:

```bash
scp -r assets/games/snes root@IP_DA_VPS:/tmp/snes
docker cp /tmp/snes <container_cloud-game>:/usr/local/share/cloud-game/assets/games/
```

## 8. Backup

Backup do volume `belko-data` (principalmente `belko.db`):

```bash
docker run --rm -v <projeto>_belko-data:/data -v $PWD:/b alpine tar czf /b/belko-data.tgz -C /data .
```

Ou use *Scheduled Backups* de volume do Coolify.

## Problemas comuns

| Sintoma | Causa / solução |
|---|---|
| Site abre, tela preta / "conectando" | UDP 8443 bloqueada, ou `PUBLIC_IP` errado |
| Login não persiste | `secureCookie` exige HTTPS; acesse pelo domínio `https://` |
| WebSocket falha (403) | `PUBLIC_DOMAIN` diferente do domínio acessado |
| Build estoura memória | adicione swap de 4 GB na VPS |
| Deploy não dispara | Auto Deploy desligado ou webhook sem secret/URL correta (veja *Recent Deliveries* no GitHub) |
| Mais de uma partida/torneio ao mesmo tempo | cada worker roda 1 jogo; suba mais workers (veja 05-operacao.md) |

## Atualizar o app

Basta `git push`. Para versões de dependências: `go get -u ./... && go mod tidy`, e a versão do
Go/GStreamer nos `ARG` do `Dockerfile`.
