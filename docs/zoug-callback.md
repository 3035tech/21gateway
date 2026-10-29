# Callback SMS Zoug/Devyx

O documento recebido `docs/references/ZOUG_SMS_API_CALLBACK.docx`, atualizado em 11/02/2026, define HTTP POST JSON com `callback_type` 1 (submissão), 2 (entrega) e 3 (MO). O adaptador em `internal/integrations/whitelabel/callback.go` valida os três formatos e traduz somente os resultados conhecidos dos tipos 1 e 2 para o contrato interno DLR. A rota opt-in `POST /webhooks/zoug/{token}` recebe o JSON original; seu token aleatório fica somente na URL e o runtime guarda SHA-256. O ingresso restringe IPs de origem, valida o corpo, persiste idempotentemente DLR/MO e só então responde 204. Erros de persistência retornam 503 para permitir as retentativas da Zoug.

Em 28/10/2026, a Zoug confirmou que não assina callbacks nem permite adicionar assinatura. Os IPs de origem informados foram `18.232.239.74`, `3.213.146.37` e `34.195.124.142`; um segredo na URL cadastrada funciona. Há **uma URL global para todas as contas/tokens**, configurada no painel em `/me`. A URL enviada em cada requisição deveria prevalecer, mas a Zoug confirmou que atualmente é ignorada por um bug; aguardar aviso da correção. `SUBMIT_ONLY` envia apenas tipo 1, `DELIVERY_ONLY` envia tipos 2–5 e `ALL` envia todos. Tipos 4 e 5 são eventos de RCS, fora do escopo SMS.

O `message_id` corresponde ao ID retornado no envio e também vincula DLR e MO ao MT. A Zoug tenta entregar cada callback até três vezes, considera qualquer resposta 2xx sucesso e descarta após a terceira falha. O MO usa `message_id` para resolver a credencial e o número de origem a partir do MT persistido; `reference_id` e `source_addr` podem faltar. Sem MT correspondente, responde 503 para não confirmar o descarte. O texto MO é cifrado antes da gravação. `ZOUG_CALLBACK_ENABLED=false` mantém a rota ausente; ao ativá-la, envios unitários solicitam `callback_level=ALL` para receber tipos 1, 2 e 3. O lote Zoug não documenta callback_level, portanto sua cobertura permanece não homologada.

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

## Configuração e pendências

- Configurar `ZOUG_CALLBACK_ENABLED=true`, `ZOUG_CALLBACK_TOKEN_SHA256` (hash SHA-256 hex de um token aleatório com pelo menos 32 bytes) e `ZOUG_CALLBACK_ALLOWED_IPS` com os três IPs informados. `TRUSTED_PROXY_CIDRS` deve refletir apenas os proxies reais que definem `CF-Connecting-IP`; provar o IP efetivo atrás do túnel antes de cadastrar a URL. Não registrar a URL completa nem o corpo MO em logs. Rotacionar o token gerando novo hash e URL.
- Cadastrar a URL global `https://21gateway-back.3035service.com/webhooks/zoug/{token}` em `/me`; a Zoug confirmou que atualmente ignora `callback_url` por requisição e sempre usa a URL do painel.
- A associação MO por `message_id` está implementada; validar em tráfego real se o telefone de origem deve ser derivado do destino do MT quando `source_addr` estiver ausente.
- Reconciliar tentativas antigas marcadas `unknown` com UUID válido antes de esperar DLR/MO para elas. Novas tentativas preservam o UUID mesmo quando o status de aceite é inesperado, sem converter `unknown` em sucesso ou reenviar SMS.
- Confirmar a lista completa de `submit_status_code`, `report_status` e `err` com o fornecedor final, além de ordenação e eventuais correções de estados finais.
- Exemplos de homologação com headers e corpos reais sanitizados, incluindo rejeição, não entrega e MO.

`CALLBACK_INGRESS_ENABLED` e `MO_INGRESS_ENABLED` controlam somente as rotas internas normalizadas e permanecem independentes. A rota Zoug não exige migration nova. O acesso por IP via `CF-Connecting-IP` só é confiável quando o peer de conexão estiver em `TRUSTED_PROXY_CIDRS`; verificar o caminho real de HML antes de ativar. Tipos 4 e 5 de RCS ficam fora do escopo e não são aceitos como SMS.
