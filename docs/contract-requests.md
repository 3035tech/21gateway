# Solicitações de contrato técnico

Textos prontos para encaminhamento. Não enviados a terceiros nesta tarefa.

## Para 21 / 3035: paginação legada

Estamos preservando /v1/status/date no novo gateway. O manual menciona blocos de 100 e um link de continuação, mas não informa seu formato; os exemplos PHP/Java oficiais não tratam a próxima página.

Podem fornecer uma chamada/resposta sanitizada de um dia com mais de 100 mensagens, incluindo primeira página, chamada da continuação, última página e consulta vazia? Precisamos manter JSON, nomes/tipos, query e headers exatamente como hoje, removendo senhas/tokens/cookies e substituindo dados dos clientes.

Confirmar também grafia correlationid/correlationId, destination, forma com barra final, ordenação, consumo ou não dos registros e fuso/data usados no filtro. Se possível, fornecer o controller PHP correspondente para comparação.

## Para Zoug / Devyx: saldo e agendamento

Estamos integrando um gateway que precisa preservar consulta de créditos e scheduledDate da API atual. O guia recebido, revisado em 10/02/2026, documenta envio individual/lote e erro de saldo insuficiente, mas não os contratos dessas duas funcionalidades.

Podem disponibilizar documentação OpenAPI/Postman da versão contratada e sandbox, com exemplos sem segredos reais?

Para **saldo**, precisamos de método/endpoint, autenticação, JSON de sucesso/erro, unidade/precisão e saldo disponível versus reservado. Confirmar o vínculo do token com conta/rota e a relação Master/Sub, inclusive zero, pós-pago ou ilimitado.

Para **agendamento**, confirmar suporte via API e os campos/formato de data/fuso em single e batch. Precisamos dos limites, resposta de aceite/IDs, consulta/cancelamento, reserva/cobrança de saldo, TPS e callbacks; incluir comportamento de timeout e idempotência. Se não suportado, confirmar explicitamente para definirmos a alternativa no gateway.

Um exemplo completo de cada fluxo nos permite fechar a tradução sem alterar as integrações dos clientes.
