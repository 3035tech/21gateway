# Normalização de texto SMS

Implementação autorizada pelo usuário após respostas da 21 em 14/10/2026. Exemplo confirmado: Olá, ação! 😊 → Ola, acao!.

Envio unitário/lote usam o mesmo helper do domínio SMS: validar UTF-8 e até 160 caracteres Unicode na entrada; remover emojis com gomoji v1.4.1 (graphemes via uniseg); NFD/remover marcas Mn/NFC com golang.org/x/text; retirar espaços externos; escolher encoding sobre o resultado. Letras, pontuação, números, moedas não classificados como emoji e quebras internas são preservados. Sequências compostas/família, bandeiras, tons de pele e keycaps cobertos por testes. Não há tabela artesanal de emojis nem remoção indiscriminada de toda escrita não ASCII.

Limite continua na origem: contar somente depois da transformação ainda requer confirmação. Texto unitário que fica vazio é inválido, sem persistência/envio. Lote mantém a decisão anterior de deixar fornecedor reportar itens vazios/inválidos; mensagem que fica vazia chega vazia ao fornecedor e é traduzida pelo resultado individual. Nunca substituir emoji por descrição textual ou enviar texto original quando a normalização falha.

Batch normaliza uma cópia e preserva ordem/correlationId/destino/IDs; não muda a entrada do chamador. Sem retry adicional, agendamento, custo/segmentação presumidos ou persistência do texto. MO recebido/consultado continua preservando o conteúdo original.

## Telefone ainda pendente

Nova captura corrigiu exemplo de começo 5 para 6: 516474044 → 5196474044. Continua com nove/dez dígitos; interlocutor ainda confirma corte 4/5. Não inferir regra geral, completar dígitos ou ativar transformação. Entrada de dez dígitos permanece rejeitada na regra atual até exemplos completos e corte confirmado. Registrar evolução em docs/backlog.md.

## Verificação

Testes unitários e de fluxo single/batch provam exemplo, equivalência de acento composto/decomposto, emoji composto, Unicode residual, moedas/pontuação, limite da origem, texto vazio e entrada do chamador intacta. APIs de saída MO não alteradas. Testes usam transportes simulados; não houve SMS real ou homologação do legado completo.

Bibliotecas: https://github.com/forPelevin/gomoji e golang.org/x/text/unicode/norm. Dependências fixadas em go.mod/go.sum.
