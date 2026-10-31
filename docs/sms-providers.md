# Provedores de SMS e canais

Decisão de 31/10/2026: manter `POST /v1/send` e suas respostas legadas como SMS, sem mudar payload, IDs, códigos ou limites. A empresa escolhe uma rota de provedor por canal. A identidade do provedor é independente do canal: o mesmo código poderá atender SMS e, em evoluções próprias, RCS ou WhatsApp. Esses canais não usam automaticamente o contrato de SMS.

## Etapa implementada

A migração 028 cria `company_provider_routes(company_id, channel, provider_code)`, preenchendo a rota SMS das empresas existentes com `zoug`, e marca `sms_attempts.provider_code` com padrão `zoug`. Novas empresas recebem a rota SMS na mesma transação do cadastro. O painel mostra o provedor da rota SMS; somente Zoug está disponível para seleção. O serviço valida o código e a resolução da conta falha fechada diante de provedor desconhecido. A autenticação Basic Auth dos clientes e o token Zoug existente permanecem iguais. O envio unitário, lote e agendamento gravam o código do provedor escolhido na tentativa.

Essa etapa **não conecta um segundo provedor** nem cria opções fictícias. Um novo adaptador requer contrato de envio, lote, autenticação, identificação de mensagem, DLR/MO, limites e homologação. A interface de envio Go ainda usa tipos originados no pacote `whitelabel`; antes de registrar outro adaptador, extrair tipos neutros e mapear os payloads no pacote de cada fornecedor. O identificador externo e os eventos DLR/MO atuais usam UUID e vínculo Zoug; ampliar o schema e a associação por `(provider_code, provider_message_id)` antes de receber IDs de outro fornecedor. O token na empresa continua específico da Zoug; credenciais de outros provedores precisarão de armazenamento cifrado próprio, sem reutilizá-lo.

Rotas de SMS agendados são marcadas no aceite. Antes de permitir troca para outro provedor, o worker deverá despachar pelo código persistido e separar lotes por provedor; não reinterpretar agendamentos antigos pela rota atual. Resultado incerto nunca autoriza fallback automático para outro provedor, pois pode duplicar a mensagem. Preço, capacidades, prioridade e alternância também dependem de dados reais e decisão da 21.

RCS e WhatsApp poderão ter linhas próprias na tabela de rotas quando seus contratos, credenciais, payloads, callbacks e operação forem definidos. Não foram ativados nesta etapa.

Aplicar migração 028 antes de publicar os binários. A migração é aditiva; um rollback de binário pode deixar a tabela e a coluna sem uso. Após o deploy, verificar a versão em `migrations`, a rota SMS das empresas e o envio controlado somente quando houver autorização de destino; testes locais não enviam SMS real.

Em HML, a migração 028 foi aplicada em 31/10/2026 antes do novo binário: `gateway21_hml` registrou versão 028, duas rotas SMS/Zoug para duas empresas e a coluna nas tentativas. Nenhum SMS foi enviado para essa verificação.
