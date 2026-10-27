# Arquitetura

> Escopo respondido em 14/10/2026: consulte docs/scope-answers.md. Este documento descreve o código existente; token por cliente e whitelist já foram implementados; MO e compatibilidade V1 completa ainda exigem ajustes. Painel solicitado pela 21 é focado em credenciais/configurações com perfil administrativo único. Não confundir recursos extras implementados com requisitos confirmados.
## Regra obrigatória: separação por domínio

Cada domínio tem arquivos próprios nas camadas necessárias: controllers/company.go, services/company.go, models/company.go, dtos/company.go, repositories/company.go; equivalentes para usuários, credenciais, mensagens, callbacks, auditoria e resumo. Não criar BackofficeController/BackofficeService genéricos para concentrar vários domínios. Essa organização foi explicitamente solicitada pelo usuário e vale para novas implementações.

Controllers dependem do service do domínio e cuidam de DTO/HTTP. Services coordenam regras e persistência/integrações. Models descrevem entidades GORM; DTOs são os contratos externos, sem exposição de hashes ou modelos inteiros. Repositories contêm consultas do domínio. Não criar camadas vazias apenas por convenção.

O router e cmd/api compõem dependências. Conexão/transações, sessão, autorização, paginação, erros e helpers transversais podem ser compartilhados. Repositories reutilizam somente operações comuns de persistência e gravação de auditoria; regras e consultas ficam no domínio proprietário.

Frontend real em features/auth, companies, users, credentials, messages, schedules, callbacks, audit e overview. App coordena rotas e layout; components/admin e components/data-table reúnem UI compartilhada. O módulo demo é prévia fictícia e não define a organização da integração real.

Visibilidade administrativa de agendamentos reside em controller, service, DTO e repository de `scheduled_sms`; o resumo agrega indicadores de fila e MO no domínio `summary`. Router e cmd/api apenas compõem. Migração 014 adiciona índices de leitura por horário previsto e credencial. Não há exceções à separação por domínio. A credencial técnica não identifica a pessoa física que enviou a requisição.

O polish da Auditoria permanece em controller, service, DTO e repository de `audit`. A listagem junta o usuário atual para exibir seu nome, preservando `actor_id` como referência estável; filtros são parametrizados e a ordenação usa lista fechada. Nenhuma tabela, migração ou serviço novo.

## Persistência e contratos

PostgreSQL/GORM, pool limitado, migrações versionadas Gormigrate com transação/advisory lock, sem AutoMigrate de startup. Compose executa migration dedicada antes do processo Go que hospeda API e workers. Migrações novas preservam dados anteriores; arquivamento lógico mantém FKs e histórico.

API administrativa: Huma/Chi, schemas/OpenAPI e Problem Details com código estável. Erros e constantes tipados; causas internas não entram em resposta/log público. Filtros parametrizados, ordenação por whitelist e desempate estável. Paginação de 1 a 100 itens por página. Huma ParamWrapper/ParamReactor distingue filtro booleano ausente de false.

API SMS preserva sua tradução legada. Fluxo: Basic Auth → validação/rate limit → tentativa durável → HTTP white label → resultado durável → resposta. Empresa é registrada na tentativa, mantendo histórico mesmo se a credencial for arquivada. Conteúdo SMS não é persistido. Timeout pode significar aceite: nunca retry automático. Runtime usa limite compartilhado no PostgreSQL: requisição por credencial e orçamento agregado de mensagens.

## Segurança e sessões

Argon2id; verificação concorrente limitada, hash fictício para contas inexistentes. SCS mantém sessões no PostgreSQL; cookies HttpOnly/Secure em produção, oito horas máximas e trinta minutos de inatividade. Renovação no login, destruição no logout, lookup da conta/versionamento a cada request. Casbin controla perfis internos. APIcredentials/clientes não ganham sessão de painel.

CSRF com CrossOriginProtection da biblioteca padrão Go, origens exatas compartilhadas com CORS, somente no prefixo /admin. API externa e ingress de callback têm autenticação independente. HTTPS terminado pela infraestrutura. SameSite=None é configurável para hosting em sites distintos e exige Secure.

Usuários internos administram a operação global; podem existir vários administradores. Convite, recuperação, MFA e sessão lembrada estão separados no domínio de usuários e documentados em [admin-security.md](admin-security.md). Não é portal multiempresa para clientes. Credenciais externas são vinculadas à empresa. Empresas bloqueadas impedem envio por suas credenciais. Último admin protegido por lock transacional. Auditoria contém apenas identificadores/ação/data, sem corpos ou segredos.

## Callbacks e workers

PostgreSQL armazena delivery_events e callback_jobs como outbox. Evento/estado/job gravados atomicamente; worker reconcilia UUIDs associados depois da chegada do evento. Callback do cliente é GET legado. Lease/SKIP LOCKED, concorrência limitada, oito tentativas, backoff com jitter, HTTP pooled/timeout, sem redirects. Entrega pelo menos uma vez: cliente deve deduplicar.

safeurl bloqueia SSRF no dial, proxy desativado, HTTPS/443; saída IPv6 desabilitada. Bloqueio de conta/empresa impede entrega. Reprocessar callback não reenvia SMS. O contrato do white label não foi documentado: ingresso interno normalizado é opt-in e envio permanece com callback_level NONE. Consulte docs/backoffice.md para detalhes.

## Frontend e hospedagem

TanStack Query/Table, Axios com cookies/timeout/cancelamento; React Hook Form/Zod. Queries incluem parâmetros e são invalidadas após alterações; mudanças de sessão limpam dados administrativos. Rotas por domínio carregadas sob demanda. Falhas não mostram resultados antigos como consulta válida; diálogos nativos com foco/Escape.

Frontend estático para Amplify; VITE_API_URL é pública. Backend/API/workers no mesmo processo Go e PostgreSQL ficam fora do hosting. Docker multi-stage, binários sem root; Compose é local, com portas loopback. Recursos AWS não foram criados; consulte docs/aws.md. Sem política definida, não expurgar metadados nem implementar retenção presumida.

## Contas e lotes

Token por conta com AES-GCM, whitelist e configurações DLR/MO no domínio de credenciais; cliente HTTP reutiliza transport/pool com token isolado por requisição. Lote pertence ao domínio SMS, com DTO/adaptador próprio e atualização PostgreSQL parametrizada em conjunto. Novo limitador SendLimits compartilha reservas entre réplicas. Não enfileirar SMS para envio tardio sem decisão. Leia docs/account-settings.md, docs/sms-batch.md e docs/framework-decision.md.

Consulta V1 em domínio legacy_status, com LegacyGuard compartilhado e recibos transacionais separados da outbox. Detalhes em docs/legacy-status.md.

Saída MO preparada no adaptador HTTP de callbacks com model transitório/DTO próprios, sem camadas vazias ou contrato presumido. Reutiliza dispatch e transporte DLR. Ativação, tenancy, outbox e armazenamento de conteúdo pendentes conforme docs/mo-callback.md.

MO evoluído com domínio próprio, eventos/outbox em tabelas separadas, chave MO_CONTENT_KEY independente, ingresso interno opt-in e worker integrado. Aplicar migração 007; configuração, segurança e limites em docs/mo-flow.md. Cadastro da URL não ativa recebimento do fornecedor.

Consulta MO legada implementada sobre eventos locais conforme docs/legacy-mo-query.md, sem consumo/outbox alterados; contratos externos e homologação de limites seguem pendentes.

Operação evoluída conforme docs/operations.md: migração 009, retenção confirmada de conteúdo MO por 90 dias com proteção de pendências, chaves versionadas/CLI, diagnóstico opt-in, scripts de backup/restauração e benchmarks. Não confundir preparação com validação de banco/produção.

Validação de aceite ampliada em 15/10/2026: consultar `docs/acceptance-validation.md` para cobertura de contratos e regra de bloqueio de callbacks DLR/MO após troca da URL. A suíte PostgreSQL passou localmente em 27/10/2026; integração externa e carga permanecem pendentes. Mantida separação por domínio; sem camadas ou dependências novas.

Observabilidade de callbacks: adaptador compartilhado DLR/MO registra falhas HTTP com dados mínimos; helper do domínio de callbacks registra conclusão/persistência e recuperação. Worker configura slog JSON em stdout. Sem novas camadas, dependências ou tabelas; detalhes e limites em docs/operations.md.

Proteção de autenticação compartilha login_limits já existente (migração 003), domínio repository authentication e helper security para chave/peer IP; regras de API/backoffice em auth/user services. Camadas por domínio preservadas, sem SQL em controllers. Sem migrations novas. Limites/política/riscos em docs/reviews/security-assessment-2026-10-15.md.

Scheduler local no domínio scheduled_sms: model/repository/service/worker próprios; controller/DTO de SMS legados reaproveitados. Adaptador do worker reutiliza SMSService e IDs aceitos, sem nova tentativa/ID por execução. Migração010 e configurações em docs/sms-scheduling.md. Sem cron/framework novo; padrão de fila PostgreSQL já existente. Summary otimizado com agregações em quatro consultas e timeout cinco segundos.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migrações até 014 obrigatórias nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. PostgreSQL local validado em 27/10/2026; SQS real/carga ainda pendentes. Testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.
