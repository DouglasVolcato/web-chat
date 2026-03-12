# Documentação completa do Super Template

Este documento serve como guia para **vibe coding** e manutenção do template. Use-o como checklist sempre que alterar a estrutura.

---

## 0) Diretrizes obrigatórias para customizações por IA

Este template é preparado para personalização por ferramentas de IA utilizadas por pessoas não programadoras. Por isso, **o código gerado deve ser preciso, seguro e bem planejado**.

### Comunicação e planejamento

- Sempre se comunique em **português** com o usuário final.
- Faça perguntas quando qualquer requisito estiver ambíguo ou incompleto.
- Quebre toda solicitação em **tarefas pequenas**, com planejamento claro antes de executar.

### Arquitetura e operação

- **Serverless por padrão**: use arquitetura serverless; filas e storage externos só devem ser adicionados quando o cliente solicitar explicitamente.
- Não altere o fluxo de **assinaturas, pagamentos e geração de notas fiscais** sem pedido explícito do usuário.
- Mudanças de nome do projeto devem ser refletidas também nos arquivos de SEO/infra (ex.: `CNAME`, `robots.txt`, `sitemap.xml`, `manifest.json`).

### Boas práticas obrigatórias

- Contextos com timeout por endpoint.
- Rate limits sempre ativos para endpoints públicos.
- Abertura e fechamento correto de transações (usar `BeginTransaction` e `defer done()`).
- SQL sempre otimizado com **bind params** (sem string concatenation).
- Goroutines apenas quando realmente necessárias e sempre com controle de lifecycle.
- Atualize esta documentação sempre que houver novas regras de padrão de código ou regras de negócio.
- Avise o usuário sempre que novas variáveis forem adicionadas ao `.env.example`.

### Front-end

- Seguir o padrão **multi-page** em Go + HTMX.
- Usar classes do Bootstrap e evitar cores fixas para texto (não fixar branco/preto) por causa dos temas claro/escuro.

---

## 1) Configuração inicial

### ✅ Passo 1 — Atualize o `.env`

```bash
cp .env.example .env
```

Edite o `.env` e atualize **obrigatoriamente**:

- `GOOGLE_CLIENT_ID`
- `OPENAI_API_KEY`
- `JWT_SECRET`
- `POSTGRES_*`
- `API_URL`, `CLIENT_URL`, `HOST`

Se você alterar qualquer desses valores futuramente, **lembre-se de atualizar o `.env.example` também**.

---

## 2) Banco de dados

O template usa **Postgres** com três tabelas:

- `users`
- `user_chats`
- `user_chat_messages`

As migrations ficam em:

```
migrations
```

Sempre que **mudar a estrutura do banco**:

✅ Crie uma nova migration `.sql`
✅ Atualize este documento (`docs.md`)
✅ Atualize o `.env.example` caso tenha mudado variáveis

---

## 3) Migrations automáticas

As migrations rodam automaticamente ao iniciar o servidor.

Arquivo responsável:

```
models/migrations.go
```

Se alterar o caminho das migrations, atualize:

- `MIGRATIONS_DIR` no `.env.example`
- `docker-compose.yml`

---

## 4) Chat com IA

O chat usa a API da OpenAI.

Local principal:

```
controllers/chat_controller.go
```

Se mudar o modelo ou a estrutura de prompt:

✅ Atualize esse arquivo
✅ Atualize o README (se necessário)

---

## 5) Front-end

O front usa:

- Bootstrap
- HTMX

Os templates estão em:

```
presentation/views
```

### Padrões adicionais obrigatórios

- Este projeto é **multi-page** com renderização server-side em Go.
- Prefira HTMX para interações de atualização parcial.
- **Não fixe cores de texto** em preto/branco para manter compatibilidade com temas claro/escuro.

### Landing Page

```
presentation/views/landing/index.ejs
```

### Login

```
presentation/views/landing/login.ejs
```

### Dashboard e Chat

```
presentation/views/app
```

---

## 6) Docker

O `docker-compose.yml` inclui:

- `postgres` (banco)
- `app` (servidor Go)

### Subir o ambiente

```bash
docker compose up --build
```

---

## 7) Deploy automático (VPS + Nginx)

O `.env.example` já contém:

```
VPS_HOST
VPS_USER
VPS_PASSWORD
VPS_SSH_PORT
VPS_APP_PATH
VPS_NGINX_DOMAIN
```

Use essas variáveis em scripts de deploy para:

✅ configurar o Nginx automaticamente
✅ enviar o código
✅ rodar o docker compose via SSH

Se mudar a estrutura de deploy, atualize este arquivo e o `.env.example`.

---

## 8) Checklist de alteração

Sempre que mudar a estrutura, **faça o seguinte**:

✅ Atualize o `.env.example`
✅ Atualize o `README.md`
✅ Atualize o `docs.md`
✅ Ajuste migrations se necessário

---

## 9) Regras de negócio atuais

- Autenticação com Google e sessão via cookie.
- Área protegida com dashboard e chat com IA.
- Camada de pagamentos ativa; **não altere assinaturas, pagamentos ou geração de notas** sem solicitação explícita do usuário.

---

## 10) Observações finais

Este template foi desenhado para você remover o máximo de regras de negócio. Use-o como base para qualquer produto com:

- Autenticação
- Chat com IA
- Banco simples
- Docker pronto
