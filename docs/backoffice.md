# Backoffice integrado

Consulta direta de credenciais e usuários: `GET /admin/credentials/{id}` e `GET /admin/users/{id}` retornam DTOs sem senha/token em claro. Sessão e permissão de leitura são obrigatórias; credenciais respeitam o escopo de empresa. A interface usa URLs próprias para abrir o detalhe a partir da empresa.

No cadastro da credencial, “Acesso do cliente · Basic Auth” define o username e a senha que autenticam `/v1`. O username pode ser gerado ou digitado; a senha pode ser gerada, digitada manualmente ou preservada na migração. O backend guarda somente hash e usa esses campos em `Authorization: Basic`, com validade, estado da credencial/empresa e whitelist. Ao editar, a senha existente não é exibida; a troca é uma ação separada. O token Zoug é outra credencial, para autenticação do gateway perante o fornecedor.

Atualização de 27/10/2026: o token Zoug é cadastrado na ficha da empresa e compartilhado por suas várias credenciais Basic Auth. O gateway gera usuário/senha da API 21, mas apenas recebe o token emitido pela Zoug. A migração 020 preserva tokens antigos por credencial em fallback controlado até configurar o token único da empresa.

> Escopo respondido em 14/10/2026: consulte docs/scope-answers.md. Este documento descreve o código existente; token por cliente, whitelist, MO e compatibilidade V1 completa ainda exigem ajustes. Painel solicitado pela 21 é focado em credenciais/configurações com perfil administrativo único. Não confundir recursos extras implementados com requisitos confirmados.
## Domínios e acessos

Empresas, usuários internos, credenciais externas, envios, callbacks, auditoria e resumo possuem arquivos próprios em cada camada necessária. Não há controller/service/repository monolítico de backoffice. Sessão e autorização são infraestrutura compartilhada.

| Perfil interno | Permissões |
| --- | --- |
| Consulta (`viewer`) | Ver dashboard, empresas, credenciais, envios, agendamentos e callbacks; sem alterações |
| Operador (`operator`) | Permissões de consulta; criar/editar/arquivar empresas e credenciais, trocar senhas da API e reprocessar callbacks falhos |
| Administrador (`admin`) | Permissões de operador; gerir usuários, consultar auditoria e fazer envio de homologação para o número configurado |

Usuários internos não pertencem automaticamente a uma empresa cliente. Todos os perfis são da equipe interna e consultam a operação inteira. Clientes externos recebem somente credenciais Basic Auth vinculadas à sua empresa. Um portal de clientes com acesso restrito por empresa não está implementado.

O cadastro oferece esses três perfis fixos e sugere `viewer` para novas contas. A descrição da permissão aparece ao selecionar cada opção. O dashboard está disponível aos três perfis; usuários e auditoria continuam exclusivos do administrador. A autorização efetiva é feita no backend.

## Autenticação

Perfil, convite por e-mail, recuperação, TOTP com QR code e sessões lembradas estão detalhados em [admin-security.md](admin-security.md). A tela de usuários permite vários administradores; os perfis atuais são internos e não concedem acesso de clientes ao painel.

Login por e-mail/senha Argon2id, SCS com sessões no PostgreSQL. Cookie `21mobile_session`, HttpOnly, escopo `/admin`; por padrão, duração máxima de oito horas e inatividade de trinta minutos. “Lembrar de mim” estende o prazo absoluto a 14 dias e a inatividade a sete dias. Renovação no login e destruição no logout. Senha, perfil, status ou MFA alterados invalidam as sessões por versão; cada request protegido verifica a conta atual. O último administrador ativo não pode ser bloqueado, arquivado ou rebaixado; alterações de usuários são serializadas por lock transacional PostgreSQL.

Proteção CSRF com `net/http.CrossOriginProtection` do Go, aplicada ao prefixo administrativo; origens confiáveis exatas com esquema/host/porta. A mesma configuração alimenta CORS, sem wildcard. Referências: [Go](https://pkg.go.dev/net/http#CrossOriginProtection), [SCS](https://github.com/alexedwards/scs). Não há token de sessão em localStorage. Chamadas HTTP do navegador usam cookies e Origin/Fetch Metadata; não é necessário inventar um token CSRF no frontend.

Login tem limite compartilhado de dez tentativas por minuto por IP de conexão, persistido no PostgreSQL, além da concorrência limitada de Argon2id. Não confiar no X-Forwarded-For informado pelo cliente; atrás de proxy, usuários podem compartilhar esse limite. Uma política de proxies confiáveis ainda deve ser definida no deployment.

Em HTTP local, configurar `COOKIE_SECURE=false`. Produção exige `COOKIE_SECURE=true` e origens HTTPS. `COOKIE_CROSS_SITE=true` configura SameSite=None e só é aceito com cookie seguro; necessário quando frontend Amplify e API pertencem a sites diferentes. Preferir domínios da mesma organização quando disponíveis. Limitações de cookies de terceiros devem ser verificadas na homologação do hosting.

## Cadastros e histórico

Empresas: nome, referência opcional e ativo. Usuários internos: nome, e-mail único, perfil e ativo; a tela envia convite para definição da senha pelo próprio usuário. A API de cadastro anterior conserva o campo de senha por compatibilidade. Credenciais: empresa, username imutável, ativo e configuração de callback. Após vincular uma credencial a uma empresa, o vínculo não pode ser trocado; criar outra credencial para outro cliente. Credenciais antigas sem empresa permanecem válidas até serem vinculadas, preservando a transição.

Vínculos de usuários: `companyIds` é vazio para `admin` e obrigatório com IDs únicos de empresas ativas para `operator`/`viewer`. A migração 018 cria `system_user_companies`; aplicá-la antes do backend novo. Contas não administradoras antigas sem vínculos deixam de visualizar dados de empresas até receberem atribuições. Um administrador pode vincular várias empresas por usuário; alteração revoga sessões existentes. O escopo é aplicado pela API, inclusive quando filtros ou IDs são manipulados manualmente. Operador não cria empresa nova; pode operar empresas atribuídas conforme permissões anteriores. A lista de usuários filtrada por empresa continua restrita ao administrador.

Uma empresa pode ter várias credenciais, com usernames globalmente únicos. Na criação administrativa, `generateUsername` gera um username aleatório `gw_...`; caso contrário, `username` é informado. A senha pode ser gerada, informada em `manualPassword` (mínimo de 15 caracteres) ou preservada em `legacyPassword` para migração. A resposta só apresenta a senha gerada uma vez; senha manual/legada não é repetida. `expiresAt` é opcional e expira no instante UTC informado; a ausência mantém validade indeterminada. `clearExpiry` remove uma validade existente na edição. Username e vínculo com empresa permanecem imutáveis; rotação de senha existente continua em `/credentials/{id}/rotate`. Aplicar migração 017 antes do backend novo. Autenticação, SMS de teste e scheduler recusam credencial vencida; callbacks de envios anteriores mantêm seu fluxo.

Mensagens públicas de erro ficam em `backend/internal/errors/locales/pt-BR.json` e `en-US.json`, por código estável. O prefixo `/admin` usa `Accept-Language` para o campo `detail`; `/v1` mantém seu texto legado. O painel guarda preferência de idioma no navegador e usa `frontend/src/locales/pt-BR.json` e `en-US.json` para labels, validações e erros.

Senhas externas são aleatórias, armazenadas somente como hash e exibidas uma vez ao criar/trocar. O formulário de edição não consulta ou recupera senhas antigas. Bloqueio de empresa impede autenticação de suas credenciais, sem apagar histórico. Arquivamento lógico mantém FKs e auditoria. Tentativas novas registram a empresa de origem; credenciais e empresas arquivadas não escondem seus envios históricos.

Listagens fazem busca literal, filtro de status/empresa, paginação de 1 a 100 itens, ordenação por whitelist e desempate por ID. Frontend usa TanStack Table/Query; formulários usam React Hook Form/Zod e respostas têm validação Zod. Falhas não apresentam linhas antigas como resultado válido. Diálogos nativos mantêm foco e aceitam Escape.

## Rotas

Prefixo `/admin`: `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`; listagem/criação em `/companies`, `/users`, `/credentials`, edição por PUT e arquivamento por DELETE em `/{id}`; `POST /credentials/{id}/rotate`; `GET /messages`, `/callbacks`, `/audit`, `/summary`; `POST /callbacks/{id}/retry`. Schemas e permissões estão refletidos na especificação OpenAPI. Consulta individual para edição utiliza o registro da listagem nesta etapa.

Homologação opt-in de credencial: `GET /v1/auth/check` no guard legado e `POST /admin/credentials/{id}/test-send` somente para administrador e número de homologação configurado no backend. O último reutiliza o envio unitário e seus limites; não confirma entrega. Consulte [credential-testing.md](credential-testing.md).

Envios mostram submissão e entrega separadas. Sem DLR, a entrega aparece como sem confirmação. Não há botão de reenvio SMS. Auditoria armazena ator, ação, recurso, ID e data; não armazena corpos de requisição, senha ou conteúdo de SMS. Resumo considera todos os registros mantidos, não somente o dia atual.

`GET /admin/audit` aceita paginação, busca por ação/recurso/nome atual do responsável, ordenação por campos permitidos e filtros `actorId`, `action`, `resource`. O nome do responsável é lido da conta atual para ajudar a equipe; `actorId` permanece a identidade estável e o nome não representa um snapshot histórico. A tela abre pelos eventos recentes. Perfis sem permissão de auditoria continuam bloqueados.

## Callbacks

Migration 003 adiciona eventos de entrega e outbox durável. `cmd/worker` reconcilia callbacks antecipados e entrega notificações GET com `idSms`, `correlationId`, `status`, conforme manual legado. O cabeçalho `X-21Mobile-Event-ID` oferece um identificador estável para deduplicação no cliente.

O recebimento real do white label continua pendente: payload, autenticação e mapeamento de status não foram fornecidos. `POST /internal/dlr` é um contrato interno normalizado, disponível somente com `CALLBACK_INGRESS_ENABLED=true` e Bearer `CALLBACK_INGRESS_TOKEN` de pelo menos 32 bytes. NÃO é uma implementação presumida do contrato do fornecedor. O envio SMS permanece com callback_level NONE até homologar o adaptador.

Contrato interno: `eventKey` (identificador estável, até 200 caracteres), `providerMessageId` (UUID), `status` (enum legado) e `occurredAt` (RFC3339). Aceite retorna 204 após persistência. Repetir mesmo evento é idempotente; reutilizar a chave com outros dados retorna 409. Horário é normalizado para microssegundos. Eventos antecipados ficam persistidos até o UUID do envio ser associado.

Estado de entrega não retrocede por evento mais antigo. SENT é intermediário; os outros estados previstos são finais. Eventos após um estado final são registrados, mas não substituem o resultado. Correções de resultados finais precisam de contrato explícito; não supor essa semântica para o fornecedor.

Worker: concorrência quatro, seleção com FOR UPDATE SKIP LOCKED, lease de trinta segundos e token para evitar confirmação por worker antigo. Mantém ordem por envio, timeout HTTP de dez segundos, até oito tentativas, backoff exponencial com jitter. Apenas HTTP 2xx confirma entrega; redirects são bloqueados. Reprocessamento manual é permitido somente para job falho cujo status ainda corresponde ao resultado atual do envio; nunca reenvia SMS. Credencial/empresa bloqueada impede entrega. Uma entrega pode ocorrer novamente após crash ou timeout: semântica pelo menos uma vez, não exatamente uma vez.

safeurl verifica IP no momento de conectar, bloqueando redes privadas, loopback, link-local e outras redes especiais. URLs exigem HTTPS/443, sem credenciais ou fragmento; proxy e redirects desabilitados. IPv6 de saída está desabilitado nesta versão. Cliente HTTP e conexões são reutilizados. Configurar URL não prova que o endpoint existe ou está homologado.

## Execução e validação

Compose possui PostgreSQL, migration explícita, API, worker e frontend. A API e o worker dependem da conclusão da migração. O primeiro administrador é criado por bootstrap explícito e senha do ambiente, sem conta padrão. A prévia fictícia continua em `/demo`, apenas no desenvolvimento.

Teste PostgreSQL opcional `TestPostgresBackofficeAndOutbox` usa TEST_DATABASE_URL, cria schema exclusivo e remove apenas esse schema no fim. Cobre migrações repetidas, CRUDs, proteção do último admin, rotação, callbacks antecipados/deduplicação/lease, arquivamento sem perda de histórico e revogação de sessão. Sem essa variável, o teste é pulado e não comprova persistência real.

Migrações 002/003 são expansivas: não removem dados legados. Não reverter schema para fazer rollback do binário. Rollback para versão que não verifica empresa/sessões pode perder controles de acesso; exige revisão operacional e bloqueio equivalente das credenciais antes de trocar a versão.

## Atualização de contas e envios

Token global foi substituído no runtime por token criptografado por credencial. Whitelist e destino MO adicionados ao painel. Lote 1–200 e limitador compartilhado implementados; normalização legada ainda não homologada. Consulte docs/account-settings.md e docs/sms-batch.md; estes detalhes prevalecem sobre descrições anteriores da etapa unitária.

Adaptador de saída POST MO preparado conforme docs/mo-callback.md; nenhuma entrega MO ativa apenas por cadastrar URL. Ingresso/outbox/worker MO permanecem pendentes.

MO evoluído com domínio próprio, eventos/outbox em tabelas separadas, chave MO_CONTENT_KEY independente, ingresso interno opt-in e worker integrado. Aplicar migração 007; configuração, segurança e limites em docs/mo-flow.md. Cadastro da URL não ativa recebimento do fornecedor.
