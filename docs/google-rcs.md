# Google RCS for Business — conector preparado

Em 31/10/2026, o catálogo inclui Google → RCS. O backend contém `providers/rcs` e `integrations/google` para envio de **texto** via `phones.agentMessages.create`, OAuth de conta de serviço e validação de webhooks. Nenhum endpoint, job, credencial ou rota de webhook Google está registrado em runtime. A empresa pode guardar Google/RCS apenas como rota `planned`, sem ativação. Não há envio real nem homologação de agente.

Decisão confirmada pelo usuário em 31/10/2026: a 21 ainda não possui parceria direta com o Google. O painel identifica essa dependência na rota planejada. O código de protocolo não representa acesso à API; não cadastrar credenciais fictícias, ativar a rota ou publicar webhook até obter conta parceira, agente e dados de homologação. A rota por canal permanece Google → RCS, separada de Zoug → SMS.

## Contrato coberto

- Envio usa número E.164 com `+`, `agentId` e `messageId` UUID fornecido pelo chamador, `contentMessage.text` (até 3072 caracteres), OAuth scope `rcsbusinessmessaging` e HTTPS. Não há fallback automático nem retry dentro do cliente. HTTP 200 exige `name` do recurso correspondente ao telefone e UUID; falha de transporte ou confirmação ambígua retorna resultado desconhecido. `404` é tratado como destinatário não alcançável; outros `4xx` documentados são rejeições. Erros públicos não contêm corpo de resposta, credencial ou número.
- Webhook: desafio `verification` valida `clientToken` e devolve `secret`; callbacks `message_callback` e `agent_callback` exigem HMAC-SHA512 em Base64 sobre o `message.data` decodificado, com comparação em tempo constante e limite de corpo. O parser devolve JSON autenticado sem supor semântica de eventos. Nenhum handler HTTP chama o parser nesta etapa.
- O tipo `rcs.Sender` é independente de `sms.Sender`. O adaptador Google não é registrado no roteador/worker SMS, e `AvailableForSMS` continua apenas Zoug.

## Para ativar

1. Obter aprovação da conta de parceiro Google, credenciais da conta de serviço, agente e `agentId`, webhook token por parceiro/agente, acesso de testes e aceite do agente nas operadoras/mercados pretendidos.
2. Definir domínio RCS com credenciais cifradas e vínculo empresa ↔ agente, consentimento/opt-out, payloads e UI de RCS, limites, orçamento e retenção. Persistir `messageId` antes de despachar; resultado desconhecido não pode disparar novo envio automático. DLR/read/inbound precisam de IDs e estados próprios, sem equivalência inventada com SMS.
3. Criar ingresso durável de callbacks assinados e deduplicados, com resposta 200 somente após armazenamento; processar de forma assíncrona. A fila de callbacks do parceiro pode afetar todos os agentes. Tratar unsubscribe antes do tráfego real.
4. Homologar credenciais, envio, entrega, leitura, falha, destinatário não RCS, opt-out, limites e custos com o agente de teste. Só então liberar Google/RCS na validação de empresa e na composição de runtime. Atualizar OpenAPI e criar migrações versionadas para o novo domínio.

Documentação oficial: [envio REST](https://developers.google.com/business-communications/rcs-business-messaging/reference/rest/v1/phones.agentMessages/create), [recurso AgentMessage](https://developers.google.com/business-communications/rcs-business-messaging/reference/rest/v1/phones.agentMessages), [webhook](https://developers.google.com/business-communications/rcs-business-messaging/guides/integrate/webhooks), [conta de parceiro](https://developers.google.com/business-communications/rcs-business-messaging/guides/get-started/partner-account). A documentação pública é suficiente para este adaptador de protocolo, mas não fornece os dados e aprovações específicos da 21 para uso em produção.

Validação local em 31/10/2026: `go test -race ./...`, `go vet ./...`, `go build ./...`, `npm run lint`, `npm run build`, teste do formulário e `docker compose config --quiet` passaram. Os testes do adaptador usam transporte e token falsos; não validam a API real, cobrança, agente ou operadoras. A validação Compose cobre apenas a configuração, não a execução de contêineres/PostgreSQL.
