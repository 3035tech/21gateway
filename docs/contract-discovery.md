# Levantamento: paginação, saldo e agendamento

Verificado em 14/10/2026. Trabalho de descoberta de contratos; nenhum endpoint novo implementado, requisição autenticada ao fornecedor ou mensagem enviada. Não concluir que um recurso inexiste por não estar documentado.

## Fontes e resultado

| Fonte | Evidência | Limite |
| --- | --- | --- |
| docs/references/21-mobile-api-2026.pdf, páginas 6–7 | GET status/date, date Unix obrigatório, até 100 itens, referência/destino opcionais e link para restantes | Não informa nome/estrutura do link ou parâmetro de continuação; tabela usa correlationid, prosa correlationId |
| [PHP oficial status_date.php](https://github.com/21Mobile-Repo/API-Tutorial/blob/master/php/status_date.php) | Chama /v1/status/date/?date=... com Basic Auth e imprime resposta | Não decodifica resposta nem segue páginas; exemplo de 2017, não confirma comportamento atual |
| [Java oficial ConsultaStatus.java](https://github.com/21Mobile-Repo/API-Tutorial/blob/master/Java/Requisicoes/ConsultaStatus.java) | consultaComData recebe Unix long e chama mesma URL com barra final | Não interpreta continuação; exemplo de 2017 |
| docs/references/white-label-api.docx, revisão 10/02/2026 | Lista POST /api/send e /api/send/batch; X-API-TOKEN; 402 em saldo insuficiente | Nenhum endpoint de consulta de saldo ou campo/método de agendamento documentado |
| [Zoug oficial](https://devyx.com.br/en/products/zoug) | Produto oferece integração API e automação/campanhas | Página comercial, sem schemas/rotas de saldo ou agendamento |

O navegador conseguiu ler GitHub e páginas Devyx; o leitor web falhou com cache miss/403 e curl sem acesso retornou HTTP 000. Falhas dessas ferramentas não comprovam inexistência dos recursos. Não foi feita exploração por tentativa de endpoints de produção.

## Resultado por contrato

- **Paginação status/date: ainda não confirmada.** Não implementar next/page/offset/cursor como se documentados. Também conferir forma com barra final usada pelos exemplos e grafia dos filtros antes do rollout.
- **Saldo: contrato do fornecedor ainda não obtido.** HTTP 402 de envio não é API de consulta nem valor disponível. Preservar /v1/credits como requisito, sem saldo local calculado ou zero fictício.
- **Agendamento: contrato do fornecedor ainda não obtido.** Restrições de dia/horário descritas na autenticação são regras de permissão, não evidência de agendamento. Manter rejeição explícita atual para scheduledDate até definir integração.

## Evidências necessárias para encerrar

### Legado / 21 / 3035

Fornecer requisição e resposta sanitizadas de /v1/status/date para um dia com mais de 100 mensagens: primeira página, continuação real, última página e dia vazio. Preservar nomes/tipos de campos, query, status HTTP e headers de paginação; trocar destinos/referências por dados fictícios, retirar Authorization/cookies/tokens. Mostrar chamada com correlationId/correlationid e destination, inclusive continuação filtrada.

Confirmar nome/local do link (JSON ou header), se absoluto/relativo, query de continuação, ordem/desempate, estabilidade enquanto novos estados chegam, se consulta consome algo e como identifica o dia/fuso/data do envio. Os exemplos oficiais não respondem a isso.

### Zoug / Devyx

Enviar documentação técnica/OpenAPI/Postman da versão contratada, URL de sandbox e exemplos com segredos substituídos por placeholders.

**Saldo:** método/path e autenticação; esquema de sucesso e erros; unidade (créditos SMS, segmentos ou dinheiro), tipos/precisão, saldo negativo/pós-pago/ilimitado, reservas e saldo disponível; conta/rota/token consultado; relação Master/Sub; limites de consulta. Exemplo com saldo positivo, zero e conta sem saldo consultável. Confirmar tradução para credits em string do legado.

**Agendamento:** suporte via API single/batch ou recurso separado; método/path e nomes/tipos exatos dos campos; segundos/milissegundos/RFC3339 e fuso; limites futuro/passado, validade e tolerância de atraso; resposta/UUID/estado após aceite; cancelamento/alteração/consulta; reserva/cobrança/liberação de saldo; TPS no aceite e execução; callbacks de entrega/cancelamento; idempotência e comportamento após timeout. Exemplo de envio futuro e lote misto imediato/agendado, se suportado. Se não houver suporte, obter confirmação explícita antes de escolher fila no gateway.

## Solicitação pronta

Mensagens objetivas para encaminhar manualmente em docs/contract-requests.md. Nenhum contato externo disparado. Link do portal técnico solicitado ao usuário; resposta é necessária para consultar documentação privada sem inventar contratos.
