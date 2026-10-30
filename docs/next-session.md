# Ponto de retomada — 15/10/2026

## Liberação HML — 29/10/2026

O commit `e2385b0` está na `main`; os workflows de backend e frontend passaram. O GitOps de Dublin sincronizou `5ca5b45`, executou o Job de migração com a mesma imagem e atualizou o worker SQS sem alternar o transporte. Consulta direta ao banco `gateway21_hml` confirmou 25 migrações, com `202610290025` como última. Argo CD mostrou backend e frontend saudáveis, worker em execução e `/readyz` respondeu 200. O detalhe real de Envios foi inspecionado no navegador após o ajuste visual da linha do tempo. Isso não constitui novo aceite de SMS/MO ponta a ponta nem prova de alertas externos ou backup recorrente. A credencial SQS ainda está no Secret versionado no histórico GitOps; a transição para ExternalSecret e a rotação aguardam bootstrap seguro no cluster.

## Estado mais recente — 29/10/2026

Há código novo para histórico por SMS, saúde da integração por empresa e avisos operacionais no painel do administrador. Aplicar migração 025 antes do backend novo; ela cria índices dentro da transação do migrador, portanto avaliar janela/volume de dados. Consulte `docs/product-observability-2026-10-29.md` para contratos e limites. Backup cifrado e restauração passaram no PostgreSQL local; backup recorrente de produção, entrega externa de alertas, aceite real Zoug → gateway → cliente e rotação/limpeza do segredo SQS histórico continuam pendentes. Não declarar esses itens homologados com base nos testes simulados.

Atualização de 27/10/2026: aplicar migrações 018 (vínculos usuário/empresa) e 019 (histórico SMS cifrado) antes da imagem nova. Configurar chave independente `SMS_HISTORY_CONTENT_KEY` em segredo de produção; a API falha ao iniciar sem ela. O conteúdo novo fica visível somente no detalhe administrativo por até 90 dias; histórico antigo não contém texto. O worker integrado expurga em lotes. Confirmar coleta/backup de chaves e política de acesso antes do deployment; nenhum SMS real foi enviado.

Atualização posterior: aplicar também migração 020 para token Zoug único por empresa. O painel cadastra esse token na ficha da empresa; cada empresa pode ter várias credenciais Basic Auth de API. Tokens individuais antigos são preservados em fallback somente até cadastrar ou remover explicitamente o token da empresa. Para rollback do binário depois de usar o token centralizado, recriptografar e restaurar tokens individuais antes; não pressupor que a imagem antiga leia `companies.provider_token_encrypted`. Sem SMS real ou deploy.

Polimento posterior de interface: formulário de credencial da prévia alinhado ao fluxo real de Basic Auth, geração fictícia de username/senha e diálogo compartilhado; empresa fixa em campo somente leitura no painel real; links diretos para usuário/credencial com estados de carregamento e erro. Sem migração ou alteração de contrato.

Controles posteriores: selects substituídos em todo o frontend por menu compartilhado personalizado, inclusive dentro de diálogo nativo; checkboxes e rádios visíveis com estilo da marca. Sem alteração de backend ou migração.

## Estado atual

Go/React com camadas por domínio, credenciais externas preserváveis, tokens criptografados por conta, whitelist e URLs separadas DLR/MO. Envio imediato unitário/lote até 200 e limitador compartilhado PostgreSQL. Normalização de acentos/emojis implementada; regra do nono dígito ainda aguarda corte e exemplos completos.

DLR/MO possuem ingresso interno normalizado, outbox e worker; contratos reais do fornecedor não homologados. Consultas locais `/v1/status` e `/v1/consultmo` implementadas com ressalvas documentadas. Jobs de callback são bloqueados se a URL cadastrada mudar, sem redirecionamento automático dos jobs antigos.

Migrações 001–014 versionadas. Retenção do conteúdo MO de 90 dias confirmada e implementada, preservando jobs ativos. Keyrings e CLI de recriptografia, diagnóstico opt-in, scripts de backup/restauração e benchmarks preparados. Metadados/backups têm políticas separadas ainda não definidas. Não repetir implementações já concluídas.

Segurança evoluída: proteção pré-autenticação API e bloqueio por conta, senhas internas novas com 15 caracteres, testes de ataques locais e SQL DryRun. Ler docs/reviews/security-assessment-2026-10-15.md antes de publicar; validar proxy/IP real e novo orçamento que inclui tráfego válido. Senhas legadas não alteradas.

Agendamento local implementado com opt-in, fila cifrada e worker em lotes até200; migração010 e chaves nas duas aplicações necessárias. Consultar docs/sms-scheduling.md antes de ativar; PostgreSQL e carga real ainda pendentes.

## Evidências e limites

Go race/vet/build, frontend lint/build e 17 testes passaram. Teste sem rede integra serviços/adaptadores de envio, DLR e MO. Três testes de segurança dos scripts usam ferramentas simuladas.

Suíte PostgreSQL ampliada para migrations, leases, falhas/backoff, deduplicação, isolamento, troca de URL, rotação e retenção. A execução real com detector de corrida passou em 27/10/2026; o registro anterior de falta de acesso Docker foi superado. Backup/restauração reais e SQS externo continuam sem prova de execução.

Evidência de falhas DLR/MO implementada: status HTTP/código/duração e IDs seguros; conclusão distingue entrega de persistência e registra recuperação. Worker em JSON/stdout, sem log de sucesso inicial. Consultar docs/operations.md; coleta/retenção/alertas ainda dependem da operação.

Nenhum deployment, importação de contas, SMS/callback real, alerta ou contato externo executado.

## Próximos passos

1. Repetir `docker compose --profile verification run --build --rm backend-test` após mudanças posteriores e em CI; a suíte PostgreSQL passou localmente em 27/10/2026.
2. Exercitar backup criptografado e restauração em banco isolado com ferramentas reais, conforme `docs/operations.md`.
3. Obter contratos Zoug: DLR/MO e autenticação, saldo e idempotência; agendamento agora é local, conforme docs/sms-scheduling.md. Portal técnico ainda não fornecido; solicitações em `docs/contract-requests.md` não enviadas. Não inventar rotas/campos ou filas de envio tardio.
4. Confirmar paginação de `/v1/status/date` com duas páginas e última resposta, filtros, ordem, fuso e grafia correlationid/correlationId. Exemplos oficiais não resolvem a continuação.
5. Fechar importação de credenciais/configurações e Master/Sub; confirmar regra telefônica, política de retry callback e demais contratos legados provisórios.
6. Preparar homologação externa/piloto após definir conta, destinos controlados, aprovador, janela e retorno. Não duplicar SMS para comparação.

## Migração confirmada e decisões abertas

Questionário final: operação indicada 3035/Mozhtech, ainda sem divisão de responsabilidades; conclusão desejada em outubro, sem dia de corte. Teste interno → piloto de produção → demais clientes em conjunto. Infinitech/grupoamaro são candidatos, sem seleção. Aceite exige envio SMS e entrega de DLR/MO completos até o cliente.

Tratamento de agendamentos/envios/callbacks antigos e rollback continuam a discutir. Não descartar ou reenviar mensagens por suposição. A resposta sem requisitos adicionais de segurança não revoga a retenção MO de 90 dias nem a segurança implementada.

## Leitura para retomar

`AGENTS.md`, `docs/status.md`, `docs/scope-answers.md`, `docs/migration-plan.md`, `docs/acceptance-validation.md`, `docs/contract-discovery.md` e `docs/operations.md`. Status mantém histórico de etapas; este documento consolida o estado atual.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migrações até 014 obrigatórias nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. PostgreSQL local validado; SQS real/carga ainda pendentes. Testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.

Polish SQS: publicador prioriza outbox pronta antes de preparar novos lotes, isolando falhas de preparação; regressão reproduzida e corrigida. Sem mudança de contrato, schema ou infraestrutura. Ver docs/sqs-dispatch.md.

## Preparação Dublin e auditoria — 27/10/2026

GitOps local preparado com Applications `21gateway-front` e `21gateway-back`; backend usa API e workers no mesmo processo/contêiner/pod, com Job PreSync de migração. Nenhum sync, imagem publicada, banco ou Secret foi criado. Domínios/imagens presumidos e dependências estão em docs/aws.md e no README do app GitOps.

Visibilidade administrativa nova: dashboard real com indicadores de fila/envio/callbacks/acessos e tela `/agendamentos` com cliente, credencial técnica, datas, estado, filtros e paginação. Aplicar migração 014 antes do backend atualizado. A pessoa física que usou a credencial não é conhecida; a tela não a atribui. Interface conferida no Chrome com API simulada e consulta/indicadores em PostgreSQL local. SQS real, carga e deployment não foram validados.

Auditoria de interface em `docs/reviews/ui-standardization-2026-10-27.md`: sete listagens reais revisadas; ordenação ampliada para todos os cabeçalhos de dados, nomes de cliente em envios/callbacks, busca e rolagem em tabelas largas refinadas. As novas ordenações e buscas exigem medição com volume de produção antes de definir metas de latência.

Polish posterior: Auditoria exibe nome atual do responsável com ID e filtros por responsável, ação e recurso; permanece sem snapshot histórico do nome. Sem migração adicional. Consulta administrativa e interface validadas com PostgreSQL local e API simulada; volume de produção ainda não medido.

Auditoria em docs/reviews/gateway-audit-2026-10-27.md. Correções de `allowed_ips`, histórico de envios, reserva de requisição sem SMS (migração 012), índices de ordenação (013), Go 1.27.3, dependência de teste frontend e headers Nginx foram aplicadas. `go test -race ./...` com PostgreSQL real, `go vet`, build Go, frontend lint/build/17 testes, `npm audit`, build/sintaxe da imagem Nginx e Compose config passaram. Antes de implantação: aplicar migrações até 014, validar IP de cliente/whitelist e headers efetivos atrás do Cloudflare Tunnel, concluir contratos e credenciais externas. Helm/Kustomize da preparação Dublin também passaram; nenhum deploy foi feito.

Atualização de 27/10/2026: credenciais múltiplas por empresa (username único), geração aleatória ou entrada manual de username/senha, expiração opcional e bloqueio em autenticação/teste/scheduler. Migração 017 obrigatória antes do backend novo. Painel e erros administrativos ganham catálogos `pt-BR`/`en-US`; `/v1` conserva resposta anterior. GitHub Actions foi renomeado para destacar `21gateway-back` e `21gateway-front`; ambas as imagens continuam no GHCR. Nenhum SMS real ou deployment executado.
# Notificações internas — 30/10/2026

Novo trabalho após `v0.1.0`: aplicar migração 026 antes de publicar API/worker para sininho, histórico e preferências de e-mail do super administrador. E-mail fica desligado por padrão; confirmar SMTP e administradores ativos antes de habilitar qualquer tipo. Consulte [notifications.md](notifications.md). Não confundir com alertas independentes de disponibilidade da API.
