> Evolução: persistência/outbox/worker internos adicionados em docs/mo-flow.md. A descrição abaixo registra a etapa original do adaptador; integração real do fornecedor permanece pendente.

# Adaptador de saída MO

## Implementação

Client.DeliverMO no domínio integrations/callback prepara POST JSON para a URL MO do cliente. Reutiliza transport/pool/timeout, validação de URL e tratamento de resposta usados no DLR. Não foi criada rota presumida do fornecedor, worker ou persistência MO.

MOMessage é um modelo transitório normalizado; LegacyMOCallback é DTO específico de saída. Campos preservados do manual: Message, Mobile21Identifier, CorrelationId, MtId, PhoneNumber e Date. CorrelationId é string, pois o exemplo alfanumérico sem aspas do manual é JSON inválido. IDs são números. Content-Type application/json; X-21Mobile-Event-ID usa ID do evento informado pelo chamador. Query existente na URL é mantida sem introduzir campos DLR.

Texto, acentos, emoji e quebras de linha são preservados; a normalização adiada de SMS não é aplicada a respostas MO. Data apresentada em BRT/UTC-3 no formato do exemplo (Tue Jul 03 17:27:02 BRT 2018). Associação do identificador MO e ID do envio, formato de data/histórico de horário de verão, grafia e tipos ainda precisam de homologação contra tráfego real. Adapter recebe IDs duráveis já resolvidos; não transforma UUID em número e não inventa IDs.

Validação defensiva: IDs não zero, data presente, UTF-8 válido, mensagem 1–4096 bytes, correlationId até 50 caracteres e telefone composto de 1–15 dígitos. Limite de conteúdo/telefone é uma proteção interna desta preparação, não limite confirmado do fornecedor; revisar no adaptador de entrada quando seu contrato chegar. Não translitera nem modifica telefone. Mensagens inválidas não chegam ao transporte.

Somente HTTP 2xx confirma entrega; erros públicos tipados sem corpo remoto. Uma chamada não realiza retry: futuro worker durável será responsável pela política. HTTPS/443, sem credenciais/fragmentos; safeurl no dial bloqueia redes privadas/loopback, proxy e redirects desativados. Nenhuma notificação externa real é enviada nos testes.

## Dependências para ativação

1. Payload e autenticação DLR/MO do white label, incluindo origem da conta e relação com envio/UUID. Verificar também respostas sem MT correlacionável; não atribuir cliente pelo telefone sozinho.
2. Identificador estável para deduplicação e semântica Mobile21Identifier/MtId; exemplos legados reais.
3. Persistência/outbox e autorização de conta/empresa antes do dispatch. Caller deve resolver URL MO autorizada da credencial; o adaptador HTTP sozinho não resolve tenancy.
4. MO precisa do conteúdo para encaminhar/consultar: definir criptografia e retenção antes de persistir. As tentativas SMS atuais continuam sem texto; esta mudança não grava conteúdo.
5. Worker com lease/retries e consulta /v1/consultmo, depois que semântica de datas/paginação estiver confirmada.

URLs MO já cadastráveis no painel não acionam DeliverMO. Não declarar integração MO completa. Entrada interna DLR existente permanece opt-in/desabilitada; este adaptador não altera callback_level NONE do envio.

## Verificação

Testes com transporte simulado cobrem JSON/campos/tipos exatos, Unicode/linha nova, datas, query preservada, IDs inválidos, UTF-8/tamanho, URL insegura, loopback e falhas HTTP sem retry. Infraestrutura DLR compartilha dispatch e sua suíte permanece obrigatória. Banco e fornecedor não foram usados.
