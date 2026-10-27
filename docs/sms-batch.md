# Envio individual e em lote

## Implementação disponível em código

POST /v1/send aceita de um a 200 itens. Um item utiliza /api/send; vários utilizam /api/send/batch conforme documentação fornecida. Basic Auth legado e whitelist da conta precedem o envio. O token Zoug vem da credencial autenticada, criptografado com AES-GCM no banco e recuperado somente no backend; não há fallback para um token global.

Resultados por item preservam a ordem original e correlationId. IDs internos numéricos e UUIDs do fornecedor continuam associados. No lote, o adaptador usa position para mapear resultados, aceita 200/207 e erros estruturados em 400. Posição ausente, resposta incompleta, UUID inválido/duplicado ou resultado incerto não viram sucesso presumido. Nenhum retry automático de SMS.

Tentativas são criadas em uma transação antes de chamar o fornecedor. Resultados do lote são atualizados em uma operação parametrizada PostgreSQL, com contexto de conclusão limitado a três segundos, mesmo após desconexão do cliente. Falha ao gravar um aceite não muda sua resposta para rejeição nem provoca reenvio. Nenhum texto de SMS é armazenado.

Destinos fora do formato brasileiro de 11 dígitos são recusados localmente por item, com `INVALID_NUMBER_FORMAT`, sem chegar ao fornecedor ou à persistência. Mensagens vazias em lote continuam encaminhadas ao fornecedor para resultado individual. Regras próprias não representáveis no fornecedor também continuam locais: agendamento não habilitado, texto acima de 160 caracteres e referência acima de 50. Inválidos locais não são enviados para evitar envio imediato/acréscimo de segmentos não solicitado. O DTO documenta as mesmas restrições de tamanho, formato e data no OpenAPI; a validação por item preserva o envelope de resposta legado.

Encoding do lote é global como exige a Zoug: GSM7 quando todos os textos enviados são representáveis; UCS2 quando necessário pelo comportamento atual. Essa implementação NÃO resolve a normalização de acentos/emojis requerida pelo legado: a regra de transformação foi solicitada ao usuário e ainda depende de exemplos. Não declarar homologação integral do texto.

Número brasileiro válido de onze dígitos recebe 55. Entrada de dez dígitos ainda é rejeitada nesta implementação até confirmar a regra exata do nono dígito; não enviar número potencialmente transformado incorretamente. O usuário recebeu pergunta sobre transformação de telefone/texto. Não interpretar ausência de resposta como confirmação.

## Limites e operação

Runtime usa SendLimits PostgreSQL: uma requisição por segundo por credencial e no máximo 200 mensagens por segundo no conjunto, em janela móvel. Conta por mensagens, inclusive tentativas recusadas pelo fornecedor. Reserva é compartilhada entre réplicas; indisponibilidade do limitador falha fechada, sem fallback local. NewSMSService sem limitador permanece para testes; cmd/api compõe o limitador real.

Banco tem lock transacional global no cálculo/reserva, índice por data e limpeza de reservas antigas. Volume real, concorrência e duração do lock ainda precisam ser medidos. Teto agregado é escolha conservadora até confirmar se os 200 TPS são globais, por conta ou por rota. Requisição sem orçamento retorna limite imediatamente; não fica guardada para enviar depois.

## Homologação pendente

Teste PostgreSQL opcional foi ampliado para configurações, senha legada, limitador e atualização em lote. Ainda não executado por acesso negado ao Docker e ausência de TEST_DATABASE_URL. Adaptador fornecedor verificado com httpmock; não representa envio real/homologação Zoug.

Continuam pendentes regra exata de texto/nonos dígitos, códigos reais do PHP, agendamento, DLR/MO Zoug, consultas/créditos e importação dos hashes existentes. Bibliotecas e domínio separados conforme AGENTS.md.
