# 21 Mobile

Gateway SMS e base do painel administrativo. O envio imediato unitário está implementado em `POST /v1/send`, traduzindo para `/api/send` do white label. Login do painel, CRUDs de empresas/usuários/credenciais, consultas administrativas e outbox/worker de callbacks estão integrados em código. O callback real do fornecedor, lotes e agendamento permanecem pendentes. O escopo completo da migração ainda está em levantamento.

## Executar com Docker

Requisitos: Docker com Compose. Na raiz:

```sh
docker compose up --build
```

Frontend: http://localhost:5173. API: http://localhost:8080. Documentação interativa Scalar, com exemplos por linguagem: http://localhost:8080/docs. Especificação pública da integração: http://localhost:8080/openapi.json, limitada a `/v1/send`, `/v1/status`, `/v1/consultmo` e `/v1/auth/check`. A 21 Mobile gera e fornece as credenciais Basic Auth de cada empresa. `/healthz` verifica o processo, `/readyz` verifica PostgreSQL e `/health/dependencies` informa PostgreSQL, SMTP e SQS. O banco persiste no volume `postgres_data`.

Compose destina-se ao desenvolvimento local. As credenciais de exemplo não são credenciais de produção. Os containers de API e frontend de produção executam sem root.

## Backend

Go 1.27, Chi, Huma (validação e OpenAPI gerados a partir dos tipos), GORM/PostgreSQL e caarlos0/env. Logs JSON via slog, timeouts HTTP, encerramento gracioso, CORS explícito e pool de conexões. Documentação desabilitada por padrão fora do Compose.

```sh
cd backend
cp .env.example .env
# Exporte as variáveis do .env no shell ou use o recurso de env-file do seu executor.
go run ./cmd/api
go test -race ./...
go vet ./...
```

O processo lê variáveis de ambiente; não carrega `.env` automaticamente. Não há AutoMigrate no startup. As migrações criam credenciais técnicas, tentativas, empresas, usuários, sessões, auditoria e outbox. Gormigrate executa migrações versionadas via comando administrativo, com transação e lock PostgreSQL. As rotas administrativas e de SMS exigem migrações aplicadas; health/readiness continuam operacionais.

## Frontend

React/TypeScript/Vite, Tailwind CSS, TanStack Query, Axios, Zod, TanStack Table, React Router e React Hook Form com resolvers. Formulários por domínio usam React Hook Form e schemas Zod. Não há tokens guardados no navegador nesta base.

```sh
cd frontend
cp .env.example .env
npm ci
npm run dev
npm run lint
npm test
npm run build
```

`VITE_API_URL` é pública e definida no build. Nunca coloque senhas ou tokens do fornecedor em variáveis `VITE_*`. O cliente HTTP tem timeout e as consultas propagam cancelamento. Mutações não têm retry automático.

## Amplify Hosting

Conecte o repositório e selecione `frontend` como raiz do aplicativo. Use o `amplify.yml` da raiz e configure `AMPLIFY_MONOREPO_APP_ROOT=frontend` e `VITE_API_URL=https://api.example.com` no Amplify. Libere a origem HTTPS do painel em `CORS_ALLOWED_ORIGINS` no backend. Não é necessário SDK Amplify para hospedagem estática.

Configure a regra de rewrite SPA recomendada pelo Amplify para rotas da aplicação, preservando arquivos estáticos. O backend e PostgreSQL precisam de hospedagem separada. Nenhum recurso AWS foi criado.

Para servir o frontend fora do Amplify:

```sh
docker build --target production --build-arg VITE_API_URL=https://api.example.com -t 21mobile-frontend ./frontend
docker run --rm -p 5173:8080 21mobile-frontend
```

## Organização

- `backend/cmd/api`: composição e ciclo de vida do processo.
- `backend/internal/config`: configuração tipada.
- `backend/internal/database`: conexão GORM e pool PostgreSQL.
- `backend/internal/httpapi`: router, middlewares e OpenAPI.
- `backend/internal/controllers`: transporte HTTP e conversão de erros para respostas.
- `backend/internal/services`: regras de negócio, sem dependência do transporte HTTP.
- `backend/internal/models`: modelos de persistência GORM, separados dos contratos externos.
- `backend/internal/dtos`: entradas e saídas da API, validação e schemas OpenAPI.
- `frontend/src/app`: providers da aplicação.
- `frontend/src/lib`: cliente HTTP e infraestrutura compartilhada.
- `frontend/src/features`: funcionalidades do painel.

Autenticação Basic Auth, integração de envio e armazenamento de tentativas estão disponíveis. A aplicação não armazena o texto do SMS. Autenticação do painel, empresas e worker de callbacks estão implementados; veja docs/backoffice.md. Retentativas de envio SMS não devem ser ativadas sem definir idempotência e reconciliação.

## Padrões obrigatórios

Backend em MVC com controllers, services, models e DTOs separados. Códigos de erro, status e direções de ordenação são tipos/constantes; mensagens públicas ficam centralizadas. A API administrativa usa Problem Details com `code` estável; contratos legados terão adaptação própria, sem alterar seus formatos por conveniência.

Usuários internos e empresas possuem CRUDs próprios. A base compartilhada `DataTable` e `useListState` já suporta filtros, paginação, tamanho de página e ordenação pelo cabeçalho. Os dados serão paginados e ordenados no servidor; os endpoints são protegidos por sessão e permissões.

Instruções de IA: `AGENTS.md`, `CLAUDE.md` e `.cursor/rules/project.mdc`. Contexto e decisões: `docs/`. Leia esses arquivos antes de implementar.

## Preparar o envio unitário

Copie `.env.example` da raiz para `.env` e configure `PROVIDER_TOKEN_KEY` localmente e o token Zoug de cada conta pelo painel. A chave deve ser base64 de 32 bytes aleatórios. Sem chave/token da conta, um envio autenticado retorna indisponibilidade e não chama o fornecedor. `WHITELABEL_API_TOKEN` global não é usado pelo runtime.

Com Docker ativo:

```sh
docker compose up --build -d
docker compose run --rm --entrypoint /admin backend -action migrate
```

Defina `CLIENT_PASSWORD` com uma senha de pelo menos 12 bytes no ambiente do terminal, usando seu gerenciador de segredos ou entrada protegida. O comando não aceita senha por argumento nem a imprime:

```sh
docker compose run --rm -e CLIENT_PASSWORD --entrypoint /admin backend -action create-credential -username cliente-teste
```

O hash Argon2id é persistido; não há senha padrão ou criação automática de cliente. Fora do Docker, os mesmos comandos são `go run ./cmd/admin -action migrate` e `go run ./cmd/admin -action create-credential -username cliente-teste`, com DATABASE_URL e CLIENT_PASSWORD configurados.

Exemplo de requisição (substitua o destinatário pelo número autorizado para seu teste). Com token válido, esta chamada envia um SMS real. `curl --user` solicita a senha sem colocá-la no argumento:

```sh
curl --user cliente-teste http://localhost:8080/v1/send \
  -H 'Content-Type: application/json' \
  --data '{"sms":[{"messageText":"Ola! Seu codigo e 1234","destination":"11987654321","correlationId":"teste-001"}]}'
```

Resposta de aceite:

```json
[{"status":"OK","code":5,"id_sms":"1","correlationId":"teste-001"}]
```

Leia `docs/sms-single.md` para contrato, limites, erros, encoding, persistência e resultado incerto. O aceite não confirma entrega no aparelho. O limitador desta etapa é local ao processo; execução com múltiplas réplicas exige limitador compartilhado.

## Primeiro acesso ao painel real

O Compose executa a migração antes do backend, que inicia API e workers no mesmo processo. Não há administrador padrão. Configure ADMIN_PASSWORD no ambiente por entrada protegida ou gerenciador de segredos e execute:

```sh
docker compose run --rm -e ADMIN_PASSWORD --entrypoint /admin backend -action bootstrap-admin -name "Administrador" -email admin@suaempresa.com.br
```

O bootstrap só funciona quando ainda não há usuários internos; não sobrescreve contas existentes. Em http://localhost:5173, entre com o e-mail e senha escolhidos. Empresas e credenciais externas são gerenciadas no painel. /demo preserva a prévia fictícia no desenvolvimento.

Consulte docs/backoffice.md para permissões, segurança, campos, callbacks e limitações. Em produção, COOKIE_SECURE=true, CORS_ALLOWED_ORIGINS com origens HTTPS exatas e COOKIE_CROSS_SITE=true somente se os sites diferirem.

## Verificar a integração com PostgreSQL

Com Docker disponível:

```sh
docker compose --profile verification run --rm backend-test
```

Esse teste usa schema exclusivo no PostgreSQL local e remove apenas esse schema. Fora do Docker, definir TEST_DATABASE_URL e executar go test -race ./.... Sem a variável, o teste de banco é pulado. Nesta sessão o socket Docker negou acesso; API/banco/worker ainda não foram iniciados e não houve homologação real do fornecedor.

## Configuração por conta e lotes

Consulte docs/account-settings.md e docs/sms-batch.md. Configure PROVIDER_TOKEN_KEY antes de salvar tokens de clientes; token não é devolvido pelo painel. Lote de até 200 implementado com resultados individuais e sem retry automático. Whitelist usa IP da conexão, exigindo política de proxy confiável na implantação. Normalização legada, contratos MO/consultas/agendamento e homologação real permanecem pendentes. Escolha do framework em docs/framework-decision.md.

### Respostas MO internas

Fluxo interno normalizado, conteúdo criptografado, outbox e worker disponíveis conforme [docs/mo-flow.md](docs/mo-flow.md). Aplicar migração 007. O backend usa MO_CONTENT_KEY independente da chave de tokens. MO_INGRESS_ENABLED permanece desligado por padrão; não representa integração pronta com os callbacks Zoug. A política de retenção e o contrato externo ainda precisam de homologação.

Consulta de respostas locais: GET /v1/consultmo?date=<Unix em segundos>, Basic Auth da credencial externa. Migração 008 e limites de homologação em [docs/legacy-mo-query.md](docs/legacy-mo-query.md).

Envio SMS normaliza diacríticos e emojis em single/batch, mantendo limite de 160 na origem. Regra de nono dígito ainda pendente. Detalhes em [docs/sms-normalization.md](docs/sms-normalization.md).

### Operação

Retenção MO de 90 dias, chaves versionadas, diagnóstico protegido, manutenção/backup e testes de desempenho em [docs/operations.md](docs/operations.md). Aplicar migração 009. Chaves antigas continuam necessárias durante a transição; nenhuma infraestrutura AWS é criada automaticamente.

## Agendamento local

POST /v1/send aceita scheduledDate futuro na fila local quando SCHEDULER_ENABLED=true. Requer migração010 e chave SCHEDULED_CONTENT_KEY privada no backend. Opt-in desligado por padrão; consultar docs/sms-scheduling.md para ativação, concorrência, resultado incerto e limites de homologação. Texto de SMS imediato não é persistido; texto agendado fica cifrado até término.

## Repositório GitHub preparado

O GitHub terá um único repositório com `backend/` e `frontend/`; ainda não há remoto configurado nem repositório publicado. Para subir apenas o código das duas aplicações, incluir também `.github/` e `.gitignore` da raiz. O `.gitignore` exclui dependências, builds, binários, caches, logs e arquivos locais de ambiente dentro das duas pastas, preservando `.env.example`.

Os workflows da raiz [backend.yml](.github/workflows/backend.yml) e [frontend.yml](.github/workflows/frontend.yml) verificam e compilam cada aplicação. Pull requests executam verificações e build Docker; push em `main` publica imagens `sha-<commit>` no GHCR, e tags `vX.Y.Z` também publicam `X.Y.Z`. A tag `v0.1.0` corresponde às referências atuais do GitOps. As imagens continuam `ghcr.io/3035tech/21gateway-back` e `ghcr.io/3035tech/21gateway-front`. O frontend incorpora `https://21gateway-back.3035service.com` durante o build; revisar esse domínio antes de publicar. A publicação exige permissão `packages: write` para o `GITHUB_TOKEN` e acesso do cluster aos pacotes privados.

Despacho agendado opcional com scheduler Go + SQS: [arquitetura, configuração e validações pendentes](docs/sqs-dispatch.md). Aplicar migração011 antes dos novos binários; fila AWS não criada.
