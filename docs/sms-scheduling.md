# Agendamento local de SMS

Autorizado pelo usuário em 15/10/2026: preservar scheduledDate do POST /v1/send e executar envio futuro no gateway, independentemente de existir endpoint de agendamento Zoug. Usuário reforçou alto volume e desempenho também nos demais endpoints.

## Contrato e comportamento

- Mesma rota, Basic Auth, whitelist e lista de até 200 itens. scheduledDate ausente/zero mantém envio imediato; futuro é Unix em segundos, convertido para UTC. Datas passadas, iguais ao segundo atual ou acima de 9999-12-31 são recusadas por item; não há envio imediato silencioso.
- Ativação explícita: SCHEDULER_ENABLED=true com chave de conteúdo válida. Sem ativação, mantém retorno anterior de agendamento não suportado. Não habilitado nesta sessão e nenhum SMS real enviado.
- OK/code5/id_sms no agendamento significa aceite durável na fila, não envio nem entrega. id_sms é criado no aceite e reaproveitado no index e resultado final; correlationId também é preservado. Formato do recibo usa o contrato existente de aceite; homologar o comportamento do legado para agendamento.
- Valida e normaliza texto antes de cifrar; a regra atual exige número brasileiro completo de onze dígitos para agendar. Nono dígito ainda sem contrato. Itens agendados inválidos são recusados localmente, diferentemente de parte da validação delegada nos lotes imediatos; confirmar esse comportamento no aceite legado.
- Pedidos mistos mantêm a ordem original. Reservam o orçamento uma vez; primeiro persistem itens futuros e só depois enviam os imediatos. Erro na fila impede chamada imediata; falha posterior dos imediatos vira NOK desses itens, sem ocultar os agendamentos já aceitos.
- Admissão de pedido só futuro reserva requisição por conta com zero mensagens de orçamento do fornecedor. Despacho usa o orçamento real de mensagens, compartilhado com envios imediatos. O limite de entrada é de uma requisição/segundo por conta. A reserva de admissão só futura não consome a reserva de despacho: isso evita que uma conta alimentando a fila continuamente impeça seu próprio worker de enviar. Imediatos e worker compartilham a reserva de despacho por conta e o teto agregado de mensagens.
- Ao consultar status, dispatched_at é usado como data do envio quando existir, preservando created_at do aceite. Não gerar DLR de entrega só porque uma mensagem entrou na fila ou foi aceita pelo fornecedor.

## Persistência e execução

Migração 010 separa o escopo de reservas API/despacho e cria scheduled_sms com FK única para sms_attempts; adiciona scheduled_at/dispatched_at às tentativas. Índices parciais por disponibilidade, conta, lease e estados terminais recentes. Texto fica cifrado com AES-GCM e AAD vinculando conta + chave aleatória do job; não entra em SMSAttempt, respostas, painel ou logs.

O domínio tem model/repository/service/worker próprios; controller SMS reutiliza a rota atual. Sem camadas vazias ou framework novo: reutiliza a infraestrutura PostgreSQL/GORM e padrão de outbox/worker já existente. Não é necessário cron por mensagem.

Worker consulta vencidos a cada segundo, quatro grupos concorrentes, no máximo 200 mensagens por grupo/conta. Seleção e gravações são em lote, sem uma goroutine ou consulta por mensagem. Usa SKIP LOCKED, lock transacional por conta e lease/token para impedir uso concorrente do mesmo job. Leituras são limitadas; não carrega toda a fila em memória.

Reutiliza SMSService, normalização, AccountSMSProvider, cliente HTTP pooled, interpretação de resultados e SendLimits. O adaptador de tentativas preserva IDs já existentes e registra a passagem de claimed para dispatching antes de chamar o fornecedor. Datas futuras diferentes que já venceram podem ser agrupadas para a mesma conta; são selecionadas em ordem de data/ID, sem garantia de ordem de entrega na operadora.

| Estado | Significado |
| --- | --- |
| queued | Aguardando horário/disponibilidade |
| claimed | Reservado pelo worker, ainda sem iniciar envio |
| dispatching | Intenção de envio comprometida antes da rede |
| done | Fornecedor aceitou; entrega continua dependente de DLR |
| failed | Rejeição conhecida ou conta bloqueada |
| unknown | Resultado incerto; não reenviar automaticamente |

Lease padrão 30 segundos. claimed expirado pode ser retomado, pois não cruzou a fronteira de envio. dispatching expirado vira unknown em lotes de até 200, sem retornar à fila. Isso é conservador: uma queda entre commit da intenção e a chamada pode deixar mensagem desconhecida mesmo sem envio; exige reconciliação, não retry cego. Não promete exactly-once.

UUID/outcome e término do job são gravados na mesma transação. Falha após resposta do fornecedor deixa dispatching para recuperação conservadora. Resultado de rede incerto não é reenviado. Repetir o POST de agendamento também não tem idempotência nova: correlationId não é chave única.

Antes da rede, falta de orçamento, erro temporário de banco/configuração/token ou indisponibilidade da chave adiam o grupo por um segundo. Não são reenvios ao fornecedor, porque Begin não ocorreu. Conta/empresa inativa ou arquivada impede envio. São usados token e configuração atuais da conta na execução, não uma cópia antiga do token.

Conteúdo é limpo ao concluir, rejeitar ou marcar unknown. Enquanto aguarda o horário ou configuração, permanece cifrado. A retenção MO de 90 dias não se aplica a esse conteúdo; metadados continuam com política separada. Uma chave ausente/ciphertext corrompido pode impedir o grupo da conta de avançar até intervenção; não há tela de recuperação/quarentena nessa entrega.

## Ativação operacional

1. Executar migração 010 em ambiente isolado e comprovar a suíte PostgreSQL antes do deployment. Binário novo lê colunas/tabela novas mesmo com scheduler desligado.
2. Configurar o processo único do backend com SCHEDULED_CONTENT_KEY privada (base64 de 32 bytes aleatórios), diferente das chaves de token/MO. Manter cópia recuperável no gestor de segredos/backup.
3. Configurar chaves de tokens do fornecedor e tokens por conta também no worker; Compose foi atualizado para isso.
4. Habilitar SCHEDULER_ENABLED nas duas aplicações e manter processo worker ativo. Não aceitar fila com worker ausente. Não foi criado recurso AWS ou agendador externo.
5. Homologar conta/destino controlados, recibo, horário UTC, DLR/MO e tratamento de falha. Não apontar teste de carga para fornecedor real.

Keyring opcional: SCHEDULED_ACTIVE_KEY_ID e SCHEDULED_ENCRYPTION_KEYS usam envelopes v2; manter leitores/chaves antigas até todos os jobs e backups correspondentes drenarem. CLI maintenance atual não recriptografa agendamentos. Não remover chave antiga enquanto houver conteúdo pendente ou backup dependente.

Desligar scheduler não cancela/apaga jobs; para voltar, reativar com as chaves e migrations compatíveis. Rollback para binário antigo não executa a fila e exige procedimento operacional; não excluir a tabela para reverter.

## Volume e observabilidade

GET /internal/operations protegido agora contém scheduler: queued, due, claimed, dispatching, oldestDueAt, unknownLastHour e failedLastHour. Não retorna conteúdos/destinos/segredos. Contagens não são snapshot transacional nem histórico de métricas. Acompanhar idade dos vencidos, crescimento, desconhecidos e worker disponível.

Resumo administrativo reduziu contagens de sete consultas para quatro, agregando estados em uma passagem por mensagens e outra por callbacks, com timeout de cinco segundos. Listagens continuam com paginação/teto existente; busca ILIKE e COUNT/OFFSET profundos ainda precisam de EXPLAIN e carga com histórico representativo. Não criar índices extensos/extensões por suposição.

O teto de 200 mensagens/s é compartilhado, não multiplicado por worker/réplica. Quantidade de workers não é garantia de TPS. Sem benchmark integrado, não há capacidade ou SLA certificados. Concorrência e SKIP LOCKED precisam ser provados com PostgreSQL real, especialmente sob backlog grande.

### Cenário de 100.000 SMS agendados

Uma credencial precisa enviar pelo menos 500 requisições de 200 itens. Com uma requisição/s por credencial, a admissão leva no mínimo 500 segundos. O teto compartilhado de 200 mensagens/s também fixa 500 segundos como limite inferior de despacho, mesmo com mais workers; HTTP do fornecedor, banco, cifra, concorrência com envios imediatos e falhas elevam a duração. O sistema persiste 100.000 tentativas e 100.000 jobs cifrados, selecionando no máximo 200 por lote, sem carregar toda a fila em memória. Não há endpoint de importação de campanha nem SLA para essa escala.

Antes de aceitar essa campanha em produção, executar carga integrada com 100.000 itens e múltiplas contas/horários, medir latência e tamanho do banco, espera no lock do limitador, consultas com `EXPLAIN (ANALYZE, BUFFERS)`, idade da fila, vazão real do fornecedor, recuperação após queda e tempo de conclusão. Definir quota de fila/backpressure e política de prazo/expiração com o usuário; não descartar nem reenviar `unknown` por suposição.

## Verificação e limites restantes

Testes locais: lote200 cifrado, IDs/correlação, pedido misto, datas inválidas, ausência de envio após falha da fila, reaproveitamento de IDs no worker, resultado desconhecido sem retry, adiamento por limite, bloqueio de conta, vínculo criptográfico e endpoint HTTP sem contato com fornecedor. Benchmark de admissão200 usa repositório em memória; não mede banco/rede/Argon2id/TPS real.

TestPostgresScheduledQueueLeasesAndUnknownRecovery prepara prova SQL de horário, lease, retomada, desconhecido, CAS e conclusão com ID preservado; ainda não executado sem TEST_DATABASE_URL/acesso Docker. Suítes anteriores continuam obrigatórias.

Pendências explícitas: prazo máximo de agendamento/TTL e atraso aceitável após indisponibilidade; quotas de armazenamento e backpressure; cancelamento/edição de agendamento; recuperação manual/quarentena; política de metadados/backups; carga integrada/EXPLAIN. Não há descarte por idade/TTL presumidos. Jobs já vencidos durante parada são elegíveis na retomada, respeitando limites.

Falhas locais conhecidas ficam nos metadados da tentativa/job. Não foi criado callback sintético para rejeição local de um agendamento; definir contrato desse aviso com a 21. DLR/MO reais do fornecedor continuam dependendo dos contratos pendentes e envio mantém callback_level NONE enquanto não homologados. Saldo e status/date permanecem pendentes. Esta implementação não declara substituição integral do legado pronta.

Validação desta execução: go test -race ./..., go vet ./... e go build ./... passaram. Frontend lint/build e 17 testes passaram; Compose válido. Banco real não exercitado, testes PostgreSQL explicitamente pulados por ausência de DSN. Benchmark ScheduledAdmission200 ~0,682ms/200 itens em memória; baseline em docs/reviews/performance-baseline.md, sem afirmar TPS integrado.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migrações até 013 obrigatórias nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. PostgreSQL local validado em 27/10/2026; SQS real/carga ainda pendentes. Testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.
