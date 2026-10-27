# Contexto e decisões do projeto

## Objetivo

A 21 Mobile oferece envio de SMS e migra de uma plataforma própria para white label. Clientes existentes usam sua API. O gateway deve preservar a interface necessária do legado e traduzir chamadas e respostas para o fornecedor. O fornecedor enviará callbacks ao gateway, que deverá traduzir e encaminhar aos clientes.

Também haverá um painel administrativo para gestão de clientes e credenciais. O usuário indicou cadastros de usuários e empresas como os primeiros CRUDs previstos. O usuário confirmou a separação: usuários de sistema acessam o backoffice por login normal; credenciais externas da API são acessos técnicos independentes, vinculados a empresas.

## Etapa atual autorizada

O usuário autorizou envio imediato unitário e as seis evoluções: homologação, login/permissões, cadastros, gestão de credenciais, callbacks e operação/migração. Pediu integrar as telas reais ao backend e separar cada domínio em suas próprias camadas. Usuários de sistema fazem login no backoffice; clientes usam credenciais externas da API, vinculadas a empresas.

Implementação e limitações estão em docs/status.md e docs/backoffice.md. Contratos ausentes do fornecedor não são presumidos; envio real requer token e destinatário autorizado. O runtime agora resolve token criptografado da conta autenticada, sem fallback global. Configurações e limites em docs/account-settings.md e docs/sms-batch.md.

## Decisões confirmadas

- Projeto independente em `21mobile-gateway`, com `backend` e `frontend`, separado dos diretórios legados.
- Repositório Git local inicializado. Não há repositório remoto ou deployment nesta etapa.
- Go; arquitetura MVC com controllers, services, models e DTOs.
- PostgreSQL e ORM GORM. Reutilização de bibliotecas e padrões úteis, com foco em segurança e alta vazão.
- Huma/Chi para HTTP, validação e documentação OpenAPI; Swagger UI.
- React/TypeScript/Vite/Tailwind. Material UI foi substituído por Tailwind a pedido do usuário.
- TanStack Query e Table, Axios, Zod e React Hook Form/resolvers.
- Listagens com filtros, paginação, número de itens por página e ordenação clicando no cabeçalho; processamento no servidor.
- Docker para as aplicações e Compose com PostgreSQL. Amplify Hosting para o frontend estático; backend e banco em infraestrutura separada, ainda não definida.
- Códigos de erro e estados tipados, constantes centralizadas e mensagens públicas padronizadas.
- Documentação compartilhada para Codex, Claude e Cursor.

## Fontes analisadas

- Manual legado: `docs/references/21-mobile-api-2026.pdf` (cópia do arquivo fornecido em Downloads), versão declarada 1.0.2.
- White label: `docs/references/white-label-api.docx` (cópia do arquivo fornecido em Downloads), guia que identifica Devyx e base `https://endpoint.devyx.com.br`. Confirmar fornecedor e endpoint definitivos antes da integração.
- Diretórios `cpanel-web` e `origem-java/cpanel-web`: cópia disponível contém estrutura e estilos, mas não arquivos PHP; controllers/models/config estão vazios. PHP/CodeIgniter é hipótese pela estrutura, não confirmação do código. Endpoint legado foi localizado no manual, não nos fontes.

## Contrato legado documentado

Basic Auth com usuário/senha, whitelist de IPs, domínio `api.21mobile.com.br`.

- `POST /v1/send`: lista `sms` de 1 a 200 itens; limite documentado de uma requisição por segundo, com exceções comerciais possíveis.
- Campos: `messageText` (160 caracteres), `destination` (DDD+número, 11 dígitos), `correlationId` opcional (50 caracteres), `scheduledDate` opcional (Unix timestamp).
- Resposta por item: `OK`/`NOK`, `code`, `id_sms`, `correlationId`; erros parciais aparecem em exemplo HTTP 200. Códigos documentados incluem 5, -5 e 429; regras reais precisam de validação.
- `GET /v1/status`: status ainda não consultados, blocos de 100; semântica de consumo precisa ser confirmada.
- `GET /v1/status/date`: consulta por data, filtros de referência/destino e paginação.
- `GET /v1/consultmo`: respostas por data.
- `GET /v1/credits`: saldo de créditos.
- DLR: callback GET com `idSms`, `correlationId`, `status`.
- MO: callback POST com campos `Message`, `Mobile21Identifier`, `CorrelationId`, `MtId`, `PhoneNumber`, `Date`.
- Status documentados: SENT, DELIVERED, NOT_DELIVERED, DUPLICATED, INVALID_DESTINATION_ADDRESS, BLACKLIST, MESSAGE_CANCELLED, INVALID_MESSAGE_FORMAT, GENERAL_ERROR.

Há exemplos de JSON inválido e inconsistências no manual. Não corrigir silenciosamente um contrato real com base em exemplos imperfeitos; conferir tráfego/implementação.

## White label documentado

Token em `X-API-TOKEN`; `POST /api/send` individual e `/api/send/batch` com até 200 mensagens. Campos principais: `type=SMS`, `message`, `encoding`, `destination_addr`; `reference_id` e `index` opcionais. IDs são UUIDs. Lote tem sucesso parcial HTTP 207, posições e resultados individuais.

Envio individual documenta `callback_url` e `callback_level`; lote não documenta esses campos. Não há payload de callback, consulta de status/saldo/MO, agendamento ou garantia de idempotência no arquivo recebido. Isso significa ausência de documentação, não prova de ausência de suporte.

Há divergência de comprimento de telefone entre single/batch e dúvidas sobre encoding/segmentação. Não implementar conversão de texto ou custo por segmento sem confirmação.

## Identidade visual

Referência autorizada: https://21sms.com.br/. CSS público consultado em https://www.21mobile.com.br/public/css/layout.css usa Raleway, roxo `#660066` e magenta `#ff0066`. Logo local obtido de https://www.21mobile.com.br/public/img/21-mobile.png e guardado em `frontend/public/brand/21-mobile.png` (branco/transparente, adequado a fundo roxo).

Fonte Raleway hospedada localmente via Fontsource. O painel usa esses elementos com layout administrativo simples. Magenta claro serve como destaque; textos/ações usam variantes com contraste adequado. Não importar scripts de marketing, chat ou analytics do site.

## Evolução autorizada

O usuário autorizou as seis frentes: homologação do envio unitário, autenticação/permissões, CRUDs de empresas/usuários, gestão de credenciais externas, callbacks confiáveis e operação/migração. A implementação está em andamento; essa autorização amplia a etapa inicial descrita acima, sem confirmar contratos não documentados do fornecedor. Envio real ainda exige número autorizado e configuração do token.

Em 14/10/2026, autorizou seguir com a AWS. Inventário, resultados e definição pendente do ambiente estão em `docs/aws.md`.

## Escopo respondido em 14/10/2026

Fonte e análise em docs/scope-answers.md. A 21 deseja compatibilidade da API V1 sem alteração pelos clientes: individual/lotes, DLR, MO, consultas e créditos; agendamento indicado como provavelmente necessário. Usuário/senha legados mantidos; token próprio do fornecedor por cliente. Painel interno focado em credenciais, whitelist de IPs, URLs de callback e tokens, com perfil único de administrador. MFA foi considerado interessante, sem método definido. Texto até 160 sem acentos/emojis, normalização e nono dígito precisam reproduzir o legado.

O código atual não atende ainda a esse escopo completo. Respostas provisórias e ausências de documentação do fornecedor continuam explícitas na consolidação; não tratar como funcionalidades já homologadas.

Evolução MO: fluxo interno opt-in com persistência cifrada/outbox/worker implementado conforme docs/mo-flow.md. Fornecedor, consulta /v1/consultmo e retenção seguem pendentes; nenhuma entrega real foi ativada.

Consulta MO legada implementada sobre eventos locais conforme docs/legacy-mo-query.md, sem consumo/outbox alterados; contratos externos e homologação de limites seguem pendentes.

Normalização SMS posteriormente autorizada: texto sem diacríticos/emojis, single/lote, conforme docs/sms-normalization.md. Contagem na origem preservada; regra telefônica ainda pendente.

Operação evoluída conforme docs/operations.md: migração 009, retenção confirmada de conteúdo MO por 90 dias com proteção de pendências, chaves versionadas/CLI, diagnóstico opt-in, scripts de backup/restauração e benchmarks. Não confundir preparação com validação de banco/produção.

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
