# Callback SMS Zoug/Devyx

O documento recebido `ZOUG_SMS_API_CALLBACK.docx`, atualizado em 11/02/2026, define HTTP POST JSON com `callback_type` 1 (submissão), 2 (entrega) e 3 (MO). O adaptador em `internal/integrations/whitelabel/callback.go` valida os três formatos e traduz somente os resultados conhecidos dos tipos 1 e 2 para o contrato interno DLR. `CallbackService.AcceptWhiteLabelDelivery` reutiliza o aceite transacional existente, mas nenhuma rota externa da Zoug foi registrada ou ativada.

Em 28/10/2026, a Zoug confirmou que não assina callbacks nem permite adicionar assinatura. Os IPs de origem informados foram `18.232.239.74`, `3.213.146.37` e `34.195.124.142`; um segredo na URL cadastrada funciona. Há **uma URL global para todas as contas/tokens**, configurada no painel em `/me`. A URL enviada em cada requisição deveria prevalecer, mas a Zoug confirmou que atualmente é ignorada por um bug; aguardar aviso da correção. `SUBMIT_ONLY` envia apenas tipo 1, `DELIVERY_ONLY` envia tipos 2–5 e `ALL` envia todos. Tipos 4 e 5 são eventos de RCS, fora do escopo SMS.

O `message_id` corresponde ao ID retornado no envio e também vincula DLR e MO ao MT. A Zoug tenta entregar cada callback até três vezes, considera qualquer resposta 2xx sucesso e descarta após a terceira falha. Portanto, o ingresso externo deve autenticar a URL/infra de origem e persistir de forma idempotente **antes** de responder 2xx. A rota externa segue desabilitada enquanto essa proteção e o vínculo MO não forem implementados.

| Evento/código Zoug | Estado interno | Decisão |
| --- | --- | --- |
| Tipo 1 `ESME_ROK` | `SENT` | Submissão bem-sucedida, intermediária |
| Tipo 1 `UNKNOWN_ERROR` | `NOT_DELIVERED` | Exemplo real descrito pela Zoug como submissão rejeitada |
| Tipo 2 `DELIVRD` | `DELIVERED` | Entrega confirmada |
| Tipo 2 `UNDELIV`, `EXPIRED`, `REJECTD` | `NOT_DELIVERED` | Falha definitiva descrita no manual |
| Tipo 2 `DELETED` | `MESSAGE_CANCELLED` | Mensagem removida antes da entrega |
| Tipo 2 `ACCEPTD`, `ENROUTE` | `SENT` | Aceite/trânsito sem confirmação de entrega |
| Tipo 2 `UNKNOWN`, outros status ou códigos de submissão | Sem tradução | Não inventar resultado final |
| Tipo 3 MO | Sem tradução | Falta vínculo confirmado com conta e SMS original |

O parser exige `send_type=SMS`, UUID válido em `message_id`, data UTC e campos essenciais de cada tipo. `reference_id` e `source_addr` podem faltar: os exemplos reais de submissão e MO não os trazem. Campos JSON adicionais são tolerados. Os exemplos do manual mostram um espaço antes do UUID; o adaptador remove espaços nas extremidades do `message_id`. O identificador interno de evento é um SHA-256 de tipo, UUID, status, instante e campos relevantes, estável para repetições equivalentes do mesmo payload. Não é um identificador fornecido pela Zoug. `reference_id` não é usado para escolher a conta: pode se repetir entre clientes. O exemplo DLR traz `DELIVRD` com `err=120`; não inferir falha apenas porque `err` difere de `000`. `ACCEPTD` e `ENROUTE` são intermediários. Outros códigos de submissão/reporte seguem sem tradução até confirmar o significado com o fornecedor final.

## Pendências para ativar o ingresso externo

- Implementar autenticação por segredo na URL, restringir as três origens informadas com tratamento correto de proxy e persistir o evento antes de retornar 2xx. Cuidar da redação de logs para não expor o segredo embutido na URL.
- Configurar uma URL global em `/me`; aguardar correção da precedência de `callback_url` por envio na Zoug antes de depender dela.
- Vincular MO por `message_id` à tentativa original, mesmo sem `source_addr`, e validar com o fornecedor final se o telefone do MT é suficiente para identificar o remetente do MO.
- Confirmar a lista completa de `submit_status_code`, `report_status` e `err` com o fornecedor final, além de ordenação e eventuais correções de estados finais.
- Exemplos de homologação com headers e corpos reais sanitizados, incluindo rejeição, não entrega e MO.

`CALLBACK_INGRESS_ENABLED` e `MO_INGRESS_ENABLED` permanecem desligados em Dublin HML. O envio ao fornecedor continua com `callback_level=NONE`; o teste real de token por empresa não solicitou callback. Esta atualização do parser não adiciona migration, rota nem configuração de ingresso externo. A suíte de testes cobre os três formatos, os exemplos recebidos, repetição equivalente e campos inválidos.
