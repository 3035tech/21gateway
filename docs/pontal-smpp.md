# Pontal SMS via SMPP — evidência e adaptação segura

O projeto Node `smpp-pontal`, fornecido pelo usuário como referência em 01/11/2026, foi lido sem alterações. Ele comprova uma integração existente por SMPP, mas não substitui a documentação formal ou uma homologação com a Pontal. Nenhum segredo ou payload real foi copiado para este repositório.

## O que o código existente mostra

- `README.md` define duas contas de conexão, `TOKEN` e `MASSIVE`, cada uma com endpoint, `system_id`, senha e `id_gw`. O significado comercial e os limites das duas contas não estão documentados.
- `sessionsmpp.service.ts` cria uma sessão e faz `bind_transceiver`; envia `destination_addr` e `short_message` com `submit_sm`. O `submit_sm_resp` fornece `message_id`.
- `deliver_sm` com `receipted_message_id` é tratado como recibo. O código procura `stat:DELIVRD` no texto; sem esse trecho marca qualquer outro caso como `NOT_DELIVERED`. Sem esse TLV, trata o PDU como MO usando `source_addr` e o texto.
- `task.service.ts` busca até 200 SMS por vez em tabelas legadas. Esse lote é uma decisão do serviço Node, não uma capacidade ou limite confirmado da Pontal.

## Problemas que não devem ser copiados

O código Node remove o SMS da fila antes de confirmar o `submit_sm_resp`; falha ou desconexão nesse intervalo pode perder o envio. Recibos são reduzidos a dois estados sem confirmar o vocabulário da Pontal. O MO é associado ao último MT do mesmo telefone, o que pode vinculá-lo à empresa errada. Há SQL montado por interpolação e logs com PDU, telefone e mensagem. A resposta `deliver_sm_resp` aparece somente no ramo MO. O novo gateway deve preservar suas regras de resultado incerto, persistência durável, isolamento de contas e logs sem dados pessoais.

## Implementado no gateway

`internal/integrations/pontal/pdu.go` usa `linxGnu/gosmpp` v0.3.1 para montar o `submit_sm` mínimo observado (destino e texto), traduzir `submit_sm_resp` e classificar `deliver_sm` sem iniciar sessão. O pedido de recibo fica no próprio PDU, quando solicitado pelo chamador. A montagem recusa mensagens acima de 140 octetos até haver política de concatenação e IDs por segmento. Source address e TON/NPI seguem os defaults da biblioteca, não uma configuração Pontal aprovada. O ID do SMSC só é aceito após resposta de envio bem-sucedida. Um recibo sem ID não vira MO; seu estado fica bruto (`stat` ou TLV), sem equivalência inventada com os estados da API 21. Uma MO mantém origem e texto em memória, sem associação por heurística. Testes sintéticos cobrem montagem, aceite/rejeição, recibo e MO. Não existe bind, envio, ack, consumo de fila, callback ou rota Pontal em runtime.

`submission.go` classifica somente o `submit_sm_resp`: sucesso com ID válido é `accepted`; resposta explícita de erro é `rejected` e conserva o código SMPP; resposta ausente, inválida ou sucesso sem ID é `unknown`. Essa política impede que uma desconexão ou um timeout sejam interpretados como recusa e reenviados. O código ainda não controla uma sessão nem persiste essa decisão.

## Desenho de execução

O processo contínuo `/worker` é o lugar para manter as sessões SMPP e consumir os IDs de envio já persistidos pelo gateway. Uma sessão deve ser compartilhada por conta SMPP, com janela e vazão configuradas conforme a Pontal; abrir conexão por SMS não escala. O domínio de SMS continua dono do aceite durável, dos estados `dispatching`/`unknown` e da proibição de repetir automaticamente resultado incerto. O transporte Pontal recebe apenas o pedido necessário para formar PDUs e devolve ID/resultado; não acessa tabelas legadas nem decide o contrato `/v1/send`.

O receptor `deliver_sm` deve gravar recibos e MO de modo durável antes de responder à Pontal, e o processamento de entrega ao cliente ocorre depois, nos domínios DLR/MO existentes. Na biblioteca escolhida, isso exige o callback com resposta manual (`OnAllPDU`), não o callback que pode confirmar automaticamente (`OnPDU`). Identidade de conta, deduplicação, resposta do PDU e mapeamento de estado dependem do contrato Pontal. Para escalar horizontalmente, um único dono de bind por credencial deve ser garantido por lease distribuído ou particionamento explícito; aumentar réplicas sem coordenação criaria sessões duplicadas. Essa fronteira de processo permite mover o worker SMPP para um serviço separado sem mudar os endpoints dos clientes; outbox/IDs e interfaces de domínio permanecem no gateway. Não ativar o worker até provar failover, backpressure, limites, persistência e ausência de reenvio incerto em simulador e HML.

## Falhas, desconexão e limites da biblioteca

- `gosmpp.Submit` apenas enfileira o PDU para escrita interna; `nil` não comprova escrita nem aceite. A tentativa precisa estar persistida antes de chamar `Submit`, e deve permanecer em resultado incerto até um `submit_sm_resp` correlacionado pela sequência e pela geração do bind. Um retorno de erro, timeout, `OnSubmitError`, expiração ou `OnClosed` sem resposta não autoriza reenvio automático.
- `NewSession` pode fazer rebind automático após uma sessão estabelecida, mas uma falha no primeiro `Connect` retorna erro ao chamador e requer retry externo com backoff limitado e jitter. Enquanto desconectado, o worker deve suspender novas submissões e marcar as tentativas em voo sem resposta como incertas. Após rebind, deve recuperar apenas trabalho que comprovadamente não chegou ao transporte; tentativas em voo exigem conciliação manual ou mecanismo confirmado pela Pontal. `OnRebind` não transforma uma submissão antiga em sucesso.
- O callback `OnAllPDU` é ignorado quando `WindowedRequestTracking` está configurado com tratamento de respostas. Com janela, usar `OnReceivedPduRequest` para `deliver_sm` e `OnExpectedPduResponse` para `submit_sm_resp`; configurar também expiração, fechamento e respostas inesperadas. Sem janela, `OnAllPDU` recebe tudo e exige correlação própria. Em ambos os modos, a resposta ao `deliver_sm` só pode ocorrer após persistência durável e com a mesma sequência do pedido. Se a persistência falhar, não confirmar o PDU: fechar o bind e registrar a falha segura para permitir a política de reentrega do fornecedor, a confirmar em HML.
- A própria biblioteca faz auto resposta de alguns PDUs em modos específicos. Validar `enquire_link`, `unbind`, erros de escrita e callbacks de fechamento num simulador antes de escolher o modo de sessão. Limitar fila em memória, janela e vazão por conta com valores homologados; não copiar os 200 itens do cron Node como limite SMPP.
- Evitar logar erro original da biblioteca sem filtragem: endereço, credencial, PDU ou conteúdo podem aparecer em diagnósticos. Logs e métricas devem usar conta interna, geração do bind, estado, código SMPP e contadores, sem telefone ou texto.

## Para avançar

Obter o contrato SMPP da Pontal ou exemplos de PDUs expurgados de dados pessoais: versão, host/porta/TLS e política de IP, binds e limites por conta, TON/NPI/source address, encoding/concatenação, `registered_delivery`, semântica do `submit_sm_resp`, formato e todos os estados de DLR, origem/identificação de MO, `deliver_sm_resp`, reconexão, janela/TPS e ambiente de homologação. Depois, implementar o transporte com persistência antes de reconhecer recibos e sem reenviar automaticamente um `submit_sm` de resultado incerto. Não usar credenciais legadas em testes sem autorização específica.
