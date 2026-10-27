# Desenvolvimento e verificação

## Executar

Na raiz do projeto: `docker compose up --build`. Requer daemon Docker ativo. Frontend localhost:5173; API localhost:8080; Swagger /docs; OpenAPI /openapi.json. Banco localhost:5432, database 21mobile. Credenciais do Compose são somente exemplos locais. Variável POSTGRES_PASSWORD pode substituir a senha; evitar caracteres especiais sem encoding na URL da API.

O Compose executa a migração dedicada antes do backend. O processo `/api` executa HTTP, callbacks, retenção MO e, quando habilitados, MO e scheduler/SQS no mesmo contêiner. Criar primeiro administrador e credenciais conforme README. Sem token, o endpoint responde indisponibilidade e não tenta envio.

Node 24 e Go 1.27 são usados pelos Dockerfiles. Sem Docker, instalar dependências no frontend com `npm ci`; executar `npm run dev`. Copiar `.env.example` para `.env` no frontend; Vite lê esse arquivo.

Backend lê variáveis de ambiente tipadas com caarlos0/env e não carrega `.env` automaticamente. Usar os exemplos exportados pelo executor/shell. PostgreSQL deve estar disponível; startup exige conexão válida.

## Verificar

Backend: `go test -race ./...`, `go vet ./...`, `go build ./cmd/api`. Frontend: `npm run lint`, `npm run build`, e `npm test`. Compose: `docker compose config --quiet`. Exercitar banco/containers quando Docker estiver ativo, sem remover volumes para resolver problemas de configuração.

Dependências ficam em go.mod/go.sum e package.json/package-lock.json. Usar versões fixadas pelo lock e npm ci em CI. Não instalar novamente bibliotecas já disponíveis para a mesma responsabilidade. Não alterar arquivos legados ou segredos para completar verificações. Nunca editar uma migração já aplicada; criar uma nova versão e avaliar reversibilidade/compatibilidade.

## Amplify

Conectar repositório, app root frontend; definir AMPLIFY_MONOREPO_APP_ROOT=frontend e VITE_API_URL com domínio HTTPS da API. Liberar origem do painel no CORS do backend. Configurar rewrite SPA com exceção para assets estáticos. O README contém comandos de build/container. Não publicar como parte de tarefas de estruturação.

Frontend inclui `npm run format` e `npm run format:check` com Prettier. Backend usa gofmt.

## Convenções

Identificadores de código em inglês; textos do painel e mensagens públicas em português. DTOs explícitos e modelos separados. Enums para estados/códigos/direções; constantes nomeadas para limites. Mapear ordenação para colunas permitidas no backend. Não concatenar filtros do usuário em SQL.

Usar a DataTable compartilhada nos cadastros. Os CRUDs atuais têm campos, validações e permissões definidos em docs/backoffice.md; novas mudanças devem preservar esses contratos. Tratar estados vazio, erro e carregamento. Não simular sucesso de cadastro enquanto persistência/auth estiverem ausentes.

Atualizar contexto, arquitetura e status quando houver decisão nova. Todos os agentes seguem AGENTS.md; CLAUDE.md e Cursor apontam para os mesmos documentos.

## Prévia de telas com dados fictícios

Com Vite ativo na porta 5173, abrir http://localhost:5173/demo. Para iniciar uma instância dedicada: `cd frontend && npm run dev:demo`; porta 5174, modo demo. A configuração `.env.demo` contém somente o sinal público VITE_DEMO_MODE=true. Nenhuma credencial é necessária.

Na sessão de revisão, o servidor existente em 5173 permaneceu disponível. A criação de um segundo listener local falhou com EPERM; Compose também falhou por permissão de acesso ao socket Docker. Backend e PostgreSQL não foram iniciados nessa sessão; a prévia frontend funciona independentemente deles.

## Painel integrado

Rotas reais em /, /login, /empresas, /usuarios, /credenciais, /envios, /callbacks e /auditoria. Criar primeiro admin via cmd/admin -action bootstrap-admin, -name e -email, com ADMIN_PASSWORD somente no ambiente. Compose mantém COOKIE_SECURE=false apenas para HTTP local. Fora do Compose, exportar essa variável ao usar CORS local; produção exige cookies seguros/origens HTTPS.

Verificação completa de banco: docker compose --profile verification run --build --rm backend-test. O teste cria schema exclusivo e comprova CRUDs, migrações e outbox, sem usar fornecedor real. Sem Docker, TEST_DATABASE_URL em formato URL e go test -race ./.... Teste é pulado se a variável não existir; registrar isso ao reportar validação.

Não declarar backend/banco/worker rodando com base em build/config YAML. Nesta sessão, o socket Docker continuou inacessível; a prévia /demo continua independente.

## Verificação obrigatória com PostgreSQL

O target Docker verification inclui build-base/CGO para o detector de races. Compose define REQUIRE_DATABASE_TESTS=true: ausência de TEST_DATABASE_URL é falha, nunca skip. Execução fora do container também pode definir essa variável para exigir banco. O modo usual de testes sem banco continua pulando a integração explicitamente.

O teste com banco verifica URLs DLR/MO persistidas, migrações reaplicáveis, outbox, limites compartilhados, polling 100+1 e duas consultas simultâneas consumindo uma atualização uma única vez. Cada execução cria/remove somente seu schema isolado. O comando inicia PostgreSQL se necessário e deixa o serviço disponível; não remove volumes. Build e execução do container ainda dependem de acesso ao Docker.

MO evoluído com domínio próprio, eventos/outbox em tabelas separadas, chave MO_CONTENT_KEY independente, ingresso interno opt-in e worker integrado. Aplicar migração 007; configuração, segurança e limites em docs/mo-flow.md. Cadastro da URL não ativa recebimento do fornecedor.

Operação evoluída conforme docs/operations.md: migração 009, retenção confirmada de conteúdo MO por 90 dias com proteção de pendências, chaves versionadas/CLI, diagnóstico opt-in, scripts de backup/restauração e benchmarks. Não confundir preparação com validação de banco/produção.

Migração 012: amplia a restrição de `send_reservations.messages` para `0..200`, permitindo registrar uma requisição de agendamento sem envio imediato. Aplicá-la antes dos binários atualizados; a suíte PostgreSQL com detector de corrida validou criação, reaplicação e orçamento compartilhado em 27/10/2026.

Migração 013: índices B-tree em `(created_at,id)` para as listagens administrativas de SMS e callbacks. Aplicar até 013 antes dos binários atualizados. Buscas com `ILIKE '%texto%'` e contagens de histórico completo ainda exigem medição de carga; não adicionar GIN/trigram ou rollup sem validar cardinalidade, permissões e custo de escrita.

Migração 014: índices B-tree em `scheduled_sms(due_at,id)` e `(credential_id,due_at,id)` para a tela de agendamentos. Aplicar antes do backend com `/admin/schedules`. A busca livre por referência/destinatário e o resumo histórico ainda exigem medição com volume representativo.

Validação de aceite ampliada em 15/10/2026: consultar `docs/acceptance-validation.md` para cobertura executada, suíte PostgreSQL pendente e regra de bloqueio de callbacks DLR/MO após troca da URL. Mantida separação por domínio; sem camadas ou dependências novas.
