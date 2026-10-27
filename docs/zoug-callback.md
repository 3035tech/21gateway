# Callback SMS Zoug/Devyx

O documento recebido `ZOUG_SMS_API_CALLBACK.docx`, atualizado em 11/02/2026, define HTTP POST JSON com `callback_type` 1 (submissão), 2 (entrega) e 3 (MO). O adaptador em `internal/integrations/whitelabel/callback.go` valida os três formatos e traduz somente os resultados conhecidos dos tipos 1 e 2 para o contrato interno DLR. `CallbackService.AcceptWhiteLabelDelivery` reutiliza o aceite transacional existente, mas nenhuma rota externa da Zoug foi registrada ou ativada.

| Evento/código Zoug | Estado interno | Decisão |
| --- | --- | --- |
| Tipo 1 `ESME_ROK` | `SENT` | Submissão bem-sucedida, intermediária |
| Tipo 2 `DELIVRD` | `DELIVERED` | Entrega confirmada |
| Tipo 2 `UNDELIV`, `EXPIRED`, `REJECTD` | `NOT_DELIVERED` | Falha definitiva descrita no manual |
| Tipo 2 `DELETED` | `MESSAGE_CANCELLED` | Mensagem removida antes da entrega |
| Tipo 2 `ACCEPTD`, `ENROUTE` | `SENT` | Aceite/trânsito sem confirmação de entrega |
| Tipo 2 `UNKNOWN`, outros status ou códigos de submissão | Sem tradução | Não inventar resultado final |
| Tipo 3 MO | Sem tradução | Falta vínculo confirmado com conta e SMS original |

O parser exige `send_type=SMS`, UUID válido em `message_id`, presença de `reference_id`, data UTC e campos obrigatórios de cada tipo. Campos JSON adicionais são tolerados. Os exemplos do manual mostram um espaço antes do UUID; o adaptador remove espaços nas extremidades do `message_id`. O identificador interno de evento é um SHA-256 de tipo, UUID, status, instante e campos relevantes, estável para repetições equivalentes do mesmo payload. Não é um identificador fornecido pela Zoug nem substitui a confirmação da semântica de retries. `reference_id` não é usado para escolher a conta: pode se repetir entre clientes.

## Pendências para ativar o ingresso externo

- Autenticação, assinatura ou origem verificável dos POSTs; o manual não especifica nenhum desses mecanismos. Não criar endpoint público que persista eventos sem essa definição.
- Escopo da URL configurada em `/me` e precedência da URL por envio; valores exatos de `callback_level` para os três eventos, inclusive lote.
- Garantia de que o `message_id` dos tipos 1 e 2 corresponde ao UUID retornado no envio; significado do `message_id` e `reference_id` do tipo 3 e como associar MO à credencial e ao SMS original.
- Lista completa de `submit_status_code`, semântica de `UNKNOWN`, correções de resultados finais, identificação estável de um evento, ordenação e retries após as três tentativas documentadas.
- Exemplos de homologação com headers e corpos reais sanitizados, incluindo rejeição, não entrega e MO.

`CALLBACK_INGRESS_ENABLED` e `MO_INGRESS_ENABLED` permanecem desligados em Dublin HML. O envio ao fornecedor continua com `callback_level=NONE`; não houve callback externo ou SMS real nesta implementação. Nenhuma migration, tabela, rota ou configuração de produção foi adicionada. A suíte de testes cobre os três formatos, códigos conhecidos e desconhecidos, repetição equivalente e campos inválidos.
