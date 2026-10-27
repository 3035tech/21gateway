# Estado da implementação

## Atualização de 27/10/2026 — contas administrativas

Cadastro de usuários internos pela tela agora envia convite por e-mail para definição da senha; suporta vários administradores e perfis admin/operator/viewer. Perfil permite troca da própria senha, MFA TOTP com QR code e códigos de recuperação. Login oferece “Lembrar de mim”; recuperação por e-mail exige MFA se ativo. Migrações 015/016 e configuração SMTP/chave MFA em [admin-security.md](admin-security.md). Sessões `/admin` são independentes da autenticação Basic Auth dos endpoints `/v1` de SMS. Portal de clientes ainda não implementado.

Testes PostgreSQL locais usam schema isolado e mailer falso; nenhum e-mail ou SMS real foi enviado. Entrega SMTP/SES, DNS remetente e operação de chave MFA ainda precisam de homologação antes da ativação em produção. As seções cronológicas anteriores abaixo descrevem o estado da época e não substituem esta atualização.

## Implementado em código

Inventário técnico abaixo; o escopo respondido em docs/scope-answers.md prevalece como requisito. Recursos extras existentes não significam que foram solicitados pela 21.

- Projeto Go MVC/React TypeScript Tailwind, GORM/PostgreSQL, OpenAPI/Swagger, Docker e configuração Amplify.
- Envio imediato unitário pelo contrato legado, Basic Auth separado do painel, Argon2id, persistência de metadados sem texto, tradução para white label e ausência de retry automático de SMS.
- Domínios separados em controllers/services/models/DTOs/repositories; regra obrigatória em Codex/Claude/Cursor.
- Login administrativo com SCS/PostgreSQL, cookies seguros configuráveis, proteção CSRF nativa Go, limite de tentativas compartilhado, Casbin e perfis admin/operator/viewer.
- CRUDs de empresas/usuários/credenciais, arquivamento lógico, auditoria, revogação de sessões e proteção do último administrador.
- Credenciais externas vinculadas à empresa, bloqueio por empresa, username/vínculo imutáveis após associação e senha exibida somente na criação/rotação.
- Telas reais conectadas à API: login, resumo, empresas, usuários, credenciais, envios, callbacks e auditoria; React Hook Form/Zod e tabelas com processamento no servidor.
- Migrações 002/003; empresa original nos metadados de envio, preservação do histórico após arquivamento.
- Outbox/worker de callbacks, reconciliador de eventos antecipados, deduplicação, lease, retries limitados, entrega GET legado e proteção SSRF. Entrada normalizada interna desligada por padrão; integração real de DLR do fornecedor ainda pendente.
- Prévia fictícia em /demo, exclusiva do desenvolvimento, preservada para revisão visual.

## Limitações concretas

Não foi possível iniciar PostgreSQL/API/worker nesta sessão: Compose recebeu permission denied no socket Docker. O frontend existente continua acessível localmente. O fluxo com PostgreSQL ainda precisa ser exercitado; não equiparar testes em memória/transportes simulados à integração real. Teste PostgreSQL opcional preparado com TEST_DATABASE_URL e schema isolado.

Sem envio SMS real: faltam token configurado e destinatário autorizado. Faltam payload/autenticação/mapeamento de callback do white label. Envio continua com callback_level NONE; /internal/dlr é contrato interno, não API presumida do fornecedor. Não houve deployment AWS/Amplify.

Runtime SMS agora usa limite compartilhado no PostgreSQL (uma requisição/s por credencial e teto conservador de 200 mensagens/s agregado). Token por conta, whitelist e lote implementados em código; recuperação de senha/MFA, normalização legada, agendamento, saldo, recebimento real DLR/MO e /v1/status/date continuam pendentes. /v1/consultmo consulta eventos MO locais. Política de retenção/limpeza de metadados não definida; nenhum expurgo implementado. Alertas, backups operacionais e teste de carga de produção ainda pendentes.

## Verificações

Cobertura existente de SMS preservada. Novos testes de sessão/CSRF/permissão, registro de todos os endpoints/OpenAPI, tradução de callbacks, redirects/SSRF e backoff. Testes frontend cobrem formulário de empresa, senha opcional do usuário e distinção entre sessão anônima e falha de infraestrutura, além das regressões anteriores. Resultados finais registrados na resposta da tarefa. Verificação PostgreSQL exige TEST_DATABASE_URL; sem a variável é pulada.

Referências: docs/backoffice.md (fluxos e limites), docs/development.md (execução), docs/aws.md (inventário). IA: AGENTS.md, CLAUDE.md e .cursor/rules/project.mdc compartilham a regra de separação por domínio.

Validação final desta etapa: go test -race ./... e go vet ./... passaram; API/admin/worker compilaram com CGO_ENABLED=0. Frontend: dez testes passaram, lint sem avisos, build e format:check passaram. Compose validou o YAML. O teste PostgreSQL foi explicitamente pulado por ausência de TEST_DATABASE_URL; containers e banco real não foram exercitados.

## Polish do frontend

PanelShell compartilhado entre painel real e demonstração; menu móvel, hierarquia visual, botões de tabela, datas, status de entrega, estados de sessão e login refinados. Diálogos respeitam gravações pendentes; senha de API evita sobreposição com outro diálogo. Mantida a separação por domínio. Revisão visual da prévia no Chrome em desktop/tela pequena, sem overflow da página. Backend/DB não alterados; validação com dados reais ainda depende de execução local do backend.

## Respostas recebidas da 21

Questionário respondido incorporado em docs/scope-answers.md. Compatibilidade completa V1 necessária; código atual é parcial. A análise inicial identificou token por cliente, importação das credenciais existentes, lote, normalização do texto/telefone, whitelist, MO, consultas/créditos e suporte ao agendamento indicado. A seção seguinte registra o que já foi implementado. DLR do fornecedor continua sem contrato. Painel solicitado é gestão de credenciais/configurações com administrador único; telas operacionais/perfis extras são código existente fora desse requisito. MFA desejável ainda não implementado. Nenhuma migração de contas realizada.

## Implementação após as respostas da 21

Adicionadas migrações 004/005, token Zoug por conta criptografado AES-GCM, whitelist IP/CIDR aplicada ao peer da API, URLs DLR/MO distintas, ID legado único, cadastro preservando senha legada conhecida e navegação focada em credenciais. Sem importação automática de contas/hashes e sem fallback global de token.

Enviador aceita 1–200 itens, adaptador de batch documentado com ordem/resultados por item, persistência em lote e limite compartilhado de mensagens/requisições. Nenhum retry automático. Normalização de texto/nono dígito depende de exemplos solicitados; não afirmar compatibilidade completa. Detalhes em docs/account-settings.md e docs/sms-batch.md. Chi/Huma mantidos conforme docs/framework-decision.md.

Docker novamente negou acesso ao socket; PostgreSQL não exercitado. Nenhum SMS real enviado. Testes simulados/ampliação do teste opcional não equivalem à homologação do fornecedor.

Normalização de texto e telefone adiada explicitamente pelo usuário; acompanhar em docs/backlog.md e seguir com as funcionalidades independentes.

Validação desta etapa: go test -race ./... e go vet ./... passaram, incluindo o contrato HTTP de lote com token por conta e bloqueio de IP mesmo com X-Forwarded-For forjado. Frontend: 12 testes, lint, build e formatação passaram. API/admin/worker compilaram; Compose validou a configuração. Integração PostgreSQL continua pulada sem TEST_DATABASE_URL. Consultas V1 permanecem pendentes, sem contratos inventados.

## Consulta legada de status

Implementado GET /v1/status em camadas próprias, com guard legado compartilhado, consumo transacional por credencial, limite 100 e recibos independentes de callbacks (migração 006). DLR posterior volta a ficar disponível. Detalhes/limites em docs/legacy-status.md; /v1/status/date ainda depende do contrato de paginação. Normalização segue no backlog. Consulta usa DLR local; entrada real do fornecedor continua pendente.

Validação da consulta: go test -race ./..., go vet ./... e build da API passaram. Frontend: 12 testes, lint e build passaram; Compose válido. TestPostgresBackofficeAndOutbox explicitamente pulado pela ausência de TEST_DATABASE_URL; transação e migração 006 ainda não exercitadas em banco real.

## Tentativa de validação local com banco

Após autorização para seguir com a validação, Docker ps e docker compose up -d postgres voltaram a receber permission denied no socket. pg_isready em 127.0.0.1:5432 não encontrou servidor. Há ferramentas libpq/initdb/pg_ctl, mas nenhum binário postgres nas instalações locais consultadas; não foi iniciado banco alternativo. Nenhum teste de integração real foi executado nesta tentativa. Executar docker compose --profile verification run --rm backend-test em terminal com acesso ao Docker para destravar esta prova.

URLs de callback já fazem parte da credencial externa: callbackUrl/callbacksEnabled para DLR e moCallbackUrl/moCallbacksEnabled para MO, editáveis no painel. Configurar MO não significa fluxo MO implementado; integração real do fornecedor segue pendente.

## Preparação corrigida da verificação PostgreSQL

Target verification separado no Dockerfile com build-base/CGO para go test -race; runtime de produção preservado. Compose exige REQUIRE_DATABASE_TESTS=true e usa saída verbose. Integração ampliada para persistência DLR/MO e consumo paralelo sem duplicação. Banco e imagem de verificação ainda não executados: socket Docker continua negado. Testes locais sem banco e vet passaram; configuração Compose válida. O modo obrigatório sem DSN deve falhar explicitamente. Reexecutar com docker compose --profile verification run --build --rm backend-test.

## Preparação de callback MO

Implementado adaptador POST JSON de saída com DTO/model transitório próprios, reutilizando cliente HTTP/SSRF/timeout/dispatch DLR. Testes de tradução, Unicode, URL, IDs, loopback e falhas sem retry. Não há ingresso, persistência/outbox ou worker MO; URL cadastrada não implica entrega ativa. Contrato do fornecedor e IDs/data legados ainda precisam de homologação. Detalhes em docs/mo-callback.md.

Validação do adaptador MO: go test -race ./..., go vet ./... e build API passaram; Compose válido. Frontend não alterado. Integração PostgreSQL pulada sem TEST_DATABASE_URL; sem fornecedor ou callback real.

## Fluxo MO interno durável

Migração 007, /internal/mo opt-in/Bearer, conteúdo AES-GCM com chave independente e HMAC para deduplicação, associação credencial+UUID e reconciliação antecipada. Worker integrado ao cmd/worker com autorização atual de conta/empresa/URL, lease/token e backoff compartilhado. Saída POST legado. Detalhes em docs/mo-flow.md. MO real do fornecedor e /v1/consultmo pendentes; retenção/rotação não definidas, sem expurgo. Nenhum SMS/callback real enviado.

Validação do fluxo MO: go test -race ./..., go vet ./... e builds API/worker passaram. Frontend: 12 testes, lint e build passaram. Compose válido. Teste PostgreSQL explicitamente pulado por ausência de TEST_DATABASE_URL; migração/transações reais ainda não exercitadas.

## Polish de credenciais após MO

Seções responsivas, callbacks DLR/MO agrupados, campos condicionais preservando valores, ações persistentes na rolagem e efeitos de bloqueio/remoção de token claros. Listagem distingue DLR/MO. Reutilizados Dialog/FormFooter com opções pequenas, sem reunir domínios. Dois testes de formulário confirmam preservação de URLs/IPs/token e remoção explícita de token após substituição digitada. Revisão Chrome com fixtures, sem gravação/backend real; página temporária removida. Frontend: 14 testes passaram; validação final de lint/build/formatação registrada nesta etapa. Backend/contratos externos não alterados.

Validação final do polish: frontend 14 testes, lint, build e format:check passaram. Backend go test -race ./..., go vet ./... e build API passaram; Compose válido. Teste PostgreSQL permanece pulado sem TEST_DATABASE_URL. Revisão visual com fixtures não equivale a integração de banco/fornecedor.

## Consulta legada MO

GET /v1/consultmo registrado em OpenAPI com Basic Auth/whitelist, filtro de dia São Paulo, retorno legado e descriptografia por conta. Consulta não consome eventos e independe de entrega/ativação dos callbacks. Migração 008 adiciona índice de consulta. Testes de contrato, limites, DST, autenticação e privacidade; PostgreSQL opcional ampliado. Detalhes/limites provisórios em docs/legacy-mo-query.md. Fornecedor, status/date, saldo e agendamento ainda pendentes.

Validação da consulta MO: go test -race ./..., go vet ./... e build API passaram. Frontend 14 testes, lint e build passaram; Compose válido. PostgreSQL explicitamente pulado sem TEST_DATABASE_URL. Nenhum callback/SMS externo acionado.

## Pesquisa de contratos externos

Revisados manual, guia white label e exemplos PHP/Java públicos da 21; página oficial Zoug consultada. Paginação status/date não confirmada; saldo/agendamento sem contrato no material disponível. Evidências e solicitações prontas em docs/contract-discovery.md e docs/contract-requests.md. Nenhum código/API/migração alterado e nenhum contato externo enviado.

## Encerramento após polish

Tela operacional DLR nomeada Notificações de entrega, estados traduzidos com StatusBadge compartilhado e falhas legíveis preservando o código no título. Erro de reprocessamento é limpo ao escolher outra notificação. Sem alteração de API/worker/banco. Ponto de retomada em docs/next-session.md; demais evoluções deixadas para amanhã pelo usuário.

Verificação do encerramento: frontend 14 testes, lint, build e format:check passaram. Backend não alterado nesta etapa; últimas verificações continuam válidas. Banco real/contratos externos permanecem pendentes.

Esclarecimento posterior sobre texto/telefone registrado em docs/backlog.md: exemplo normalizado Ola, acao!; regra do nono dígito ainda parcial. Sem alteração de código.

## Normalização SMS autorizada

Helper compartilhado single/batch com gomoji e x/text, sem acentos/emojis, encoding após transformação e limite mantido na origem. Texto vazio unitário rejeitado; lote mantém validação de itens vazios pelo fornecedor. Sem alterar MO/destino/agendamento/persistência do texto. Regra telefônica segue pendente. Detalhes em docs/sms-normalization.md.

Verificação da normalização: go test -race ./..., go vet ./... e build API passaram; Compose válido. Frontend não alterado. PostgreSQL segue sem execução real por ausência de TEST_DATABASE_URL.

## Operação e retenção de 90 dias

Usuário autorizou itens 3/4/5 e confirmou 90 dias de retenção do conteúdo MO. Migração 009, worker de expurgo preservando jobs ativos, keyrings AES-GCM v1/v2, CLI manutenção dry-run/recriptografia com CAS e diagnóstico operacional protegido. Backup/restauração via pg_dump/pg_restore+age, testes de segurança dos scripts e benchmarks em memória/roteiro k6. Detalhes em docs/operations.md. Nenhum recurso AWS ou dado real alterado. Contratos de paginação/saldo/agendamento e corte 4/5 continuam aguardando evidências.

Validação operacional: go test -race ./..., go vet ./... e builds API/worker/maintenance passaram. Frontend: 14 testes, lint e build passaram; Compose válido. Três testes de segurança dos scripts passaram com CLIs simulados. Benchmarks executados e registrados. PostgreSQL explicitamente pulado sem TEST_DATABASE_URL; age/k6 não instalados, backup/restauração e carga HTTP reais não executados. Rotação/expurgo não foram aplicados a dados reais.

## Questionário final — 15/10/2026

Versão final em docs/references/perguntas-migracao-respondidas-finalizado.docx; somente respostas 54–61 mudaram. Operação indicada 3035/Mozhtech; conclusão desejada em outubro; teste interno, piloto de produção (infinitech ou grupoamaro, ainda não escolhido) e demais em conjunto. Aceite exige SMS e DLR/MO ponta a ponta até o cliente. Pendências antigas e rollback continuam a discutir, sem autorização de descarte. Sem requisitos adicionais de segurança informados; retenção MO de 90 dias permanece. Plano e evidências em docs/migration-plan.md; contratos externos e prova PostgreSQL continuam pendentes. Atualização apenas documental.

## Validação dos fluxos — 15/10/2026

Novo teste sem rede integra serviços/adaptadores de envio, DLR e MO. Nova suíte PostgreSQL prepara aceite com repositories/workers reais, falhas/backoff, deduplicação, isolamento e troca de URL; fixture compartilhada usa schema exclusivo. DLR agora bloqueia job cuja URL difere do cadastro atual, como MO já fazia, sem redirecionamento automático. Go race/vet/build, frontend 14 testes/lint/build, Compose e três testes simulados de backup passaram. PostgreSQL explicitamente pulado: sem DSN e Docker sem permissão. Modo REQUIRE_DATABASE_TESTS testado: falha esperada sem DSN. SQL e backup/restauração reais ainda não comprovados. Evidências/execução em docs/acceptance-validation.md.

Polish de 15/10/2026: ponto de retomada consolidado, instruções de Codex/Claude/Cursor corrigidas para texto/retenção já implementados; testes de aceite protegidos contra UUID nulo e fixture PostgreSQL valida DSN antes de criar objetos, tratando erros de pool. Sem alteração no contrato público. Prova PostgreSQL continua pendente.

Confirmação de 15/10/2026: clientes utilizam correlationId. Campo já preservado no envio unitário/lote, persistido e traduzido para reference_id do fornecedor; retorna nas respostas de envio, /v1/status e callbacks DLR/MO conforme grafia do contrato. /v1/status/date continua pendente. Não inferir unicidade nem idempotência a partir dessa confirmação.

## Evidência de callbacks — 15/10/2026

Resposta 22 do formulário implementada com logs seguros: HTTP status/código/duração/evento nas falhas e conclusão do worker com tipo DLR/MO, job/evento/tentativas e outcome_persisted. Recuperação só após Finish bem-sucedido; sucesso inicial sem log por mensagem. Worker usa slog JSON em stdout. Sem URLs, texto, telefones, correlationId, tokens ou erros originais. Detalhes em docs/operations.md. Sem migration, dependência, alteração de payload ou política de retry. Coleta/retenção/alertas e PostgreSQL real continuam pendentes.

Polish da observabilidade em 15/10/2026: logs migrados para atributos tipados e LogAttrs do slog, preservando campos/níveis/mensagens. Testes reforçam mensagens exatas e falha simultânea de entrega/persistência. Nenhuma política de retry, contrato HTTP, dependência ou migration alterada.

## Segurança e ataques controlados — 15/10/2026

Relatório em docs/reviews/security-assessment-2026-10-15.md. Corrigida falta de throttling pré-Basic Auth: runtime injeta AuthenticationAttempts PostgreSQL, API com orçamento por peer IP configurável (API_AUTH_IP_LIMIT_PER_MINUTE padrão 600, inclui tráfego válido), falhas por conta bloqueiam após dez até o próximo minuto; backoffice mantém dez logins/IP e ganha limite por e-mail. Sem confiança em XFF, janelas fixas e possível excesso por tentativas já em curso. Falha do limitador fecha acesso. Novas senhas internas/bootstrap mínimo 15 caracteres/máximo128 bytes, pequena denylist; login e senhas legadas preservados. Headers DENY/no-referrer adicionados. Testes de ataques HTTP e SQL DryRun; suíte PostgreSQL nova preparada, não executada. npm audit zero vulnerabilidades conhecidas; govulncheck não instalado. MFA, DAST/browser, infraestrutura/proxy/IP real e prova de banco ainda pendentes. Sem SMS ou tráfego externo de ataque.

Polish de segurança em 15/10/2026: frontend agora conta mínimo da senha interna sem espaços nas pontas como backend, com limites centralizados e mensagens claras; senha original preservada na validação/envio. Regressão reproduzida antes da correção. Testes de configuração fixam orçamento de autenticação no ambiente de teste para evitar influência do terminal. Sem alteração de senha legada ou limite de autenticação em produção.

## Agendamento local e volume — 15/10/2026

Usuário autorizou scheduler interno e reforçou alto volume/desempenho em todos os endpoints. Agendamento local implementado em docs/sms-scheduling.md, revogando a dependência de agendamento do fornecedor: migração010, chave AES independente/keyring, aceite durável/id_sms preservado, pedidos mistos, worker por conta até200, SKIP LOCKED/lease/CAS, limites compartilhados, nenhum retry após dispatch/resultado incerto e limpeza do conteúdo terminal. Reutiliza o sender existente via adaptador de tentativas. Opt-in desligado até configuração/prova PostgreSQL. Diagnóstico inclui scheduler; resumo agrega contagens em quatro consultas. Sem prova de TPS/SQL real; limites e pendências (TTL/quota/cancelamento/aviso de falha local) explícitos.

AWS para scheduler discutida em 15/10/2026: nenhuma integração SQS/EventBridge ou recurso novo criado. Opções verificadas em docs/aws.md: worker contínuo ECS/RDS como ponto de partida; EventBridge por lote tem precisão60s e SQS delay15min, com deduplicação ainda necessária. Usuário pediu avaliar, sem decisão de migrar a fila.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migração011 obrigatória nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. Prova PostgreSQL/SQS/carga ainda pendente; testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.

Polish SQS: publicador prioriza outbox pronta antes de preparar novos lotes, isolando falhas de preparação; regressão reproduzida e corrigida. Sem mudança de contrato, schema ou infraestrutura. Ver docs/sqs-dispatch.md.

## Correções da auditoria — 27/10/2026

Suíte PostgreSQL real com `go test -race ./...` passou após corrigir `allowed_ips` ausente, a consulta do histórico de envios e a reserva de requisição sem mensagens. Migração 012 ajusta a reserva; 013 adiciona índices de ordenação para listagens de SMS e callbacks. Aplicar ambas antes dos binários novos. `go vet`, build Go, frontend lint/build/17 testes, `npm audit`, build/sintaxe da imagem Nginx e Compose config também passaram. A imagem de build usa Go 1.27.3. Detalhes e riscos remanescentes em `docs/reviews/gateway-audit-2026-10-27.md`. PostgreSQL local não substitui homologação de fornecedor, SQS real, carga ou validação de proxy/IP no Dublin.

## Visibilidade do administrador — 27/10/2026

Dashboard real com fila de agendamentos (vencidos, próximas 24h, em processamento e total), resultados históricos de envio, pendências/falhas DLR e MO, empresas e credenciais ativas. Tela de agendamentos com cliente, credencial técnica, datas e estados, busca/filtros/ordenação/paginação. Sem conteúdo SMS e sem identificação da pessoa que usou a credencial. Endpoint `/admin/schedules` e consultas no domínio scheduled_sms; resumo ampliado no próprio domínio. Migração 014 de índices necessária antes do backend novo. Scheduler/SQS mantêm ativação opt-in. Interface revista no Chrome com API simulada; `go test -race ./...` com PostgreSQL, `go vet`, build Go, frontend lint/build/formatação/17 testes e Compose config passaram. Homologação externa, carga e deploy seguem pendentes.

Auditoria de padronização posterior: `docs/reviews/ui-standardization-2026-10-27.md`. Ordenação ampliada às colunas informativas, sem ordenar Ações; nomes de cliente nos históricos de envios/callbacks; busca e leitura das tabelas largas refinadas. Consultas usam listas fechadas de ordenação; medir custo dos novos sort e buscas com volume representativo antes de prometer desempenho.

Polish seguinte de Auditoria: nome atual do responsável e ID no painel; filtros por responsável, ação e recurso, busca também pelo nome e ordem inicial pelos eventos recentes. Domínio `audit` preservado, sem migração. Teste PostgreSQL do nome/filtros e teste de interação frontend adicionados; consulta continua restrita ao administrador.

## Processo único e CI — 27/10/2026

O backend agora inicia API, callbacks, retenção MO e, quando habilitados, MO e scheduler/SQS no mesmo processo `/api`. Compose e GitOps Dublin usam um contêiner de backend; o Job de migração continua separado. O comando/binário worker foi removido. Um repositório GitHub com `backend/` e `frontend/` foi escolhido; workflows em `.github/workflows` compilam e verificam cada aplicação e preparam imagens GHCR `21gateway-back` e `21gateway-front`. O `.gitignore` da raiz cobre artefatos temporários das duas pastas e preserva `.env.example`. Apenas a estrutura local foi preparada: não há remoto, publicação ou sync. O build do frontend incorpora o domínio Dublin atualmente presumido. Validação local: `go test -race ./...` com PostgreSQL (após uma falha não reproduzida em teste de lote), `go vet ./...`, build Docker backend, frontend lint/18 testes/build, Compose config, Helm lint/template e smoke local de `/healthz` com log do callback worker no mesmo contêiner. SQS, fornecedor e cluster Dublin não foram exercitados.

## Credenciais múltiplas e idiomas — 27/10/2026

Empresa aceita várias credenciais distintas. Migração 017 adiciona `expires_at` nulo para as existentes. Cadastro suporta username gerado ou manual, senha gerada/manual/legada e validade opcional; expiração bloqueia autenticação, SMS de teste e envio agendado. A listagem exibe vencimento. Catálogos `pt-BR`/`en-US` centralizam textos do painel e mensagens públicas de erro; `/admin` traduz `detail` por `Accept-Language`, preservando o contrato `/v1`. Workflows do GitHub passam a identificar `21gateway-back` e `21gateway-front` no nome e metadados da imagem. Aplicar a migração 017 antes da nova imagem de backend. Não foram enviados SMS reais.
