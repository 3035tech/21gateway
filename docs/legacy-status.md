# Consulta legada de status

## Implementado

GET /v1/status com Basic Auth e whitelist compartilhados com envio, registrado em OpenAPI. Controller, service, DTO, model e repository próprios do domínio de consulta. Até 100 mensagens com correlationId e estado DLR persistido, ordenadas por ID. Estados pending/accepted de envio não viram SENT ou DELIVERED artificialmente.

Consumo separado da outbox: migração 006 adiciona legacy_status_receipts e índice parcial em sms_attempts. Consulta e recibos são uma transação; advisory lock serializa consultas da mesma credencial. Consultas não alteram callback_jobs nem delivery_events. Versão é o par delivery_updated_at/status: uma atualização posterior fica disponível novamente. Se vários estados chegarem antes da consulta, retorna o mais recente. Histórico prévio com DLR fica elegível na primeira consulta.

Dados de outra credencial nunca entram no filtro. Contas bloqueadas/arquivadas e IPs proibidos são rejeitados pelo guard comum. Falha de banco retorna indisponibilidade, sem mensagem SQL pública. Não há chamada ao fornecedor nem token exigido para consultar dados locais.

Formato documentado: status OK, code 5, message REQUIRED_OK e sent_status com correlationId, destination, sentDate, unixSentDate e status. Datas de envio usam created_at da tentativa; apresentação America/Sao_Paulo com tzdata embutido e Unix em string. Lista vazia retorna OK/code 6 e array vazio.

## Limites de compatibilidade ainda a homologar

- O manual não fornece o texto da mensagem para code 6: NO_RECORDS é provisório, não contrato confirmado.
- Semântica adotada: consumir antes de escrever resposta HTTP; desconexão do cliente após commit não desfaz consumo. Confirmar contra legado. Consumo não garante que o cliente recebeu a resposta.
- Datas usam criação da tentativa, não horário de aceite do fornecedor. Destino usa o número armazenado atualmente, com prefixo 55 nos números válidos; forma original do legado ainda requer evidência.
- Somente DLR recebido é exposto; status do white label continua sem payload/autenticação homologados. Ingresso interno normalizado permanece opt-in e desativado por padrão. Não afirmar que estados reais já chegam.
- /v1/status/date não registrado nesta etapa. Manual páginas 6–7 exige date Unix, cita correlationid em minúsculas na tabela (correlationId na prosa), destination e link de continuação, mas não documenta nome/formato do link nem parâmetros de paginação. Não inventar esses detalhes.

## Migração e validação

Aplicar migração explícita antes de atualizar API. Mudança aditiva, sem alterar metadados ou callbacks existentes. Retorno ao binário anterior pode manter tabela/índice; reativar consulta preserva recibos. Apagar recibos reexpõe estados, portanto não fazer rollback destrutivo automático.

Teste HTTP cobre autenticação, whitelist com X-Forwarded-For falso, formato/datas, retorno vazio e indisponibilidade. Teste PostgreSQL opcional ampliado para 100+1, consumo repetido, isolamento e atualização posterior; exige TEST_DATABASE_URL. Sem banco disponível não representa prova de transações/advisory locks executados.
