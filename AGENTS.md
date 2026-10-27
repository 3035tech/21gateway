# Instruções para agentes de IA

Este é o projeto novo 21 Mobile Gateway. Os diretórios legados são irmãos desta pasta; não os modifique como parte de tarefas deste projeto.

Antes de trabalhar, leia `docs/context.md`, `docs/architecture.md`, `docs/development.md` e `docs/status.md` e `docs/sms-single.md`. Consulte `docs/scope-questions.md` para funcionalidades ainda em definição. Esses documentos são a fonte compartilhada para Codex, Claude e Cursor. Atualize-os quando decisões, escopo ou implementação mudarem; evite manter versões contraditórias nas instruções de cada ferramenta.

## Requisitos do usuário

- Backend Go com MVC: controllers, services, models e DTOs quando necessário.
- PostgreSQL com GORM. Reutilizar bibliotecas consolidadas; não reinventar ORM, router, validação, tabelas ou autenticação.
- OpenAPI e Swagger UI gerados a partir dos contratos HTTP.
- Frontend React, TypeScript e Tailwind CSS. Material UI foi removido por decisão do usuário.
- Docker para backend e frontend; Docker Compose para desenvolvimento com PostgreSQL.
- Frontend estático preparado para AWS Amplify; não criar infraestrutura ou publicar sem uma tarefa de deployment.
- Cadastros de usuários e empresas previstos. Toda listagem deve ter filtros, paginação, seleção de itens por página e ordenação por clique no cabeçalho.
- Priorizar segurança, desempenho, reutilização e padrões de projeto com finalidade concreta.
- Evitar números mágicos e strings soltas para estados, erros e regras; usar tipos, enums por constantes Go e valores centralizados. Literais de protocolo, tags de schema e copy de apresentação não exigem enums artificiais.
- Erros internos não devem expor segredos, SQL, host do banco ou credenciais. Mensagens públicas centralizadas e códigos estáveis.

Para tarefas de frontend, consulte também `docs/screens.md` para separar telas atuais e propostas.

## Limites e validação

O envio imediato unitário já foi autorizado e implementado; consulte `docs/sms-single.md`. As regras completas de SMS e migração dependem das respostas da 21. Não invente equivalência de status, saldo, agendamento, MO ou idempotência. Registre hipóteses explicitamente. Não envie SMS reais em testes e não duplique envios para comparar plataformas.

Preserve o contrato legado, inclusive nomes, métodos, erros e IDs quando confirmados. O formato administrativo de erro não deve ser imposto ao legado.

Execute `go test -race ./...`, `go vet ./...` e build no backend; `npm run lint`, `npm run build` e testes pertinentes no frontend; `docker compose config --quiet` para Compose. Informe quais verificações não puderam ser executadas. Não declare Docker ou PostgreSQL testados com base apenas na validação YAML.

Não adicione abstrações ou serviços sem necessidade. Não implemente autenticação improvisada ou armazenamento de senhas reversível. Alterações de banco exigem migrações versionadas; não usar AutoMigrate em startup de produção.

## Regra obrigatória: separação por domínio

Toda nova implementação deve organizar as camadas por domínio. Empresas, usuários, credenciais da API, envios, callbacks e auditoria possuem controllers, services, models, DTOs e repositories próprios quando a responsabilidade existir. Não concentrar domínios em BackofficeController, BackofficeService ou arquivos genéricos backoffice.go.

No Go, utilizar arquivos como controllers/company.go, services/company.go, models/company.go, dtos/company.go e repositories/company.go. Cada controller depende do service do seu domínio; regras e consultas pertencem ao domínio responsável. O router e cmd/api apenas compõem dependências. Não criar camadas vazias para cumprir a convenção.

Compartilhar somente infraestrutura/coerência transversal: conexão/transação, sessão, autorização, validação HTTP, paginação, códigos de erro e helpers úteis. Compartilhar infraestrutura não autoriza reunir regras de vários domínios no mesmo service ou repository.

No frontend, organizar funcionalidades reais em features/companies, features/users, features/credentials, features/messages, features/callbacks e features/audit; componentes e infraestrutura comuns podem ser reutilizados. A prévia fictícia é um artefato de demonstração, não modelo de arquitetura para a integração real.

Antes de finalizar mudanças, verificar essa separação e registrar exceções concretas na arquitetura. Esta regra foi explicitamente solicitada pelo usuário e vale para todas as implementações futuras.

Consulte também `docs/backoffice.md` para contratos administrativos, perfis, sessões, credenciais externas, outbox e limites de homologação.

## Escopo respondido pela 21

Antes de implementar novas funcionalidades, ler `docs/scope-answers.md` e distinguir decisões confirmadas, propostas e pendências. A API V1 deve manter integrações e credenciais existentes; token do white label por cliente, individual/lotes, DLR/MO e consultas/créditos fazem parte do escopo. Agendamento foi indicado como provavelmente necessário. O painel novo é interno, focado em credenciais/configurações, com administrador único; perfis/telas extras atuais não são requisito. Não presumir contratos ausentes da Zoug, não remover endpoints sem evidência de desuso e não gerar novas senhas para os clientes durante a migração. A separação por domínio permanece obrigatória.

Implementações após o escopo respondido: consultar `docs/account-settings.md`, `docs/sms-batch.md` e `docs/framework-decision.md`. Runtime não deve voltar a token global nem a rate limit local; não substituir senhas legadas durante importação e não presumir normalização ou contratos de MO/consultas/agendamento.

Normalização de texto SMS autorizada e implementada em `docs/sms-normalization.md`. Somente a regra do nono dígito permanece pendente em `docs/backlog.md`; não inferir o corte 4/5 nem repetir implementação do texto.

Consultas de status legadas: ler `docs/legacy-status.md`. Guard legado compartilhado para Basic Auth/whitelist; consumo de polling é independente da outbox. Não inventar paginação/date, estados do fornecedor ou mensagens ausentes do manual.

Saída MO documentada em `docs/mo-callback.md`; evolução interna durável em `docs/mo-flow.md`. Não inventar identificação de conta, UUID/ID ou payload do fornecedor; definir retenção antes da ativação de produção.

Fluxo MO interno evoluído conforme `docs/mo-flow.md`, que prevalece sobre a etapa anterior do adaptador. Outbox/worker, conteúdo AES-GCM com MO_CONTENT_KEY independente e ingress opt-in desligado. Não presumir contrato do fornecedor. Retenção MO de 90 dias e expurgo protegendo jobs ativos implementados em `docs/operations.md`, dependentes da migração 009 e configuração do worker. Não expor conteúdo/fingerprints no painel/logs nem trocar chave sem recriptografia.

Consulta `/v1/consultmo` em `docs/legacy-mo-query.md`: filtro por dia/credencial, sem consumo, texto aberto somente para conta autenticada. Não presumir paginação ou retornar listas truncadas; limites de erro/volume ainda precisam de homologação.

Pesquisa de paginação/saldo/agendamento em `docs/contract-discovery.md`, solicitações em `docs/contract-requests.md`. Exemplos oficiais confirmam date e barra final, mas não paginação; 402 não é consulta de saldo. Não presumir campos/rotas nem enviar solicitações externas sem instrução explícita.

Estado atual e próximos passos em `docs/next-session.md`; consultar antes de retomar o trabalho.

Normalização de texto SMS autorizada e implementada conforme `docs/sms-normalization.md`: diacríticos/emoji removidos por bibliotecas, single/batch, limite mantido na origem. MO não é normalizado. Adiamento anterior do texto foi revogado; regra do nono dígito continua pendente e não deve ser inferida dos exemplos incompletos.

Operação implementada em `docs/operations.md`: conteúdo MO retido por 90 dias (confirmado pelo usuário), preservando entregas ativas; keyring v1/v2 e CLI de recriptografia com dry-run/CAS; diagnóstico opt-in; scripts age/pg_dump e benchmarks. Aplicar migração 009 antes de usar. Não trocar HMAC de deduplicação ao rotacionar cifra nem retirar leitores/chaves antigas antes da migração. Itens 3/4 seguem aguardando contratos/corte telefônico; o usuário autorizou retomar o trabalho.

Questionário final de 15/10/2026 incorporado em `docs/scope-answers.md` e `docs/migration-plan.md`. Ler o plano antes de qualquer migração: teste interno → piloto de produção → demais contas; outubro é prazo desejado, sem dia de corte. Pendências antigas/rollback não decididos; não descartar nem reenviar mensagens por suposição. Aceite inclui SMS e DLR/MO completos até o cliente. Retenção MO de 90 dias permanece; responsabilidade 3035/Mozhtech ainda sem divisão operacional.

Validação de aceite em `docs/acceptance-validation.md`: testes simulados executados; suíte PostgreSQL ampliada e executada localmente em 27/10/2026 com `go test -race ./...`. Não declarar integração real com fornecedor aprovada. DLR e MO bloqueiam jobs se URL cadastrada mudar; não redirecionar automaticamente jobs antigos.

Evidência de falhas de callback implementada em `docs/operations.md`: worker em JSON/stdout, IDs internos/status HTTP/códigos seguros, outcome_persisted separado de aceite HTTP. Nunca registrar URL, correlationId, telefone, mensagem, headers/corpos ou erro original. Não declarar recuperação antes do Finish, nem interpretar http_status=0 como status do cliente. Coleta/alertas e retenção de logs não configurados; 90 dias aplica-se ao conteúdo MO.

Segurança: ler `docs/reviews/security-assessment-2026-10-15.md`. Runtime deve usar NewAuthServiceWithProtection com AuthenticationAttempts compartilhado e orçamento configurável; nunca substituir por construtor isolado sem proteção. Preservar senhas legadas e política nova somente para usuários internos (15 caracteres/128 bytes/denylist pequena). Não confiar em XFF; validar proxy/NAT antes do deployment. 429 pré-autenticação e bloqueio temporário por conta precisam de homologação. Testes locais e PostgreSQL não equivalem a pentest.

Agendamento interno autorizado em 15/10/2026: ler `docs/sms-scheduling.md`. Aplicar migração010 antes de binários novos; opt-in SCHEDULER_ENABLED e chave independente nas duas aplicações. Preservar ID de aceite/correlationId e limites compartilhados; claimed pode ser retomado, dispatching/unknown nunca deve ser reenviado automaticamente. Texto agendado é exceção cifrada ao SMS sem conteúdo persistido, limpo ao terminar. Alto volume exige lotes/consultas limitadas; não afirmar TPS sem carga integrada. Dependência anterior de agendamento do fornecedor foi revogada pelo usuário.

Despacho SQS autorizado em 15/10/2026: ler docs/sqs-dispatch.md. Aplicar migrações até 014 antes de binários novos, mesmo com flag desligada. Scheduler Go/outbox/SQS Standard opt-in; somente referências sem dados pessoais. Preservar limites compartilhados, leases/CAS e proibição de repetir dispatching/unknown. Nunca alternar para worker local com lotes SQS abertos; drenar e parar workers conforme runbook. PostgreSQL local validado; SQS real e carga não validados. Não afirmar exactly-once ou TPS.
