# Despacho agendado com SQS

Implementado em 15/10/2026, opt-in e sem deployment. Decisão autorizada: scheduler Go identifica mensagens vencidas no PostgreSQL, publica referências duráveis no SQS e consumidores Go encaminham ao white label. Envios imediatos continuam síncronos.

## Fluxo e concorrência

1. O aceite grava tentativa, ID estável e conteúdo cifrado no PostgreSQL; não depende de SQS disponível.
2. Scheduler cria lote/outbox e associa mensagens na mesma transação. Há no máximo um lote aberto por credencial, até 200 mensagens. Contas atendidas há mais tempo têm prioridade; tamanho é dividido entre contas com trabalho vencido. Isso mitiga monopolização, sem garantir SLA de equidade.
3. Publicador envia apenas `{"version":1,"batchId":"UUID"}` com SDK oficial AWS v2. Sem texto, telefone, token ou correlationId na fila. Falha/reinício repete a referência, preservando o lote no banco.
4. Cada réplica executa quatro consumidores e quatro publicadores limitados, sem goroutine por SMS. ReceiveMessage lê uma referência por vez, long polling de 20s, visibilidade de 90s e processamento limitado a 20s. Publicação tem timeout de 10s.
5. Consumidor disputa lote/lease no PostgreSQL. Duplicatas ocupadas aguardam; finalizadas apenas recebem ACK. Os limites de despacho são compartilhados entre réplicas e com envios imediatos. Reserva parcial usa a capacidade disponível e libera a cauda ainda não enviada.
6. Estado dispatching é persistido antes da chamada externa. Se o resultado ficar incerto, torna-se unknown, sem reenvio automático. Conteúdo terminal é limpo. Antes de confirmar uma entrega com trabalho restante, publica-se outra referência para o mesmo lote; falha nessa publicação preserva a referência atual.

Locks de seleção usam SKIP LOCKED; transações terminam antes do HTTP. Leases/CAS protegem retomada e atualização. Configuração inválida pode atrasar o próprio lote, sem manter lock durante espera externa. Não há promessa de exactly-once no fornecedor nem de TPS sem carga integrada.

A outbox republica referências abertas a cada cinco minutos, recuperando referências perdidas/expiradas. Isso também pode reativar uma referência que foi para DLQ: a DLQ registra falhas, não é quarentena permanente. Payload inválido aguarda visibilidade normal; outras indisponibilidades usam pausas limitadas. Operação deve observar idade da outbox, atraso e DLQ, além de estados unknown.

## Ativação e infraestrutura

- Aplicar migrações até **013** antes dos novos binários, mesmo com SQS desabilitado; 011 expande o despacho, 012 corrige a reserva de requisição sem SMS e 013 adiciona índices de listagem.
- Configurar SCHEDULER_ENABLED=true no backend e as chaves independentes de conteúdo agendado descritas em [sms-scheduling.md](sms-scheduling.md).
- Configurar SCHEDULED_SQS_ENABLED=true, AWS_REGION e SCHEDULED_SQS_QUEUE_URL no backend. URL deve ser HTTPS oficial de fila Standard na região indicada; FIFO/endpoints customizados não suportados nesta etapa. Padrão é desligado.
- O processo único usa a cadeia de credenciais do SDK, preferencialmente IAM do pod, sem chaves no repositório/frontend. A aplicação precisa alcançar PostgreSQL, SQS e fornecedor; dimensionar pool de banco, conexões e réplicas com carga medida.
- Template revisável: ops/aws/scheduled-sqs.cloudformation.json. Contém fila Standard cifrada SSE-SQS, retenção de 4 dias, DLQ de 14 dias, maxReceiveCount de 100 configurável, política TLS, managed policy de quatro ações apenas na fila principal e alarme DLQ. Recursos possuem Retain. Policy não está anexada a role e alarme não tem destino de notificação. Template preparado, não validado pelo CloudFormation nem implantado.

Não alternar transportes com lotes ativos. Para voltar ao worker local: suspender novo aceite agendado, drenar lotes SQS, confirmar ausência de lotes abertos e mensagens claimed/dispatching, parar todos os workers e então alterar configuração. Worker local ignora mensagens vinculadas a lotes SQS; desligar apenas a flag pode deixá-las pendentes. Não destacar mensagens, apagar outbox ou repetir unknown automaticamente. Evitar versões antigas/mistos de transporte durante implantação.

## Evidências e pendências

Estimativa informada pela 21 em 27/10/2026: até **1 milhão de SMS por mês** enviados pelos usuários da API. É teto de planejamento, não medição, quota, SLA ou garantia de distribuição temporal. Nem todos esses envios passam pelo SQS: no desenho atual, somente os agendados usam referências de lote na fila; os imediatos continuam síncronos. Um milhão de agendados formariam pelo menos 5.000 referências se todos os lotes tivessem 200 itens, mas mais referências quando horários, contas ou capacidade compartilhada impedirem lotes cheios. A projeção de custos deve incluir `SendMessage`, `ReceiveMessage` (inclusive vazio), `DeleteMessage`, republicações e DLQ. Medir picos, tamanho do PostgreSQL/índices, uso de conexões, idade da fila e vazão do fornecedor antes de dimensionar produção. A estimativa não autoriza mudar limites ou retenção.

Testes sem rede cobrem referências concorrentes, uma chamada ao fornecedor, duplicatas, ACK perdido, persistência incerta, reserva parcial, continuação antes do ACK, publicação/retry, payload inválido e cancelamento com consumidores limitados. SDK é exercitado com HTTP simulado. Go race/vet/build e frontend lint/build/17 testes passaram.

Suíte PostgreSQL executada em 27/10/2026 com `go test -race ./...`, incluindo transações, SKIP LOCKED, outbox/leases, distribuição entre contas e orçamento parcial. Antes da produção: executar EXPLAIN e carga com múltiplas réplicas; validar SQS/DLQ/IAM em ambiente isolado e falhas/reinícios; configurar alertas. Nenhuma fila/recurso, SMS real ou deployment foi executado.

Continuam pendentes quota de agendamentos, TTL/cancelamento/edição, retenção de metadados, dimensionamento e contratos DLR/MO externos. Retenção MO de 90 dias não define retenção destes metadados.

## Referências AWS

- [Entrega pelo menos uma vez em Standard](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/standard-queues-at-least-once-delivery.html): exige consumidor idempotente.
- [Long polling](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-short-and-long-polling.html).
- [Retries e timeouts SDK Go v2](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html): retries somente para referências SQS, nunca repetição automática de envio SMS.
- [Fila CloudFormation](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-sqs-queue.html) e [DLQ](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html).

## Revisão de robustez — 15/10/2026

O publicador prioriza referências já persistidas e disponíveis na outbox antes de preparar novos lotes. Uma falha na preparação não bloqueia a publicação de lotes prontos. Teste de regressão reproduziu o bloqueio anterior e cobre o despacho da referência sem chamar o fornecedor. Esta revisão não altera os limites de concorrência, contrato V1 ou regras de resultado incerto.
