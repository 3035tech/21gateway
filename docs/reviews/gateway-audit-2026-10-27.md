# Auditoria do gateway e da preparação GitOps — 27/10/2026

## Escopo e método

Revisão estática de backend Go, frontend React, migrações, Docker/Compose e manifests preparados em `workspace/3035/gitops` para Dublin. Execução de Helm/Kustomize, frontend, Compose, `go vet`, `go build`, suíte Go com detector de races e PostgreSQL isolado, `govulncheck` e `npm audit`. Nenhum SMS/callback externo, sync ArgoCD, imagem GHCR ou alteração do cluster foi executado. Achados distinguem falha reproduzida, inferência de topologia e risco sem medição de carga.

## Achados priorizados

### A1 — Alto: origem de IP perde identidade atrás do ingress (risco de implantação)

`LegacyGuard.Authenticate` passa `request.RemoteAddr` para o limite pré-autenticação e para a whitelist (`backend/internal/controllers/legacy_guard.go:19-25`). `AuthService` usa o peer IP como chave (`backend/internal/services/auth.go:54-64,123-127`). O ingress GitOps passa pelo controlador Cloudflare Tunnel (`dublin/21gateway-back/helm-values.yaml`), portanto o endereço observado pelo Go tende a ser o proxy, compartilhado entre clientes. A inferência precisa ser confirmada em tráfego controlado no cluster. Se confirmada, uma whitelist por cliente deixará de representar o IP do cliente e o orçamento padrão de 600 autenticações/minuto será compartilhado. Não liberar integrações legadas até estabelecer uma cadeia de proxy confiável e testar IP real, whitelist e 429 por cliente.

### A2 — Alto: credencial padrão falha no PostgreSQL real (reproduzido)

`allowed_ips` é `JSONB NOT NULL` (`backend/internal/database/migrations/202610140004_account_settings.sql:3`). O model tem slice sem valor padrão (`backend/internal/models/credential.go:7`); `SaveCredential` deixa o slice `nil` quando o campo opcional é omitido (`backend/internal/services/credential.go:81-95,155-164`). GORM tenta persistir `null`, e PostgreSQL recusa. `TestPostgresBackofficeAndOutbox` chegou ao fluxo real e retornou `SERVICE_UNAVAILABLE` na criação (`backend/internal/services/postgres_integration_test.go:48-51`). A UI envia `allowedIPs: []` (`frontend/src/features/credentials/api.ts:125-130`), mas outros clientes administrativos e o serviço podem omitir o campo. Corrigir a inicialização/persistência no domínio de credenciais e provar ambos os casos no banco.

### A3 — Alto: segredo de pull do GHCR contém token literal no GitOps existente

`workspace/3035/gitops/base/apps/manifests/external-secret-github-token.yaml:8-16` declara um gerador GitHub, mas o template `.dockerconfigjson` contém um token literal, sem usar o valor gerado. A rotação pelo gerador não atualiza a credencial realmente renderizada. Não se verificou se o token está ativo; tratar sua presença em Git como exposição, validar/revogar se aplicável e fazer o template consumir o valor gerado. Outros manifests legados no GitOps também contêm valores com formato de credencial; não foram reutilizados no gateway.

### A4 — Alto: toolchain Go da imagem com vulnerabilidade alcançável

`backend/Dockerfile:1` compila com Go 1.27.2. `govulncheck ./...` encontrou GO-2026-6809 em `crypto/x509`, com caminho alcançável a partir do cliente HTTPS do fornecedor; a [base oficial Go](https://pkg.go.dev/vuln/GO-2026-6809) informa correção em Go 1.27.3. Atualizar a imagem de build e recompilar todos os binários antes de publicar. O scan encontrou zero vulnerabilidades alcançáveis nas bibliotecas importadas e uma vulnerabilidade de módulo sem caminho chamado.

### A5 — Médio: fluxo V1 ainda incompleto para migração

O router registra `/v1/send`, `/v1/status` e `/v1/consultmo` (`backend/internal/httpapi/router.go:100-107`), mas não `/v1/status/date` nem `/v1/credits`. Ingress DLR/MO é contrato interno normalizado, com flags desligadas por padrão (`backend/internal/controllers/callback.go:42-43`, `backend/internal/controllers/mo.go:17`). Contratos reais do fornecedor e regra do nono dígito continuam pendentes (`docs/scope-answers.md`, `docs/next-session.md`). A configuração GitOps não resolve essas lacunas. Não conduzir piloto/corte com base apenas em migração de schema ou build verde.

### A6 — Médio: troca de URL durante callback pode escapar do bloqueio (inferência)

DLR/MO verificam se a URL do job ainda é a cadastrada ao carregar dados (`backend/internal/repositories/callbacks.go` e `backend/internal/repositories/mo.go:103-114`). O worker faz a chamada HTTP depois, sem manter uma versão/lock da configuração (`backend/internal/services/callback_worker.go:93-104`, `backend/internal/services/mo_worker.go:70-87`). Uma troca concorrente entre verificação e envio pode entregar ao destino antigo. Não há teste dessa interleaving; confirmar política operacional de troca de URL e usar versionamento/fence antes de prometer bloqueio estrito.

### A7 — Médio: limite global serializa envios no PostgreSQL (risco de vazão)

Cada reserva de SMS obtém o mesmo advisory lock global, executa limpeza e agrega a janela de um segundo (`backend/internal/repositories/send_limits.go:45-97`). O desenho garante coordenação entre réplicas, mas serializa o caminho de envio de todas as contas. Não há medição PostgreSQL sob carga nem EXPLAIN no volume alvo; medir espera de lock, latência p95/p99 e TPS antes de afirmar capacidade de 200 mensagens/s. Uma requisição de lote reserva todo o tamanho antes de filtrar itens locais (`backend/internal/services/sms_batch.go:45-75`), o que pode consumir orçamento de itens rejeitados sem envio.

### A8 — Médio: resumo faz agregações sobre histórico completo (risco de degradação)

O painel executa contagens filtradas em todas as linhas de `sms_attempts` e `callback_jobs` a cada consulta (`backend/internal/repositories/summary.go:16-48`), com timeout de cinco segundos. Não há retenção de metadados aprovada nem tabela de agregação. Com crescimento do histórico, a tela pode ficar indisponível e competir com consultas operacionais. Medir com cardinalidade representativa; somente depois escolher rollup, cache ou recorte temporal explícito.

### A9 — Médio: suíte PostgreSQL ainda não comprova os fluxos completos

`docker compose --profile verification run --build --rm backend-test` executou PostgreSQL real e terminou com cinco testes de serviço falhando. Quatro falham ao inserir `APICredential` sem `AllowedIPs` (`acceptance_test.go:129-136`, `dispatch_postgres_test.go:14-19`, `scheduled_postgres_test.go:14-19`, `security_postgres_test.go:85-89`); o teste de backoffice falha no produto descrito em A2. A primeira execução também falhou no limite de login após dez tentativas; repetição isolada passou dessa linha e chegou ao erro de `allowed_ips`. A janela de autenticação é minuto fixo do PostgreSQL (`backend/internal/repositories/authentication.go:25-34`), enquanto o teste pressupõe que dez hashes ocorram no mesmo minuto (`security_postgres_test.go:26-34`), tornando essa asserção sensível à virada. Corrigir fixture/asserção e repetir a suíte completa sem skips.

### A10 — Baixo para produção: dependência de teste frontend vulnerável

`npm audit` encontrou `undici@8.12.1` via `jsdom@30.1.2`; a [advisory de maior severidade](https://github.com/advisories/GHSA-xjj6-f33q-929c) aponta correção em 8.12.2. `npm audit --omit=dev` encontrou zero vulnerabilidades, portanto o pacote não integra a imagem Nginx de produção. Atualizar o lock de desenvolvimento e repetir audit/testes.

### A11 — Médio: progresso do worker não integra o readiness do backend

O pod conjunto usa `/readyz` apenas na API e não define probe de progresso para o contêiner worker (`workspace/3035/gitops/dublin/21gateway-back/helm-values.yaml`). O processo worker executa loops de callbacks, MO, retenção e scheduler (`backend/cmd/worker/main.go:46-91`). Um processo vivo, mas sem progresso, pode deixar a API pronta enquanto a fila atrasa. Contadores operacionais já existem, mas coleta e alertas ainda não estão configurados. Definir alerta para idade da outbox, jobs pendentes e falhas persistentes; a simples saúde da API não comprova entrega.

### A12 — Baixo: headers de segurança do frontend ainda dependem da borda

`frontend/nginx.conf:1-8` define `nosniff` e Referrer-Policy, sem CSP ou HSTS. O GitOps registra ingress via Cloudflare Tunnel, mas não contém política específica de HSTS para os novos hosts. Validar os headers efetivos na borda e aplicar política compatível com o painel após teste em navegador; configuração do backend não cobre HTML/JS servido pelo Nginx.

## Preparação GitOps

Há duas Applications ArgoCD, `21gateway-front` e `21gateway-back`. O backend tem um único Deployment com API e worker Go no mesmo pod; migrações em Job PreSync. O chart compartilhado ganhou `extraContainers` opcional. `fullnameOverride` mantém Service/Ingress válidos embora as Applications comecem por dígito. Tokens de ServiceAccount não são montados automaticamente nos novos pods.

Ainda faltam imagens GHCR com tags reais, o build do frontend com `VITE_API_URL` correto, banco/usuário isolados e Secret próprio `21gateway-back-runtime`. Os hostnames `.3035service.com` são a suposição de configuração ainda sujeita à confirmação. Scheduler, SQS e ingress DLR/MO ficaram desligados. Não houve push, sync, criação de banco/Secret ou teste da borda Cloudflare. O Job de migração e a API não iniciarão sem essas dependências.

## Validação executada

- GitOps: `helm lint` frontend/backend e app existente; `helm template` confirmou um pod backend com dois contêineres; `kubectl kustomize dublin` e `git diff --check` passaram.
- Backend: `go vet ./...` e `go build ./...` passaram dentro do contêiner; `go test -race ./...` com PostgreSQL falhou nos casos acima. Go não estava no PATH local. `govulncheck` encontrou A4.
- Frontend: lint, build e 17 testes passaram. `npm audit` encontrou A10; `npm audit --omit=dev` passou.
- Compose: `docker compose config --quiet` passou. O perfil de verificação iniciou PostgreSQL real; isto comprova conexão/migrações executadas pela fixture, não os fluxos de aceite que falharam.

## Limites desta auditoria

Sem DAST/pentest, análise de tráfego real, teste de carga, execução de SQS, inspeção de permissões Cloudflare/GHCR em produção ou homologação com fornecedor e clientes. Riscos de concorrência e desempenho indicados como inferência requerem reprodução ou medição; nenhum dado de cliente foi processado.

## Correções verificadas em 27/10/2026

- A2/A9: o domínio de credenciais normaliza `allowed_ips` ausente para `[]`; a suíte PostgreSQL completa agora passa com `go test -race ./...`. O teste de bloqueio de autenticação tolera a virada da janela de um minuto.
- A4: a imagem de build usa Go 1.27.3. Os quatro binários compilaram no Docker. Repetir `govulncheck` na cadeia de CI antes da publicação.
- A10: lockfile atualizado; `npm audit` encontrou zero vulnerabilidades. Lint, build e 17 testes frontend passaram.
- Dois defeitos adicionais expostos pela suíte foram corrigidos: a listagem de envios usava uma seleção SQL inválida para GORM; a reserva de uma requisição sem SMS precisava admitir `messages=0`. A migração versionada 012 altera somente essa restrição e deve ser aplicada antes dos binários novos. Migrações reaplicáveis e fluxos de PostgreSQL passaram na suíte.
- A migração 013 acrescenta índices de ordenação em `(created_at,id)` para as listagens administrativas de SMS e callbacks; foi exercitada na suíte PostgreSQL. Buscas por trecho com `ILIKE` e contagens globais ainda dependem de cardinalidade/`EXPLAIN` representativos. Os novos índices não demonstram vazão para 100 mil SMS agendados.
- A12: CSP e HSTS foram configurados no Nginx do frontend, inclusive para `/assets/`. A imagem de produção compilou e `nginx -t` passou. Falta conferir headers efetivos e navegação pela borda Cloudflare.

A1, A3, A5–A8 e A11 continuam abertos nos limites descritos acima. A1 exige conhecer e testar a cadeia real de proxies antes de confiar em cabeçalhos de origem. A3 afeta um segredo compartilhado do GitOps existente e requer rotação operacional se o token literal ainda estiver válido. A7/A8 exigem medição sob carga representativa para escolher uma mudança de consulta ou de coordenação. A11 exige configuração operacional; A12 depende da verificação na borda. Nenhuma dessas pendências autoriza corte de produção.
