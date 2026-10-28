# Fila por ID de envio

O modo `SMS_SQS_ENABLED=true` aceita **todos** os SMS válidos na mesma transação PostgreSQL que cria a tentativa e o registro de despacho. Cada tentativa recebe um `id_sms` estável. Um publicador envia à SQS Standard somente `{"version":2,"sendId":"<id_sms>"}`; não publica número, texto, token ou `correlationId`. O texto fica cifrado no banco e é removido do registro de despacho no estado terminal.

Modelagem: `sms_attempts.id` é a identidade do envio; `scheduled_sms.attempt_id` tem FK e unicidade, portanto existe no máximo um job por tentativa. `credential_id` e `company_id` preservam o vínculo de conta. A migração 022 acrescenta `send_reference` como domínio booleano para separar o transporte novo dos lotes antigos e campos de publicação para retomada, sem alterar IDs ou registros existentes. Não há payload livre em JSONB na persistência.

O endpoint `/v1/send` retorna um resultado por item na ordem recebida. `OK`/código 5 confirma **aceite durável para processamento**, não aceite do gateway nem entrega ao aparelho. `id_sms` identifica a tentativa no painel e no suporte. A consulta legada `/v1/status` continua trazendo apenas eventos DLR e não informa uma rejeição posterior do gateway; um contrato de consulta de submissão para clientes deve ser definido antes do corte público. A tela de teste mostra estado `pending` enquanto o worker não conclui o envio. SMS futuro só pode ser publicado quando `scheduledDate` vencer.

O publicador reclama um registro com `FOR UPDATE SKIP LOCKED`, marca a próxima tentativa de publicação e libera a transação antes de chamar a SQS. Falha ou perda da resposta permite republicação da mesma referência. O consumidor reclama o envio por ID com lease no PostgreSQL; uma cópia duplicada não cruza novamente o limite de chamada externa. `dispatching` é gravado antes da chamada ao gateway. Se o processo perder o resultado, o estado passa a `unknown`, sem reenvio automático. Há nova publicação apenas para falhas comprovadamente anteriores à chamada externa. Falhas repetidas de payload ou banco devem ser observadas na DLQ; o atraso da outbox e os estados `unknown` exigem alerta.

## Ativação

1. Aplicar a migração **022**, que só acrescenta colunas e índice a `scheduled_sms`, antes de ativar qualquer binário novo. Os modos antigos continuam lendo as linhas existentes. Preservar chaves de cifra antigas para mensagens pendentes.
2. Drenar os lotes antigos (`sms_dispatch_batches.completed_at IS NULL` deve contar zero), pausar os workers antigos e aguardar a saída dos pods antigos. A inicialização do modo novo falha se houver lote aberto. O formato v2 usa **outra fila**; nunca apontar `SMS_SQS_QUEUE_URL` para a fila v1 de lotes.
3. Criar fila Standard com DLQ, TLS, criptografia e permissões IAM de envio/recebimento/exclusão/visibilidade/atributos. O template em `ops/aws/scheduled-sqs.cloudformation.json` serve como base, com nomes e role próprios para a nova fila.
4. Implantar a imagem com `/worker` como processo separado, mesma configuração de banco/cifras/token da API e IAM da fila. Configurar na API e no worker `SCHEDULER_ENABLED=true`, `SMS_SQS_ENABLED=true`, `SMS_SQS_QUEUE_URL`, `AWS_REGION`, `SCHEDULED_CONTENT_KEY` ou keyring equivalente e `WORKERS_IN_API=false`. Manter `SCHEDULED_SQS_ENABLED=false`.
5. Confirmar worker ativo, publicação, consumo, duplicata, resultado incerto, DLQ, limite por credencial e consulta de status em ambiente isolado antes de abrir o tráfego de clientes. Um aceite sem worker apenas acumula envios no banco.

## Retorno

Parar novos aceites e drenar a fila v2 e todos os registros `queued`, `claimed` e `dispatching` antes de voltar ao envio síncrono ou ao lote v1. Não desativar a flag nem apagar a fila enquanto houver registros pendentes. Conservar IDs, histórico e estados `unknown`; não reenviar resultados incertos. Uma versão antiga do worker local pode reclamar registros sem lote, então não rodar versões antigas e novas juntas durante o corte.

O código implementa o modo e o processo separado, mas a fila v2, o IAM e o deployment **não foram provisionados**. Até ativação controlada, o envio imediato continua com a configuração anterior.
