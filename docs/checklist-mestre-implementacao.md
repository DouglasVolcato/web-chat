# Backlog Priorizado de Implementação — Chat 1:1 MVP (PO Técnico + Tech Lead)

> Stack: Go + chi + PostgreSQL + EJS + HTMX + Bootstrap + Nginx + VPS.
> Escopo: chat 1:1 estilo WhatsApp, login Google, PWA obrigatório antes do login, contatos por QR, mensagens texto com expiração 24h, push com conteúdo e deep-link para chat, perfil editável e exclusão de conta.

---

## Critério de priorização usado

- **P0 (MVP crítico):** bloqueia jornada principal ou risco operacional alto.
- **P1 (MVP importante):** agrega valor direto, mas não bloqueia fundação.
- **P2 (Pós-MVP):** melhora evolução/escala/UX avançada.

---

## Fase 1 — Fundação técnica

### Épico 1.1 — Plataforma base de execução

#### Feature 1.1.1 — Setup de infraestrutura VPS + Nginx + PostgreSQL
- **Nome:** Base de infraestrutura de produção.
- **Descrição:** Provisionar VPS, PostgreSQL, Nginx TLS e estrutura de ambiente (staging/prod).
- **Valor de negócio:** viabiliza entrega real e reduz risco de indisponibilidade no lançamento.
- **Dependências:** domínio DNS, credenciais de cloud/VPS.
- **Risco:** configuração insegura (TLS/firewall), drift entre ambientes.
- **Prioridade:** P0.
- **MVP:** sim (obrigatório).
- **Pode esperar:** autoscaling/horizontalização avançada (Pós-MVP).

**Histórias técnicas**
- [ ] Provisionar VPS com hardening básico (SSH key-only, usuário não-root, firewall).
- [ ] Configurar Nginx com HTTPS obrigatório e renovação de certificado.
- [ ] Instalar PostgreSQL com usuários separados (app vs administração).
- [ ] Definir variáveis de ambiente segregadas por ambiente.
- [ ] Criar pipeline simples de deploy com restart seguro.

**Definição de pronto**
- [ ] App responde em HTTPS com proxy Nginx.
- [ ] Banco acessível somente pela aplicação/rede autorizada.
- [ ] Staging e produção isolados com configs independentes.

#### Feature 1.1.2 — Estrutura do monólito por camadas
- **Nome:** Arquitetura handler/service/repo.
- **Descrição:** Organizar backend Go em camadas com contratos claros.
- **Valor de negócio:** acelera evolução do MVP com menor risco de regressão.
- **Dependências:** setup do repositório e padrão de projeto.
- **Risco:** acoplamento indevido entre camadas.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** modularização avançada por bounded context.

**Histórias técnicas**
- [ ] Definir estrutura de pastas e convenções.
- [ ] Padronizar retorno de erro e DTOs de entrada/saída.
- [ ] Implementar middlewares base (request_id, recovery, timeout, logging).

**Definição de pronto**
- [ ] Toda rota passa por pipeline de middlewares padrão.
- [ ] Dependência unidirecional handler → service → repo validada.

---

## Fase 2 — Autenticação

### Épico 2.1 — Login Google com sessão server-side

#### Feature 2.1.1 — OAuth Google
- **Nome:** Fluxo de autenticação federada.
- **Descrição:** Implementar start/callback/logout com validação de state e sessão segura.
- **Valor de negócio:** remove fricção de cadastro e acelera aquisição no MVP.
- **Dependências:** Fase 1 concluída, credenciais Google.
- **Risco:** falhas de callback/state; indisponibilidade do provedor.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** múltiplos provedores de login.

**Histórias técnicas**
- [ ] Implementar endpoints `/auth/google/start`, `/auth/google/callback`, `/auth/logout`.
- [ ] Validar identidade e criar/atualizar usuário.
- [ ] Emitir sessão server-side com cookie seguro.

**Definição de pronto**
- [ ] Usuário autentica e acessa app com sessão ativa.
- [ ] Logout invalida sessão corretamente.

#### Feature 2.1.2 — Controle de sessão e autorização básica
- **Nome:** Proteção de rotas privadas.
- **Descrição:** Restringir acesso por sessão e ownership de recurso.
- **Valor de negócio:** protege dados e evita acesso cruzado.
- **Dependências:** feature 2.1.1.
- **Risco:** IDOR/violação de privacidade.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** gestão avançada de dispositivos/sessões.

**Histórias técnicas**
- [ ] Middleware auth obrigatório nas rotas privadas.
- [ ] Guardas de autorização por `user_id`/participação da conversa.

**Definição de pronto**
- [ ] Usuário A não acessa recursos de B.

---

## Fase 3 — PWA

### Épico 3.1 — Instalação obrigatória antes do login

#### Feature 3.1.1 — Manifest + Service Worker básico
- **Nome:** PWA instalável.
- **Descrição:** Configurar manifest, ícones, SW e critérios de instalabilidade.
- **Valor de negócio:** habilita distribuição app-like e push no navegador.
- **Dependências:** Fase 1.
- **Risco:** inconsistência entre browsers (especialmente iOS).
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** cache offline avançado.

**Histórias técnicas**
- [ ] Publicar `manifest` com ícones e `start_url` corretos.
- [ ] Registrar SW com estratégia de atualização segura.
- [ ] Criar fallback offline mínimo.

**Definição de pronto**
- [ ] App instalável em navegadores-alvo.
- [ ] SW registrado sem quebrar fluxo web padrão.

#### Feature 3.1.2 — Gate de instalação pré-login
- **Nome:** Enforcement de instalação.
- **Descrição:** Bloquear login sem instalação confirmada (frontend + backend).
- **Valor de negócio:** cumpre requisito central de produto.
- **Dependências:** feature 3.1.1 + autenticação.
- **Risco:** falso negativo de detecção em browser com suporte parcial.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** UX de onboarding otimizada com experimentos A/B.

**Histórias técnicas**
- [ ] Tela de onboarding de instalação com fluxo manual por plataforma.
- [ ] Persistir estado de instalação por dispositivo.
- [ ] Validar pré-condição no backend antes de completar login.

**Definição de pronto**
- [ ] Sem instalação, login é bloqueado com mensagem clara.
- [ ] Com instalação, usuário segue para login sem fricção extra.

---

## Fase 4 — Contatos por QR

### Épico 4.1 — Descoberta fechada sem busca pública

#### Feature 4.1.1 — Geração e consumo de QR temporário
- **Nome:** Adição de contato via QR one-time.
- **Descrição:** Usuário gera QR temporário; outro usuário consome e ambos viram contatos.
- **Valor de negócio:** onboarding social sem diretório público.
- **Dependências:** autenticação + base de dados usuários.
- **Risco:** replay/brute force de token.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** convites alternativos (link/código curto avançado).

**Histórias técnicas**
- [ ] Endpoint para gerar QR com TTL curto.
- [ ] Endpoint de consumo transacional com idempotência.
- [ ] Criação bilateral de contatos com constraints de unicidade.

**Definição de pronto**
- [ ] QR válido cria contato bilateral uma única vez.
- [ ] QR expirado/consumido retorna erro controlado.

#### Feature 4.1.2 — UI de contatos e scanner
- **Nome:** Experiência de contatos por QR.
- **Descrição:** Lista de contatos + tela para gerar/escanear QR com fallback.
- **Valor de negócio:** transforma regra de negócio em fluxo usável.
- **Dependências:** feature 4.1.1 + frontend base.
- **Risco:** câmera indisponível em parte dos dispositivos.
- **Prioridade:** P1.
- **MVP:** sim.
- **Pode esperar:** scanner otimizado avançado.

**Histórias técnicas**
- [ ] Página de contatos com estado vazio e CTA.
- [ ] Fluxo de scanner com tratamento de permissão negada.
- [ ] Fallback sem câmera (entrada alternativa do token).

**Definição de pronto**
- [ ] Usuário consegue adicionar contato com câmera ou fallback.

---

## Fase 5 — Chat

### Épico 5.1 — Conversa 1:1 texto

#### Feature 5.1.1 — Lista de chats + timeline
- **Nome:** Núcleo de chat 1:1.
- **Descrição:** Exibir chats, abrir conversa e listar mensagens em ordem.
- **Valor de negócio:** entrega funcionalidade central do produto.
- **Dependências:** contatos por QR.
- **Risco:** performance de listagem e ordenação sob crescimento.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** recursos de presença/typing.

**Histórias técnicas**
- [ ] Endpoints de listagem de chats e mensagens paginados.
- [ ] Templates EJS/HTMX para sidebar e painel de conversa.
- [ ] Controle de autorização por participante.

**Definição de pronto**
- [ ] Dois contatos conseguem abrir conversa e ver histórico.

#### Feature 5.1.2 — Envio de mensagem de texto
- **Nome:** Envio e persistência de mensagens.
- **Descrição:** Permitir envio de texto com validação e idempotência.
- **Valor de negócio:** habilita comunicação efetiva entre usuários.
- **Dependências:** feature 5.1.1.
- **Risco:** duplicidade por retry e spam.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** edição/exclusão de mensagem pelo usuário.

**Histórias técnicas**
- [ ] Endpoint de envio com `client_message_id`.
- [ ] Validação de tamanho e conteúdo do texto.
- [ ] Atualização de UI incremental via HTMX.

**Definição de pronto**
- [ ] Mensagem enviada aparece no chat sem duplicar em retry.

#### Feature 5.1.3 — Expiração de mensagens em 24h
- **Nome:** Mensagens efêmeras.
- **Descrição:** Mensagens expiram em 24h e deixam de ser exibidas.
- **Valor de negócio:** diferencial de privacidade e requisito obrigatório.
- **Dependências:** feature 5.1.2 + jobs.
- **Risco:** retenção indevida por falha de purge.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** políticas de retenção configuráveis por usuário.

**Histórias técnicas**
- [ ] Persistir `expires_at` em criação.
- [ ] Filtrar expiradas em leitura.
- [ ] Job de purge físico com observabilidade.

**Definição de pronto**
- [ ] Mensagens expiradas não aparecem na API/UI.
- [ ] Job remove expiradas periodicamente com heartbeat.

---

## Fase 6 — Push

### Épico 6.1 — Notificação de nova mensagem com deep-link

#### Feature 6.1.1 — Registro de subscriptions push
- **Nome:** Cadastro de dispositivo para push.
- **Descrição:** Inscrever/desinscrever navegador/dispositivo no backend.
- **Valor de negócio:** permite notificação ativa e retorno ao app.
- **Dependências:** PWA e sessão autenticada.
- **Risco:** base suja com endpoints inválidos.
- **Prioridade:** P1.
- **MVP:** sim.
- **Pode esperar:** preferências granulares por dispositivo.

**Histórias técnicas**
- [ ] Endpoints subscribe/unsubscribe idempotentes.
- [ ] Persistência por usuário+dispositivo.
- [ ] Limpeza periódica de endpoints inválidos.

**Definição de pronto**
- [ ] Usuário com permissão ativa recebe status de subscription válido.

#### Feature 6.1.2 — Envio de push + clique abrindo chat correto
- **Nome:** Entrega e roteamento de notificações.
- **Descrição:** Notificar nova mensagem com preview e deep-link para conversa.
- **Valor de negócio:** aumenta retorno e tempo de resposta do usuário.
- **Dependências:** chat pronto + feature 6.1.1.
- **Risco:** deep-link incorreto, sessão expirada no clique.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** controles avançados de privacidade do preview.

**Histórias técnicas**
- [ ] Disparar push no envio de mensagem com payload mínimo.
- [ ] Service worker tratar click e focar/abrir app no chat alvo.
- [ ] Redirecionar para login e retornar ao destino quando sessão expirada.

**Definição de pronto**
- [ ] Push mostra conteúdo da mensagem por padrão.
- [ ] Clique leva ao chat correto com app aberto/fechado.

---

## Fase 7 — Configurações

### Épico 7.1 — Perfil e ciclo de vida da conta

#### Feature 7.1.1 — Edição de perfil
- **Nome:** Atualização de nome do perfil.
- **Descrição:** Permitir editar nome com validação e feedback imediato.
- **Valor de negócio:** personalização mínima do usuário.
- **Dependências:** autenticação e tela de settings.
- **Risco:** XSS se output/input não tratado.
- **Prioridade:** P1.
- **MVP:** sim.
- **Pode esperar:** avatar/status avançado.

**Histórias técnicas**
- [ ] Endpoint de update de nome com validação.
- [ ] Atualização parcial na UI (HTMX) com mensagens de sucesso/erro.

**Definição de pronto**
- [ ] Nome alterado com persistência e reflexo nas telas principais.

#### Feature 7.1.2 — Exclusão de conta
- **Nome:** Remoção de conta autoatendida.
- **Descrição:** Usuário exclui conta e encerra acesso imediatamente.
- **Valor de negócio:** conformidade e confiança do usuário.
- **Dependências:** sessão, push, contatos/chats.
- **Risco:** exclusão parcial deixando dados órfãos.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** fluxo de desativação temporária.

**Histórias técnicas**
- [ ] Endpoint de exclusão com confirmação forte.
- [ ] Revogar sessões e subscriptions push.
- [ ] Tratar referências relacionais conforme política de retenção.

**Definição de pronto**
- [ ] Conta excluída não autentica novamente com sessão antiga.
- [ ] Dados tratados conforme política definida.

---

## Fase 8 — Segurança/LGPD

### Épico 8.1 — Segurança mínima obrigatória de produção

#### Feature 8.1.1 — OWASP baseline + rate limit
- **Nome:** Controles de segurança de aplicação.
- **Descrição:** Implementar CSRF, validação de input, proteção XSS/SQLi e limites de abuso.
- **Valor de negócio:** reduz risco de incidente e vazamento de dados.
- **Dependências:** fundação técnica e rotas principais.
- **Risco:** exploração de endpoints críticos.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** WAF avançado e hardening extra.

**Histórias técnicas**
- [ ] CSRF em rotas state-changing incluindo HTMX.
- [ ] Sanitização/escape consistente em EJS.
- [ ] Queries parametrizadas + whitelist de filtros.
- [ ] Rate limit por IP/usuário em auth, QR e mensagens.

**Definição de pronto**
- [ ] Controles ativos e validados nos fluxos críticos.

#### Feature 8.1.2 — LGPD mínima aplicada
- **Nome:** Privacidade operacional do MVP.
- **Descrição:** Minimização de dados, retenção 24h, política publicada e trilha de atendimento.
- **Valor de negócio:** reduz risco regulatório e melhora confiança.
- **Dependências:** expiração de mensagens e exclusão de conta.
- **Risco:** retenção indevida e baixa transparência.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** automação avançada de direitos do titular.

**Histórias técnicas**
- [ ] Publicar política de privacidade específica.
- [ ] Registrar inventário mínimo de dados e terceiros.
- [ ] Definir e aplicar retenção técnica por domínio.

**Definição de pronto**
- [ ] Evidência de retenção/purge e documentação mínima disponível.

---

## Fase 9 — Observabilidade

### Épico 9.1 — Operação orientada por sinais

#### Feature 9.1.1 — Logs, métricas e alertas essenciais
- **Nome:** Telemetria base de produção.
- **Descrição:** Instrumentar app/Nginx/jobs com correlação e alertas.
- **Valor de negócio:** reduz MTTR e melhora confiabilidade.
- **Dependências:** rotas e jobs implementados.
- **Risco:** incidentes sem detecção precoce.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** tracing distribuído completo.

**Histórias técnicas**
- [ ] Logs JSON com `request_id` e redaction.
- [ ] Dashboards de API, DB, push e jobs.
- [ ] Alertas para 5xx, latência, falha de backup e jobs atrasados.

**Definição de pronto**
- [ ] Time recebe alerta acionável antes de impacto prolongado.

#### Feature 9.1.2 — Runbooks operacionais
- **Nome:** Resposta padronizada a incidentes.
- **Descrição:** Playbooks para OAuth, chat lento, push degradado, purge parado e restore.
- **Valor de negócio:** acelera recuperação em incidentes reais.
- **Dependências:** feature 9.1.1.
- **Risco:** resposta ad-hoc com erro humano.
- **Prioridade:** P1.
- **MVP:** sim.
- **Pode esperar:** automação de remediação avançada.

**Histórias técnicas**
- [ ] Criar runbook com sintomas, hipóteses, passos e rollback.
- [ ] Treinar time em simulado curto de incidente.

**Definição de pronto**
- [ ] Qualquer on-call consegue executar primeiros 15 minutos de diagnóstico.

---

## Fase 10 — Testes e Go-live

### Épico 10.1 — Qualidade e lançamento controlado

#### Feature 10.1.1 — Suíte de testes priorizada por risco
- **Nome:** Cobertura de qualidade do MVP.
- **Descrição:** Unitários, integração, E2E e segurança focados em fluxos críticos.
- **Valor de negócio:** evita regressões em funcionalidades essenciais.
- **Dependências:** features de produto concluídas.
- **Risco:** release com falhas ocultas em jornadas centrais.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** cobertura extensa de cenários long-tail.

**Histórias técnicas**
- [ ] Unitários para QR one-time, expiração 24h, exclusão e authz.
- [ ] Integrações com PostgreSQL real e migrações.
- [ ] E2E de jornada completa (PWA -> login -> QR -> chat -> push -> perfil/exclusão).
- [ ] Testes de segurança mínimos (CSRF/XSS/SQLi/IDOR/rate limit).

**Definição de pronto**
- [ ] Gate de qualidade bloqueia promoção quando fluxo crítico falha.

#### Feature 10.1.2 — Go-live com smoke + rollback
- **Nome:** Lançamento com controle de risco.
- **Descrição:** Executar smoke tests, validar backup/restore e habilitar rollback rápido.
- **Valor de negócio:** reduz impacto de falha no primeiro release.
- **Dependências:** observabilidade e testes ativos.
- **Risco:** indisponibilidade prolongada por rollback mal preparado.
- **Prioridade:** P0.
- **MVP:** sim.
- **Pode esperar:** estratégia blue/green completa.

**Histórias técnicas**
- [ ] Smoke pós-deploy (login, chats, QR, envio, push, jobs, backup status).
- [ ] Critérios objetivos de rollback documentados.
- [ ] Janela de hypercare com monitoramento reforçado (7 dias).

**Definição de pronto**
- [ ] Deploy só é concluído após smoke crítico aprovado.
- [ ] Rollback executável em tempo alvo acordado.

---

## Ordem recomendada de implementação (sequência executiva)

- [ ] **Ordem 01:** Fase 1 (Fundação técnica).
- [ ] **Ordem 02:** Fase 2 (Autenticação).
- [ ] **Ordem 03:** Fase 3 (PWA + gate obrigatório).
- [ ] **Ordem 04:** Fase 4 (Contatos por QR).
- [ ] **Ordem 05:** Fase 5 (Chat + expiração 24h).
- [ ] **Ordem 06:** Fase 6 (Push + deep-link).
- [ ] **Ordem 07:** Fase 7 (Configurações: perfil/exclusão).
- [ ] **Ordem 08:** Fase 8 (Segurança/LGPD baseline MVP).
- [ ] **Ordem 09:** Fase 9 (Observabilidade + runbooks).
- [ ] **Ordem 10:** Fase 10 (Testes finais e Go-live).

---

## Dependências críticas cruzadas (mapa rápido)

- [ ] Push depende de: PWA instalado + sessão + chat funcional.
- [ ] QR depende de: autenticação + modelo de contatos + transação no banco.
- [ ] Expiração 24h depende de: campo `expires_at` + filtro de leitura + job purge.
- [ ] Exclusão de conta depende de: sessão, contatos/chats, push subscriptions e política de retenção.
- [ ] Go-live depende de: observabilidade ativa + smoke + rollback testado.

---

## Riscos principais do programa e mitigação

- [ ] **Risco:** bloqueio de onboarding por suporte parcial de PWA em alguns browsers.
  - [ ] Mitigação: fluxo manual por plataforma + validação backend + mensagem clara.

- [ ] **Risco:** duplicidade/consistência em QR e criação de contatos.
  - [ ] Mitigação: token one-time + transação idempotente + constraints.

- [ ] **Risco:** retenção indevida de mensagens além de 24h.
  - [ ] Mitigação: filtro em leitura + job monitorado + alerta de backlog.

- [ ] **Risco:** falha de deep-link de push com sessão expirada.
  - [ ] Mitigação: preservar destino e retomar chat após login.

- [ ] **Risco:** release sem visibilidade operacional suficiente.
  - [ ] Mitigação: dashboards, alertas acionáveis, smoke e runbook antes do go-live.

---

## Recorte MVP vs Pós-MVP (resumo objetivo)

- [ ] **Entra no MVP (obrigatório):**
  - [ ] Login Google + sessão segura.
  - [ ] Gate de instalação PWA antes do login.
  - [ ] Contatos por QR temporário one-time.
  - [ ] Chat 1:1 texto com expiração 24h.
  - [ ] Push com conteúdo da mensagem e clique abrindo chat correto.
  - [ ] Perfil editável e exclusão de conta.
  - [ ] Segurança baseline, LGPD mínima, observabilidade essencial e backup diário.

- [ ] **Pode esperar (Pós-MVP):**
  - [ ] Presença/typing, mídias, grupos, chamadas.
  - [ ] Preferência de ocultar preview de push por usuário/dispositivo.
  - [ ] Escala horizontal avançada (fila/event bus dedicado).
  - [ ] Testes long-tail e automações operacionais avançadas.
