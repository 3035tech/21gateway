# 21 Mobile Gateway

Gateway de SMS compatível com a API V1 da 21 Mobile e painel interno para empresas, usuários, credenciais técnicas, envios, callbacks, agendamentos e auditoria. O backend é Go com Chi, Huma, GORM e PostgreSQL; o frontend é React, TypeScript e Vite. O repositório contém as duas aplicações e os workflows que publicam suas imagens no GHCR.

## Estado da homologação

- Painel: <https://21gateway-front.3035service.com>
- API: <https://21gateway-back.3035service.com>
- Referência interativa Scalar: <https://21gateway-back.3035service.com/docs>
- OpenAPI público: <https://21gateway-back.3035service.com/openapi.json>

O OpenAPI documenta somente os endpoints de integração de clientes: `POST /v1/send`, `GET /v1/status`, `GET /v1/consultmo` e `GET /v1/auth/check`. A **21 Mobile gera e entrega** o usuário e a senha Basic Auth de cada empresa. O token Zoug é cadastrado na ficha da empresa e permanece no backend; clientes não o enviam no Basic Auth. Senhas são armazenadas como hash Argon2id, e tokens do fornecedor são cifrados. Uma credencial gerada deve ser copiada no momento da criação; a senha não é exibida novamente.

As URLs do painel para empresas, usuários e credenciais usam `public_ref` UUID, sem mostrar o ID sequencial do banco. A migração 021 já foi aplicada em HML. O backend e o frontend são publicados pelo GitHub Actions; o Argo CD acompanha as novas imagens e atualiza HML. O GitOps fica no repositório separado `3035tech/gitops`, em `dublin/21gateway-back` e `dublin/21gateway-front`.

O [relatório de segurança de HML](docs/reviews/hml-pentest-2026-10-27.md) registra testes, correções e pendências. O IP real do cliente atrás do Cloudflare Tunnel ainda requer configuração e validação da cadeia de proxies antes de ativar whitelists de clientes. Callback DLR/MO real, SQS e scheduler continuam opt-in/desligados em HML até homologação. Nenhum teste deste relatório enviou SMS.

## Desenvolvimento local

Requisitos: Docker com Compose. Na raiz do repositório:

```sh
cp .env.example .env
docker compose up --build
```

O Compose sobe PostgreSQL, aplica migrações pelo serviço `migrate` e inicia backend e frontend. Acesse <http://localhost:5173>, a API em <http://localhost:8080>, Scalar em <http://localhost:8080/docs> e o OpenAPI em <http://localhost:8080/openapi.json>. As credenciais do Compose são apenas para desenvolvimento. O processo Go lê variáveis de ambiente; não carrega `.env` automaticamente fora do Compose.

Não existe administrador padrão. Após subir o Compose, defina `ADMIN_PASSWORD` por entrada protegida ou gerenciador de segredos e execute:

```sh
docker compose run --rm -e ADMIN_PASSWORD --entrypoint /admin backend -action bootstrap-admin -name "Administrador" -email admin@suaempresa.com.br
```

O bootstrap só funciona sem usuários internos existentes. Para testar uma credencial técnica sem enviar SMS, crie uma no painel e chame `GET /v1/auth/check` da rede que usará a integração; HTTP 204 confirma autenticação e IP autorizado. Consulte o [guia de integração](docs/sms-batch.md) e a documentação Scalar antes de usar `POST /v1/send`: uma chamada válida pode enviar um SMS real. HTTP 200 nessa rota exige examinar **cada resultado**; itens inválidos podem vir como `NOK` e `code: -5` no corpo.

## Verificação

```sh
cd backend
go test -race ./...
go vet ./...
go build ./cmd/api ./cmd/admin ./cmd/maintenance
```

```sh
cd frontend
npm ci
npm run lint
npm test
npm run build
```

Para os testes de integração com PostgreSQL local:

```sh
docker compose --profile verification run --rm backend-test
```

O serviço de teste usa schema isolado. Sem `TEST_DATABASE_URL`, a suíte de banco executada fora do Compose é pulada; resultados apenas de unidade não demonstram PostgreSQL real. O workflow de [backend](.github/workflows/backend.yml) inicia PostgreSQL 18 e exige a suíte de integração, incluindo isolamento entre duas empresas; o workflow de [frontend](.github/workflows/frontend.yml) valida a interface. Ambos publicam imagens no push de `main`.

## Configuração e operação

- `DATABASE_URL` é obrigatório. As migrações são versionadas e executadas antes do novo backend; produção não usa AutoMigrate no startup. Consulte [migrações e operação](docs/operations.md).
- `PROVIDER_TOKEN_KEY` e `SMS_HISTORY_CONTENT_KEY` são chaves privadas independentes. O token Zoug é único por empresa; o histórico recente de SMS usa conteúdo cifrado. Não coloque segredos em `VITE_*` ou em documentação versionada. Consulte [configuração da conta](docs/account-settings.md).
- `RECOVERY_SMTP_*` e `PUBLIC_APP_URL` configuram convites e recuperação de senha. `GET /healthz` verifica o processo, `GET /readyz` verifica PostgreSQL e `GET /health/dependencies` verifica banco, SMTP e SQS com autenticação pelo hash configurado em `HEALTH_DEPENDENCIES_TOKEN_SHA256`.
- `TRUSTED_PROXY_CIDRS` é opcional e fica vazio por padrão. Quando configurado com os CIDRs **dos peers confiáveis da conexão**, o backend aceita `CF-Connecting-IP` somente desses peers; caso contrário usa `RemoteAddr`. A resolução é compartilhada pela whitelist, pelo limite de autenticação da API, pelo login e pela recuperação. Não inclua o CIDR inteiro do cluster sem restringir acesso direto ao serviço. Confirme o peer do Tunnel e teste duas origens antes de habilitar whitelists. Veja [configuração da conta](docs/account-settings.md).
- `SCHEDULER_ENABLED`, `SCHEDULED_SQS_ENABLED`, `CALLBACK_INGRESS_ENABLED` e `MO_INGRESS_ENABLED` são opt-in. Ativação exige chaves, migrations e contratos/filas correspondentes; veja [agendamento](docs/sms-scheduling.md), [SQS](docs/sqs-dispatch.md) e [callbacks](docs/mo-flow.md).

O backend organiza controllers, services, models, DTOs e repositories por domínio. `backend/internal/httpapi` compõe rotas e OpenAPI; `backend/internal/security` concentra a resolução segura do IP de origem. O frontend organiza funcionalidades em `frontend/src/features`, com infraestrutura compartilhada em `frontend/src/lib`. Para decisões de arquitetura, escopo do fornecedor e limitações de aceite, consulte [docs/architecture.md](docs/architecture.md), [docs/scope-answers.md](docs/scope-answers.md) e [docs/acceptance-validation.md](docs/acceptance-validation.md). Instruções para agentes estão em [AGENTS.md](AGENTS.md).
