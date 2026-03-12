# Checklist Go/No-Go (Arquitetura) — Chat 1:1 MVP

> Checklist curto para aprovação técnica antes de seguir desenvolvimento.

## 1) Decisões de arquitetura que precisam estar fechadas
- [ ] **Modelo de sessão definido (server-side real vs JWT stateless)**
  - Go: decisão documentada + impactos em segurança/expiração/logout alinhados.
- [ ] **Contrato de autorização por recurso fechado (chat/contatos/perfil)**
  - Go: matriz rota x ownership validada com exemplos negativos.
- [ ] **Modelo de dados 1:1 definitivo (conversa bilateral, contatos bilaterais)**
  - Go: constraints de unicidade e idempotência aprovadas.

## 2) Segurança mínima obrigatória antes de avançar features
- [ ] **CORS de produção seguro (sem wildcard com credentials)**
  - Go: configuração revisada e validada em staging.
- [ ] **CSRF ativo para rotas state-changing (incluindo HTMX)**
  - Go: teste de ataque simples bloqueado.
- [ ] **Rate limit por IP + usuário em auth, QR e mensagens**
  - Go: limites definidos e 429 padronizado.

## 3) PWA + Push (requisitos críticos de produto)
- [ ] **Regra “instalar antes de logar” aplicada em frontend e backend**
  - Go: bypass por chamada direta de API bloqueado.
- [ ] **Deep-link de push para chat correto com sessão expirada tratada**
  - Go: clique com app aberto/fechado funciona; sessão inválida retorna ao destino após login.
- [ ] **Fallback por navegador/plataforma para install prompt parcial**
  - Go: fluxo manual validado para Safari iOS e navegadores sem prompt.

## 4) Dados, retenção e consistência
- [ ] **Expiração de mensagens 24h garantida por filtro + purge job**
  - Go: mensagem expirada não aparece mesmo se job atrasar.
- [ ] **Exclusão de conta sem dados órfãos e sem quebra do modelo 1:1**
  - Go: política de exclusão/anonimização e impacto em contatos/chats definidos.
- [ ] **Backups alinhados com retenção e restore testado**
  - Go: restore drill executado e RPO/RTO conhecidos.

## 5) Observabilidade e operação
- [ ] **Logs estruturados com correlação Nginx ↔ app**
  - Go: `request_id` ponta a ponta e redaction de dados sensíveis.
- [ ] **Alertas acionáveis para API, DB, jobs e push**
  - Go: alertas com runbook e responsável on-call.
- [ ] **Runbooks críticos aprovados (OAuth, push, purge, rollback, restore)**
  - Go: simulado curto executado com evidência.

## 6) Qualidade e liberação
- [ ] **Suite mínima de testes críticos aprovada**
  - Go: QR one-time concorrente, authz/IDOR, expiração 24h, deep-link push, exclusão de conta.
- [ ] **Smoke pós-deploy automatizado com rollback objetivo**
  - Go: critérios de rollback documentados e testados.
- [ ] **Checklist de go-live assinado por Engenharia + Produto**
  - Go: decisão explícita de risco residual aceitável.

---

## Regra prática de decisão
- [ ] **GO**: todos os itens P0 acima atendidos em staging com evidência.
- [ ] **NO-GO**: qualquer falha em sessão/authz, gate PWA, expiração 24h, deep-link push ou rollback.
