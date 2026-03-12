# Definition of Done (DoD) por Feature — Chat 1:1 MVP

> Escopo: DoD objetivo para features MVP com foco em prontidão real de entrega.

## 1) Login Google

- [ ] **Backend pronto**
  - [ ] Callback OAuth valida `state` e rejeita mismatch.
  - [ ] Apenas identidade Google válida cria/atualiza sessão.
  - [ ] Fluxo bloqueia usuário soft-deletado.
- [ ] **Banco pronto**
  - [ ] Usuário persistido com `google_sub` único.
  - [ ] Timestamps (`created_at`, `updated_at`) consistentes.
- [ ] **Frontend pronto**
  - [ ] Botão “Entrar com Google” funcional.
  - [ ] Feedback claro para sucesso/falha/autorização negada.
- [ ] **Tratamento de erro pronto**
  - [ ] Erros OAuth mapeados para mensagens seguras (sem leak técnico).
- [ ] **Logs prontos**
  - [ ] Log de tentativa/sucesso/falha com `request_id` e `user_id` quando houver.
- [ ] **Segurança pronta**
  - [ ] Cookie de sessão com `HttpOnly`, `Secure`, `SameSite`.
  - [ ] Rotação/invalidade de sessão após login.
- [ ] **Testes prontos**
  - [ ] Unitários para validação de `state` e criação de sessão.
  - [ ] Integração do callback completo.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de sucesso/erro de login por causa.
- [ ] **Documentação mínima pronta**
  - [ ] Fluxo OAuth e variáveis de ambiente documentados.
- [ ] **Critérios de aceite atendidos**
  - [ ] Usuário novo entra e cai na área autenticada.
  - [ ] Usuário existente entra sem duplicar conta.

## 2) Gate de Instalação PWA

- [ ] **Backend pronto**
  - [ ] Rotas autenticadas rejeitam acesso sem sinal de instalação concluída.
  - [ ] Gate aplicado antes do login efetivo no app.
- [ ] **Banco pronto**
  - [ ] Estado mínimo de onboarding/instalação persistível por usuário (ou sessão temporária).
- [ ] **Frontend pronto**
  - [ ] Tela de gate com instruções desktop/mobile.
  - [ ] Fallback para navegadores sem `beforeinstallprompt`.
- [ ] **Tratamento de erro pronto**
  - [ ] Mensagem clara para “instalação não detectada”.
- [ ] **Logs prontos**
  - [ ] Eventos: exibiu gate, instalou, pulou por fallback.
- [ ] **Segurança pronta**
  - [ ] Flag de instalação validada server-side (não confiar só no cliente).
- [ ] **Testes prontos**
  - [ ] E2E cobrindo: não instalado (bloqueia) vs instalado (prossegue).
- [ ] **Observabilidade pronta**
  - [ ] Taxa de conversão gate -> instalado.
- [ ] **Documentação mínima pronta**
  - [ ] Matriz de comportamento por navegador/plataforma.
- [ ] **Critérios de aceite atendidos**
  - [ ] Usuário não instalado não consegue logar.
  - [ ] Usuário instalado segue para login.

## 3) Geração de QR Code

- [ ] **Backend pronto**
  - [ ] Endpoint gera token temporário one-time com expiração curta.
  - [ ] Token vinculado ao usuário autenticado.
- [ ] **Banco pronto**
  - [ ] Tabela/token com `expires_at`, `consumed_at`, `created_by_user_id`.
- [ ] **Frontend pronto**
  - [ ] Tela “Meu QR” com validade visível e ação de regenerar.
- [ ] **Tratamento de erro pronto**
  - [ ] Limite de geração excedido retorna erro amigável.
- [ ] **Logs prontos**
  - [ ] Log de emissão e expiração do token QR.
- [ ] **Segurança pronta**
  - [ ] Token não previsível (alta entropia) e uso único.
  - [ ] Rate limit por usuário/IP para gerar QR.
- [ ] **Testes prontos**
  - [ ] Unitário para TTL e unicidade.
  - [ ] Integração para persistência e expiração.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de QRs gerados/consumidos/expirados.
- [ ] **Documentação mínima pronta**
  - [ ] Regra de validade e one-time descrita.
- [ ] **Critérios de aceite atendidos**
  - [ ] QR válido pode ser lido uma única vez dentro do TTL.

## 4) Leitura de QR Code

- [ ] **Backend pronto**
  - [ ] Endpoint de consumo valida existência, validade e não consumo prévio.
  - [ ] Operação idempotente para retries do cliente.
- [ ] **Banco pronto**
  - [ ] Consumo atômico (evita dupla leitura concorrente).
- [ ] **Frontend pronto**
  - [ ] Scanner com fallback para entrada manual do código.
- [ ] **Tratamento de erro pronto**
  - [ ] Mensagens distintas: expirado, inválido, já utilizado.
- [ ] **Logs prontos**
  - [ ] Tentativas de consumo com status final.
- [ ] **Segurança pronta**
  - [ ] Rate limit agressivo contra brute-force de token.
- [ ] **Testes prontos**
  - [ ] Concorrência: duas leituras simultâneas do mesmo token.
- [ ] **Observabilidade pronta**
  - [ ] Alerta para picos de QR inválido.
- [ ] **Documentação mínima pronta**
  - [ ] Códigos de erro funcionais documentados.
- [ ] **Critérios de aceite atendidos**
  - [ ] Leitura válida cria vínculo esperado sem duplicidade.

## 5) Criação de Contato Bilateral

- [ ] **Backend pronto**
  - [ ] Serviço cria relação bilateral no mesmo fluxo transacional.
  - [ ] Impede auto-contato e duplicidade.
- [ ] **Banco pronto**
  - [ ] Constraint de unicidade do par canônico de usuários.
- [ ] **Frontend pronto**
  - [ ] Confirmação visual de contato adicionado para ambos os lados.
- [ ] **Tratamento de erro pronto**
  - [ ] “Contato já existe” tratado sem falha técnica.
- [ ] **Logs prontos**
  - [ ] Evento auditável de criação de vínculo.
- [ ] **Segurança pronta**
  - [ ] Autorização garante que só usuário autenticado cria vínculo do próprio usuário.
- [ ] **Testes prontos**
  - [ ] Integração cobrindo idempotência e corrida de criação.
- [ ] **Observabilidade pronta**
  - [ ] Métrica diária de contatos criados/falhados.
- [ ] **Documentação mínima pronta**
  - [ ] Regras de bilateralidade e idempotência registradas.
- [ ] **Critérios de aceite atendidos**
  - [ ] Após leitura válida, ambos aparecem na lista de contatos.

## 6) Lista de Chats

- [ ] **Backend pronto**
  - [ ] Endpoint retorna apenas chats do usuário autenticado.
  - [ ] Ordenação por última atividade.
- [ ] **Banco pronto**
  - [ ] Índices para leitura por usuário + ordenação.
- [ ] **Frontend pronto**
  - [ ] Lista renderizada com último texto e horário.
  - [ ] Estado vazio quando sem chats.
- [ ] **Tratamento de erro pronto**
  - [ ] Falha de carregamento com opção de tentar novamente.
- [ ] **Logs prontos**
  - [ ] Latência e volume de retorno logados.
- [ ] **Segurança pronta**
  - [ ] Sem exposição de chat alheio por enumeração de IDs.
- [ ] **Testes prontos**
  - [ ] Integração para escopo de autorização.
- [ ] **Observabilidade pronta**
  - [ ] p95/p99 de latência da listagem.
- [ ] **Documentação mínima pronta**
  - [ ] Contrato de paginação/limite definido.
- [ ] **Critérios de aceite atendidos**
  - [ ] Lista consistente em desktop e mobile.

## 7) Tela de Conversa

- [ ] **Backend pronto**
  - [ ] Carrega histórico 1:1 autorizado por participação no chat.
  - [ ] Não retorna mensagens expiradas.
- [ ] **Banco pronto**
  - [ ] Índices por `chat_id` + `created_at`.
- [ ] **Frontend pronto**
  - [ ] Bolhas de mensagem, timestamps e rolagem estáveis.
  - [ ] Estado vazio para conversa sem mensagens.
- [ ] **Tratamento de erro pronto**
  - [ ] Chat inexistente/não autorizado com UX clara.
- [ ] **Logs prontos**
  - [ ] Evento de abertura de conversa com `chat_id`.
- [ ] **Segurança pronta**
  - [ ] Checagem de ownership em toda leitura da conversa.
- [ ] **Testes prontos**
  - [ ] E2E de navegação lista -> conversa correta.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de erro 403/404 por rota de conversa.
- [ ] **Documentação mínima pronta**
  - [ ] Regras de autorização e expiração registradas.
- [ ] **Critérios de aceite atendidos**
  - [ ] Usuário só vê conversas em que participa.

## 8) Envio de Mensagem

- [ ] **Backend pronto**
  - [ ] Endpoint cria mensagem textual com validação de tamanho/blank.
  - [ ] `expires_at` definido para +24h no momento da criação.
- [ ] **Banco pronto**
  - [ ] Persistência de mensagem com FK íntegra para chat e autor.
- [ ] **Frontend pronto**
  - [ ] Composer com envio por botão/Enter e prevenção de duplo envio.
- [ ] **Tratamento de erro pronto**
  - [ ] Erros de validação exibidos inline.
- [ ] **Logs prontos**
  - [ ] Log de envio (sem conteúdo sensível completo em texto puro).
- [ ] **Segurança pronta**
  - [ ] CSRF e validação server-side obrigatória.
  - [ ] Rate limit por usuário para anti-spam.
- [ ] **Testes prontos**
  - [ ] Unitário de validação de payload.
  - [ ] Integração de criação e retorno da mensagem.
- [ ] **Observabilidade pronta**
  - [ ] Taxa de mensagens enviadas e falhas por motivo.
- [ ] **Documentação mínima pronta**
  - [ ] Limites de tamanho e regras de conteúdo documentados.
- [ ] **Critérios de aceite atendidos**
  - [ ] Mensagem enviada aparece na conversa sem duplicar.

## 9) Expiração de Mensagens em 24h

- [ ] **Backend pronto**
  - [ ] Leitura ignora mensagens vencidas.
  - [ ] Job de purge remove vencidas em lote com lock.
- [ ] **Banco pronto**
  - [ ] Índice por `expires_at` para purge eficiente.
- [ ] **Frontend pronto**
  - [ ] UI não exibe mensagens após expiração.
- [ ] **Tratamento de erro pronto**
  - [ ] Falha do job gera alerta e retentativa controlada.
- [ ] **Logs prontos**
  - [ ] Quantidade purgada por execução e duração do job.
- [ ] **Segurança pronta**
  - [ ] Sem endpoint que permita recuperar conteúdo expirado.
- [ ] **Testes prontos**
  - [ ] Teste temporal (clock controlado) para expiração.
  - [ ] Integração do purge por batches.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de backlog de mensagens expiradas pendentes.
- [ ] **Documentação mínima pronta**
  - [ ] SLA do purge e comportamento de consistência eventual.
- [ ] **Critérios de aceite atendidos**
  - [ ] Mensagem some do produto em até janela operacional definida.

## 10) Push com Conteúdo da Mensagem

- [ ] **Backend pronto**
  - [ ] Serviço envia push para subscriptions ativas do destinatário.
  - [ ] Não envia push para mensagem já expirada.
- [ ] **Banco pronto**
  - [ ] Subscriptions versionadas/ativas com vínculo por usuário/dispositivo.
- [ ] **Frontend pronto**
  - [ ] Fluxo de permissão e estado de inscrição claros.
- [ ] **Tratamento de erro pronto**
  - [ ] Endpoints inválidos (410/404) removidos automaticamente.
- [ ] **Logs prontos**
  - [ ] Resultado por envio: sucesso, retry, inválido, removido.
- [ ] **Segurança pronta**
  - [ ] Chaves VAPID em secret manager/env seguro.
  - [ ] Conteúdo em notificação tratado como dado pessoal.
- [ ] **Testes prontos**
  - [ ] Integração com mock de gateway Web Push.
- [ ] **Observabilidade pronta**
  - [ ] Taxa de entrega/falha por navegador.
- [ ] **Documentação mínima pronta**
  - [ ] Política de privacidade sobre preview em push.
- [ ] **Critérios de aceite atendidos**
  - [ ] Destinatário recebe push com texto da mensagem em cenário feliz.

## 11) Clique na Notificação Abrindo o Chat

- [ ] **Backend pronto**
  - [ ] Deep-link resolve `chat_id` e valida autorização do usuário logado.
- [ ] **Banco pronto**
  - [ ] Referência consistente entre payload (`chat_id`, `message_id`) e dados reais.
- [ ] **Frontend pronto**
  - [ ] Service Worker abre/foca app e navega para conversa correta.
- [ ] **Tratamento de erro pronto**
  - [ ] Se chat indisponível/expirado, redireciona para lista com aviso.
- [ ] **Logs prontos**
  - [ ] Evento de click da notificação + resultado de navegação.
- [ ] **Segurança pronta**
  - [ ] Não confiar apenas no payload do push para autorização final.
- [ ] **Testes prontos**
  - [ ] E2E cobrindo app fechado, app aberto e múltiplas abas.
- [ ] **Observabilidade pronta**
  - [ ] Taxa de click-to-open-success.
- [ ] **Documentação mínima pronta**
  - [ ] Contrato de payload para deep-link documentado.
- [ ] **Critérios de aceite atendidos**
  - [ ] Clique sempre leva ao chat correto quando usuário está autenticado.

## 12) Edição de Perfil

- [ ] **Backend pronto**
  - [ ] Endpoint atualiza nome com validação de tamanho/trim.
- [ ] **Banco pronto**
  - [ ] Coluna `name` atualizável com constraints mínimas.
- [ ] **Frontend pronto**
  - [ ] Formulário com confirmação visual de sucesso.
- [ ] **Tratamento de erro pronto**
  - [ ] Erros de validação exibidos no campo.
- [ ] **Logs prontos**
  - [ ] Evento de atualização de perfil sem logar PII desnecessária.
- [ ] **Segurança pronta**
  - [ ] CSRF obrigatório em submit.
- [ ] **Testes prontos**
  - [ ] Unitário da regra de validação.
  - [ ] Integração do fluxo salvar perfil.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de sucesso/falha de atualização de perfil.
- [ ] **Documentação mínima pronta**
  - [ ] Regras do campo nome documentadas.
- [ ] **Critérios de aceite atendidos**
  - [ ] Nome novo aparece imediatamente na UI relevante.

## 13) Exclusão de Conta

- [ ] **Backend pronto**
  - [ ] Endpoint executa deleção lógica/anônima com transação.
  - [ ] Sessões e subscriptions do usuário são revogadas.
- [ ] **Banco pronto**
  - [ ] Estratégia de soft-delete/anonymization consistente.
- [ ] **Frontend pronto**
  - [ ] Fluxo de confirmação forte (ação destrutiva explícita).
- [ ] **Tratamento de erro pronto**
  - [ ] Falha parcial não deixa conta em estado inconsistente.
- [ ] **Logs prontos**
  - [ ] Auditoria de exclusão com motivo técnico/resultado.
- [ ] **Segurança pronta**
  - [ ] Reautenticação recente ou confirmação robusta antes de excluir.
- [ ] **Testes prontos**
  - [ ] Integração cobrindo exclusão e bloqueio de novo acesso.
- [ ] **Observabilidade pronta**
  - [ ] Métrica de exclusão iniciada vs concluída vs falha.
- [ ] **Documentação mínima pronta**
  - [ ] Runbook de exclusão e efeitos em dados relacionados.
- [ ] **Critérios de aceite atendidos**
  - [ ] Usuário excluído não consegue acessar conta; dados seguem política definida.
