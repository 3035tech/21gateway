> A etapa unitária abaixo é referência histórica. O runtime atual também suporta lote conforme docs/sms-batch.md e usa token por conta conforme docs/account-settings.md. Normalização/fornecedor ainda exigem homologação.

# Envio simples unitário

O envio síncrono descrito neste documento continua como modo legado. A implementação opt-in de aceite durável para todos os envios e fila SQS por `id_sms` está em [sms-send-queue.md](sms-send-queue.md); nesse modo a resposta não confirma aceite do fornecedor.

## Escopo desta etapa

Uma mensagem imediata por requisição, preservando POST /v1/send e a lista sms do legado. O fornecedor recebe POST /api/send. Unitário significa um destinatário por chamada; não garante um único segmento físico ou uma única cobrança por SMS. Não há lotes, agendamento, saldo, MO ou consulta legada de status. O módulo interno de callbacks/outbox está em docs/backoffice.md; o envio não solicita DLR ao white label até homologar seu contrato.

## Entrada e tradução

Basic Auth com credencial técnica ativa. Credenciais são criadas via CLI e armazenadas com hash Argon2id. Não há senha padrão. Username até 120 bytes; novas senhas de 12 a 128 bytes. Credenciais antigas ainda não foram importadas.

O envelope sms deve conter exatamente um item. messageText não pode ser vazio/espaços e aceita até 160 caracteres Unicode. destination exige exatamente 11 dígitos (DDD+número), conforme manual legado; o adaptador acrescenta 55. correlationId é opcional e limitado a 50 caracteres. scheduledDate ausente/nulo/zero representa envio imediato; qualquer valor diferente de zero é rejeitado explicitamente.

messageText → message; destination → destination_addr com 55; correlationId → reference_id; ID interno → index. type é SMS. callback_level é NONE. Não é enviado callback_url nesta etapa.

Encoding é selecionado por biblioteca GSM7: 0 para texto representável no alfabeto GSM7, 8 para outros caracteres. O texto não é transliterado, truncado nem alterado. Acentos/emojis podem produzir múltiplos segmentos e custos conforme o fornecedor; confirmar regras de segmentação na homologação. O formato Latin1 ambíguo do manual não é utilizado.

Token e endpoint ficam no backend via WHITELABEL_API_TOKEN e WHITELABEL_BASE_URL. Base de produção exige HTTPS e não permite credenciais/query/fragment no URL; redirects não são seguidos. Não desabilitar validação TLS. Sem token, retornar indisponibilidade sem chamar o fornecedor.

## Resultado

HTTP 200 com lista de resultados. Aceite confirmado: status OK, code 5, id_sms numérico em string e correlationId quando informado. O aceite significa que o fornecedor respondeu com message_id UUID válido e status sent; não prova entrega ao aparelho.

Falhas por item: status NOK, code -5 e message centralizada. Limite de envio: code 429. Número inválido usa INVALID_NUMBER_FORMAT; mensagem inválida INVALID_MESSAGE_FORMAT. Lotes/agendamento retornam BATCH_NOT_SUPPORTED e SCHEDULING_NOT_SUPPORTED.

Erros de saldo/autenticação/TPS/rejeição do fornecedor são traduzidos sem repassar seus corpos ou segredos. Os novos códigos textuais não presentes no manual são decisões provisórias desta etapa e devem ser conferidos com a implementação real do legado antes de migrar clientes.

Falhas de autenticação da chamada, JSON inválido e indisponibilidade de infraestrutura usam Problem Details com code estável. JSON inválido retorna HTTP 422 por padrão Huma. O manual recebido não determina esses envelopes; validá-los contra o PHP/tráfego real. Não declarar compatibilidade integral com a API antiga.

## Persistência e falhas

Antes do envio, criar SMSAttempt com credencial, empresa de origem quando vinculada, destino normalizado, correlationId, encoding, ID numérico e estado pending. Nenhum texto ou senha é armazenado nessa tabela.

- accepted: fornecedor confirmou aceite; UUID associado ao ID interno.
- rejected: rejeição conhecida.
- unknown: falha de rede, resposta inválida, 5xx ou outro resultado que não permita afirmar se houve aceite.
- pending: tentativa criada e ainda sem resultado durável; pode permanecer assim após interrupção do processo ou falha de gravação.

Após o envio, tentar persistir resultado mesmo se o cliente desconectar, com timeout próprio. Se o fornecedor já confirmou aceite e a atualização local falhar, retornar OK para não induzir reenvio; emitir log seguro com ID interno/estado/UUID. A reconciliação desse caso ainda é operacional e não possui worker automático.

Resultado desconhecido retorna SEND_OUTCOME_UNKNOWN com id_sms da tentativa. Nunca repetir automaticamente. O cliente também deve evitar retry cego; esta etapa não fornece idempotência nem elimina duplicações em novas chamadas. Não tratar correlationId como chave única.

Limitador x/time/rate: uma chamada de envio por segundo por credencial, burst 1, local ao processo. Não há limite distribuído. A verificação de senha tem concorrência limitada, sem cache. Dimensionamento de alto volume e regras especiais de clientes requerem medição e confirmação.

## Preparação e testes

Aplicar migrações via cmd/admin -action migrate, com Gormigrate, transação e lock PostgreSQL; criar credencial via create-credential e CLIENT_PASSWORD do ambiente. API startup não altera schema.

Testes cobrem tradução e ID, autenticação/conta desativada, validação, rate limit por credencial, falhas antes/depois do envio, rejeições e resultados incertos, redirects e encoding. HTTPmock evita acesso externo; repositórios de teste são em memória. Não há teste real de PostgreSQL ou fornecedor nesta etapa.

## Próximas confirmações

Token/rota e sandbox do fornecedor; número de teste autorizado; semântica exata de erros do PHP; política de encoding/segmentação; limites especiais por cliente; futuro vínculo por empresa e credenciais de fornecedor. Callback e idempotência continuam dependentes da documentação faltante.

## Esclarecimento sobre armazenamento

O usuário perguntou se precisamos salvar todas as mensagens. A recomendação apresentada é persistir o registro mínimo de cada tentativa encaminhada ao fornecedor, sem o texto do SMS: credencial/cliente, ID interno, UUID do fornecedor, correlationId, datas e resultado. Destinatário é útil para consultas por número do legado. A implementação atual armazena esses metadados, sem mensagem completa.

Persistir conteúdo só se houver necessidade confirmada de histórico completo, reenvio ou exigência específica. Prazo de retenção e limpeza dos metadados continuam em aberto; não assumir retenção permanente nem apagar registros necessários a callbacks/reconciliação sem política definida.
