# Arquitetura

O diagnóstico temporário Zoug fica no domínio de mensagens: controller/service coordenam autorização, auditoria e empresa; o adaptador white label contém os dez casos HTTP fixos. O DTO administrativo não depende do adaptador. Sem tabela nova; logs JSON são a evidência transitória. Não há exceção à separação por domínio.

O novo modo de envio assíncrono por ID está descrito em [sms-send-queue.md](sms-send-queue.md). A API admite todos os SMS no PostgreSQL; `/worker` publica apenas `id_sms` na SQS v2 e processa o envio. O modo permanece desligado até provisionamento e corte controlado.

> Escopo respondido em 14/10/2026: consulte docs/scope-answers.md. Este documento descreve o código existente; token por cliente e whitelist já foram implementados; MO e compatibilidade V1 completa ainda exigem ajustes. Painel solicitado pela 21 é focado em credenciais/configurações com perfil administrativo único. Não confundir recursos extras implementados com requisitos confirmados.
## Regra obrigatória: separação por domínio

Cada domínio tem arquivos próprios nas camadas necessárias: controllers/company.go, services/company.go, models/company.go, dtos/company.go, repositories/company.go; equivalentes para usuários, credenciais, mensagens, callbacks, auditoria e resumo. Não criar BackofficeController/BackofficeService genéricos para concentrar vários domínios. Essa organização foi explicitamente solicitada pelo usuário e vale para novas implementações.

Controllers dependem do service do domínio e cuidam de DTO/HTTP. Services coordenam regras e persistência/integrações. Models descrevem entidades GORM; DTOs são os contratos externos, sem exposição de hashes ou modelos inteiros. Repositories contêm consultas do domínio. Não criar camadas vazias apenas por convenção.

O router e cmd/api compõem dependências. Conexão/transações, sessão, autorização, paginação, erros e helpers transversais podem ser compartilhados. Repositories reutilizam somente operações comuns de persistência e gravação de auditoria; regras e consultas ficam no domínio proprietário.

Frontend real em features/auth, companies, users, credentials, messages, schedules, callbacks, audit e overview. App coordena rotas e layout; components/admin e components/data-table reúnem UI compartilhada. O módulo demo é prévia fictícia e não define a organização da integração real.

Internacionalização transversal usa catálogos JSON por idioma no frontend e no pacote de erros públicos do backend. Componentes de cada domínio referenciam chaves; o seletor persiste `pt-BR`/`en-US`. Respostas administrativas localizam `detail` por código e `Accept-Language`; o middleware HTTP não altera `/v1`. O domínio de credenciais possui a validade opcional e a geração/cadastro manual de username e senha; autenticação e workers consultam `APICredential.ValidAt`. Migração 017 acrescenta `expires_at` sem alterar as contas existentes. A separação por domínio permanece: o middleware de idioma e o catálogo de erros são infraestrutura transversal, sem regras de credenciais.

Migração 020 separa o token white label das credenciais Basic Auth: `companies.provider_token_encrypted` armazena um token Zoug por empresa, com AAD `company:{id}`; as credenciais da API continuam múltiplas e vinculadas à empresa. O envio resolve primeiro o token da empresa. Tokens legados nas credenciais permanecem cifrados e só são usados no fallback de compatibilidade habilitado para empresas preexistentes; substituir/remover o token da empresa desliga esse fallback. O domínio company grava o token; credential gerencia somente Basic Auth e suas configurações. O resolvedor do envio é infraestrutura de integração compartilhada.

Migração 021 adiciona `public_ref UUID NOT NULL` com geração no PostgreSQL e índice único a empresas, usuários e credenciais. Ela preenche registros existentes sem mudar chaves primárias ou FKs; referências textuais de empresas permanecem editáveis. URLs do painel usam `publicRef`; buscas por referência pública mantêm o escopo e a autorização dos endpoints administrativos. Endpoints numéricos continuam temporariamente para clientes internos antigos e ações administrativas. O identificador público é endereço estável, não substitui autorização. A migração é expansiva: aplicar antes do binário que lê `public_ref`; rollback do binário não exige desfazer a coluna. Não há exceção à separação por domínio.

Visibilidade administrativa de agendamentos reside em controller, service, DTO e repository de `scheduled_sms`; o resumo agrega indicadores de fila e MO no domínio `summary`. Router e cmd/api apenas compõem. Migração 014 adiciona índices de leitura por horário previsto e credencial. Não há exceções à separação por domínio. A credencial técnica não identifica a pessoa física que enviou a requisição.

O polish da Auditoria permanece em controller, service, DTO e repository de `audit`. A listagem junta o usuário atual para exibir seu nome, preservando `actor_id` como referência estável; filtros são parametrizados e a ordenação usa lista fechada. Nenhuma tabela, migração ou serviço novo.

## Persistência e contratos

PostgreSQL/GORM, pool limitado, migrações versionadas Gormigrate com transação/advisory lock, sem AutoMigrate de startup. Compose executa migration dedicada antes do processo Go que hospeda API e workers. Migrações novas preservam dados anteriores; arquivamento lógico mantém FKs e histórico.

API administrativa: Huma/Chi, schemas/OpenAPI e Problem Details com código estável. Erros e constantes tipados; causas internas não entram em resposta/log público. Filtros parametrizados, ordenação por whitelist e desempate estável. Paginação de 1 a 100 itens por página. Huma ParamWrapper/ParamReactor distingue filtro booleano ausente de false.

O OpenAPI público apresenta schemas de integração com nomes como `SMSSendRequest`, `SMSRequestMessage`, `SMSSendResult`, `SMSSubmissionStatus`, `SMSDeliveryStatusResponse` e `MORepliesResponse`, além do esquema `basicAuth`. Aliases internos mantêm o código de compatibilidade sem expor o termo `Legacy` ao cliente. A nomenclatura da documentação muda, mas caminhos, campos JSON, autenticação HTTP e códigos de resposta permanecem os mesmos; geradores de SDK podem produzir novos nomes de tipos ao regenerar clientes.

API SMS preserva sua tradução legada. Fluxo: Basic Auth → validação/rate limit → tentativa durável → HTTP white label → resultado durável → resposta. Empresa é registrada na tentativa, mantendo histórico mesmo se a credencial for arquivada. Conteúdo SMS não é persistido. Timeout pode significar aceite: nunca retry automático. Runtime usa limite compartilhado no PostgreSQL: requisição por credencial e orçamento agregado de mensagens.

O envio direto de teste do token da empresa usa `company_test_sends`, separado de `sms_attempts` porque não possui credencial Basic Auth. Cada linha pertence a uma empresa e a um usuário administrador, com referência única, destino, resultado e horário; não guarda texto nem token. A linha nasce como `unknown` antes da chamada ao fornecedor, para que uma interrupção não se transforme em aceite presumido. O painel da empresa lista os 20 testes recentes e o resumo soma esses resultados aos demais SMS por estado. Apenas `accepted` conta como aceite; `unknown` fica separado e exige conciliação antes de novo envio. Os testes anteriores à migração 023 não podem ser reconstruídos com segurança. A migração cria somente uma tabela e índices, sem alterar dados existentes; rollback operacional é voltar a versão da aplicação e manter a tabela para preservar o histórico.

## Segurança e sessões

Argon2id; verificação concorrente limitada, hash fictício para contas inexistentes. SCS mantém sessões no PostgreSQL; cookies HttpOnly/Secure em produção, oito horas máximas e oito horas de inatividade sem “Lembrar de mim”; com a opção, até 14 dias. Renovação no login, destruição no logout, lookup da conta/versionamento a cada request. Casbin controla perfis internos. APIcredentials/clientes não ganham sessão de painel.

CSRF com CrossOriginProtection da biblioteca padrão Go, origens exatas compartilhadas com CORS, somente no prefixo /admin. API externa e ingress de callback têm autenticação independente. HTTPS terminado pela infraestrutura. SameSite=None é configurável para hosting em sites distintos e exige Secure.

Administradores administram a operação global; podem existir vários. Operadores e usuários de consulta possuem vínculos com uma ou mais empresas em `system_user_companies` (migração 018). O guard carrega os vínculos da conta a cada requisição e injeta escopo; os repositórios limitam empresas, credenciais, envios, agendamentos, callbacks e indicadores. Alterar vínculos incrementa a versão da sessão. Convite, recuperação, MFA e sessão lembrada estão separados no domínio de usuários e documentados em [admin-security.md](admin-security.md). Não é portal externo de clientes. Credenciais Basic Auth continuam separadas. Empresas bloqueadas impedem envio por suas credenciais. Último admin protegido por lock transacional. Auditoria contém apenas identificadores/ação/data, sem corpos ou segredos.

## Callbacks e workers

O ingress Zoug permite temporariamente em HML o log integral do corpo e cabeçalhos não secretos por configuração opt-in. A exceção de diagnóstico foi autorizada pelo usuário para observar o contrato real; a rota mantém token e whitelist. Eventos `UNKNOWN` ficam em `delivery_events` sem atualizar o estado legado nem criar callback ao cliente. Não há migração nova nem exceção à separação por domínio.
O histórico administrativo de recebidos usa o domínio callbacks e consulta `delivery_events` e `mo_events` com escopo de empresa. O painel separa esse histórico da outbox de notificações ao cliente. A consulta expõe apenas metadados e estados normalizados; conteúdo MO e payloads brutos ficam fora do DTO. Sem migração.

PostgreSQL armazena delivery_events e callback_jobs como outbox. Evento/estado/job gravados atomicamente; worker reconcilia UUIDs associados depois da chegada do evento. Callback do cliente é GET legado. Lease/SKIP LOCKED, concorrência limitada, oito tentativas, backoff com jitter, HTTP pooled/timeout, sem redirects. Entrega pelo menos uma vez: cliente deve deduplicar.

safeurl bloqueia SSRF no dial, proxy desativado, HTTPS/443; saída IPv6 desabilitada. Bloqueio de conta/empresa impede entrega. Reprocessar callback não reenvia SMS. O payload Zoug é validado pelo adaptador de integração; o controller de callback externo verifica hash do segredo na URL e IP, então encaminha DLR ao serviço de callbacks e MO ao serviço MO. Cada domínio persiste seu evento antes da resposta 204. A rota é opt-in independente das rotas internas normalizadas. Não há exceção à separação por domínio: o controller só coordena o contrato HTTP do fornecedor; regras de DLR/MO continuam nos services e repositories respectivos. Consulte docs/zoug-callback.md e docs/backoffice.md.

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

Visão da empresa: leitura individual em `GET /admin/companies/{id}` no domínio de empresas; a listagem de credenciais reutiliza o filtro `companyId` do domínio de credenciais. A lista de usuários vinculados usa o filtro `companyId` do domínio de usuários e só fica acessível a administradores. Operadores/consulta não leem cadastros de outros usuários.
