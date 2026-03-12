# Super Template

Template para iniciar projetos web com **Go + Chi**, **Postgres**, **Bootstrap**, **HTMX** e **ChatGPT**.
Ele já vem com landing page, login Google, área autenticada e chat com IA.

## 🧭 Diretrizes para customizações por IA

Este repositório foi pensado para ser customizado por ferramentas de IA utilizadas por pessoas não programadoras. Para manter o código seguro e consistente:

- Sempre converse em **português** com o usuário final.
- Faça perguntas sempre que algo estiver incompleto ou ambíguo.
- Planeje as mudanças em tarefas pequenas antes de executar.
- **Serverless por padrão**: use filas ou storage externos apenas se o cliente solicitar explicitamente.
- Não altere o fluxo de **assinaturas, pagamentos e geração de notas** sem pedido explícito.
- Mudanças de nome do projeto devem refletir também em arquivos de SEO/infra (ex.: `CNAME`, `robots.txt`, `sitemap.xml`, `manifest.json`).
- Sempre avise quando novas variáveis forem adicionadas ao `.env.example`.

## ✅ O que este template inclui

- Landing page genérica e editável.
- Login com Google (OAuth via Google Identity Services).
- Área autenticada (dashboard + chat).
- Chat com IA usando a API da OpenAI.
- Rate limits, JWT e cookies seguros já configurados no servidor Chi.
- Controle de transações no acesso ao banco.
- Banco com apenas: **users**, **user_chats**, **user_chat_messages**.
- Migrations automáticas ao iniciar o servidor.
- Docker com Postgres junto da aplicação.
- Conexão Postgres configurável via `.env` e migrations automáticas.
- Bootstrap e HTMX no front-end.

---

## ⚙️ Como rodar (passo a passo)

### 1) Copie o arquivo de ambiente

```bash
cp .env.example .env
```

> **Importante:** Atualize todos os valores do `.env` antes de rodar.

### 2) Configure o `.env`

Campos essenciais:

- `GOOGLE_CLIENT_ID` → Client ID do Google.
- `OPENAI_API_KEY` → Chave da API da OpenAI.
- `JWT_SECRET` → Segredo para assinar tokens.
- `POSTGRES_*` → Dados do banco.
- `API_URL`, `CLIENT_URL`, `HOST` → URLs da aplicação.

> Se você adicionar novas variáveis, atualize o `.env.example` e informe o usuário.

### 3) Suba o Docker

```bash
docker compose up --build
```

A aplicação sobe em `http://localhost:${PORT}`.

---

## 🧠 Chat com IA

O chat usa o SDK da OpenAI e o modelo `GPT-4o mini` por padrão.
Você pode alterar o modelo no arquivo:

```
controllers/chat_controller.go
```

---

## 🧱 Banco de Dados

**Tabelas existentes:**
- `users`
- `user_chats`
- `user_chat_messages`

As migrations são executadas automaticamente na inicialização.
Os arquivos ficam em:

```
migrations
```

---

## 🚀 Deploy automático em VPS

O `.env.example` inclui variáveis para deploy automatizado em VPS:

- `VPS_HOST`
- `VPS_USER`
- `VPS_PASSWORD`
- `VPS_SSH_PORT`
- `VPS_APP_PATH`
- `VPS_NGINX_DOMAIN`

Você pode usar essas variáveis em scripts de deploy para configurar o Nginx e fazer o deploy por comando.

---

## 📂 Estrutura do projeto

```
controllers
models
migrations
presentation
docker-compose.yml
```

---

## ✅ Próximos passos para customizar

- Ajuste a landing page (`presentation/views/landing/index.ejs`).
- Renomeie o projeto e textos.
- Crie novas tabelas e migrations.
- Adicione regras de negócio.

---

## 🤖 Como pedir alterações para a IA

Para obter respostas precisas, peça mudanças de forma objetiva:

1. Explique o objetivo principal.
2. Liste regras de negócio e o que **não** deve ser alterado (ex.: pagamentos/assinaturas).
3. Informe restrições técnicas (serverless, HTMX, Bootstrap, etc.).
4. Peça para dividir a implementação em tarefas pequenas e confirmar dúvidas antes de executar.

---

## 🎨 Como trocar a logo do site

- Substitua `presentation/public/icons/logo.png` (logo do navbar).
- Substitua `presentation/public/icons/favicon.svg` (ícone do navegador).
- Caso mude o nome do projeto, atualize também `presentation/public/manifest.json`, `presentation/public/robots.txt`, `presentation/public/sitemap.xml` e `presentation/public/CNAME`.

---

## 🧪 Testes

Você pode rodar o binário localmente com:

```bash
cd app
GOOS=linux CGO_ENABLED=0 go build -o binaryApp
```

---

## 🤝 Licença

Use livremente como base para novos projetos.
