package httpapi

import "github.com/danielgtaylor/huma/v2"

const clientIntegrationIntroduction = "## Integração com a API 21 Mobile\n\n" +
	"1. Solicite à **21 Mobile** a credencial técnica da sua empresa. A 21 Mobile gera e entrega o **usuário e a senha Basic Auth**; eles são diferentes do token Zoug, que permanece sob gestão da 21 Mobile. Não existe usuário ou senha padrão.\n" +
	"2. Use a URL base do ambiente informada pela 21 Mobile, sempre por **HTTPS**, e envie o cabeçalho `Authorization: Basic base64(usuario:senha)`. Ferramentas como curl montam esse cabeçalho com `-u 'USUARIO:SENHA'`. Se houver lista de IPs autorizados, faça as chamadas da rede cadastrada.\n" +
	"3. Teste o acesso com `GET /v1/auth/check` a partir da rede que fará os envios. HTTP **204** confirma autenticação e IP, sem enviar SMS. Depois use `POST /v1/send` com JSON.\n\n" +
	"Cada endpoint abaixo mostra os campos, exemplos de entrada e saída, limites e significado dos resultados. **Aceite de envio não significa aceite do fornecedor nem entrega ao celular quando o modo de fila estiver ativo.** Guarde `id_sms` e consulte `GET /v1/send/result?id_sms=...` para saber o estado da submissão. Não repita automaticamente uma tentativa com resultado incerto; `correlationId` é referência de rastreamento, não chave de idempotência. Os retornos de DLR/MO reais ainda dependem da homologação do fornecedor."

func describeClientIntegration(spec *huma.OpenAPI) {
	spec.Info.Title = "21 Mobile · API de integração SMS"
	spec.Info.Description = clientIntegrationIntroduction
	spec.Tags = []*huma.Tag{{Name: "SMS", Description: "Autenticação, envio e consultas disponíveis para integrações de clientes."}}
	if basic := spec.Components.SecuritySchemes["basicAuth"]; basic != nil {
		basic.Description = "Usuário e senha Basic Auth gerados e fornecidos pela 21 Mobile para a empresa. Envie sobre HTTPS; não use o token Zoug neste cabeçalho."
	}

	if path := spec.Paths["/v1/auth/check"]; path != nil && path.Get != nil {
		op := path.Get
		op.Summary = "Validar a credencial Basic Auth e o IP de origem"
		op.Description = "Chame da **mesma rede do cliente** que usará a API. Envie o usuário e a senha recebidos da 21 Mobile por HTTP Basic Auth. **204** indica que a credencial está ativa, dentro da validade e autorizada para o IP visto pelo gateway. Nenhum SMS é enviado e nenhuma consulta é consumida. **401** indica falha de autenticação; **403**, IP não permitido; **429**, limite de tentativas; **503**, indisponibilidade temporária. Um teste feito apenas no painel não comprova a autorização da rede do cliente.\n\nExemplo: `curl -i -u 'USUARIO:SENHA' 'https://SEU_HOST/v1/auth/check'`."
		if response := op.Responses["204"]; response != nil {
			response.Description = "Credencial e IP autorizados; resposta sem corpo."
		}
	}

	if path := spec.Paths["/v1/send"]; path != nil && path.Post != nil {
		op := path.Post
		op.Summary = "Enviar de 1 a 200 SMS"
		op.Description = "Envie JSON com `sms` contendo **1 a 200 itens**. Cada `destination` tem **11 dígitos** (DDD + celular), sem 55, espaços ou pontuação. `messageText` aceita até **160 caracteres** na entrada; o gateway remove acentos e emojis conforme a regra de normalização. `correlationId` é opcional (até 50 caracteres) e volta no resultado, mas **não evita duplicações**. `scheduledDate` é Unix em segundos; omita ou use zero para envio imediato. Envio futuro só é aceito quando o agendador do ambiente estiver habilitado.\n\nA resposta HTTP **200** é uma lista com **um resultado por item, na mesma ordem**. Em modo de fila, `status: OK` e `code: 5` confirmam aceite durável para processamento, **não aceite do fornecedor ou entrega ao aparelho**. Guarde `id_sms` e consulte `/v1/send/result`. No modo síncrono antigo, `OK` confirma aceite do fornecedor. `NOK` indica falha conhecida ou resultado incerto; veja `message`. `SEND_OUTCOME_UNKNOWN` exige reconciliação antes de qualquer novo envio. Por compatibilidade com a API legada, `sms` vazio ou itens com telefone/texto inválido também podem retornar HTTP **200** com `status: NOK`, `code: -5` e `message` como `INVALID_REQUEST`, `INVALID_NUMBER_FORMAT` ou `INVALID_MESSAGE_FORMAT`. Confira **cada item** da lista, inclusive quando o HTTP for 200. JSON malformado, autenticação e falhas de infraestrutura usam status HTTP de erro. Limite por credencial: uma requisição por segundo; existe também orçamento compartilhado de mensagens.\n\nExemplo: `curl -u 'USUARIO:SENHA' -H 'Content-Type: application/json' -d '{\"sms\":[{\"destination\":\"11987654321\",\"messageText\":\"Teste de integracao\",\"correlationId\":\"pedido-123\"}]}' 'https://SEU_HOST/v1/send'`."
		setRequestExamples(op, map[string]*huma.Example{
			"envioImediato": {Summary: "Um SMS imediato", Value: map[string]any{"sms": []map[string]any{{"destination": "11987654321", "messageText": "Teste de integracao", "correlationId": "pedido-123"}}}},
		})
		setResponseExamples(op, "200", map[string]*huma.Example{
			"aceito":           {Summary: "Tentativa aceita; entrega ainda não confirmada", Value: []map[string]any{{"status": "OK", "code": 5, "id_sms": "123", "correlationId": "pedido-123"}}},
			"itemInvalido":     {Summary: "Telefone inválido; HTTP 200 exige conferir cada item", Value: []map[string]any{{"status": "NOK", "code": -5, "message": "INVALID_NUMBER_FORMAT"}}},
			"resultadoIncerto": {Summary: "Resultado incerto; não reenviar automaticamente", Value: []map[string]any{{"status": "NOK", "code": -5, "id_sms": "124", "message": "SEND_OUTCOME_UNKNOWN", "correlationId": "pedido-124"}}},
		})
	}

	if path := spec.Paths["/v1/send/result"]; path != nil && path.Get != nil {
		op := path.Get
		op.Description = "Informe o `id_sms` recebido em `/v1/send` para consultar o resultado da submissão sem consumir o registro. A credencial Basic Auth só pode consultar seus próprios envios; um ID de outra credencial retorna **404**. `pending` significa na fila ou aguardando o horário agendado; `accepted` significa aceito pelo fornecedor, ainda sem confirmação de entrega; `rejected` é falha conhecida; `unknown` é resultado incerto e **não deve ser reenviado automaticamente**. `deliveryStatus` é opcional e reflete apenas o DLR recebido. A resposta não expõe telefone nem texto.\n\nExemplo: `curl -u 'USUARIO:SENHA' 'https://SEU_HOST/v1/send/result?id_sms=123'`."
		setResponseExamples(op, "200", map[string]*huma.Example{
			"naFila":               {Summary: "Aceito para processamento", Value: map[string]any{"id_sms": "123", "state": "pending", "createdAt": "2026-10-28T10:00:00Z"}},
			"aceitoPeloFornecedor": {Summary: "Fornecedor aceitou; entrega não confirmada", Value: map[string]any{"id_sms": "123", "state": "accepted", "createdAt": "2026-10-28T10:00:00Z", "dispatchedAt": "2026-10-28T10:00:01Z"}},
		})
	}

	if path := spec.Paths["/v1/status"]; path != nil && path.Get != nil {
		op := path.Get
		op.Summary = "Consultar até 100 estados de entrega novos"
		op.Description = "Retorna até **100 estados DLR ainda não consultados** para esta credencial, somente de envios com `correlationId`. A leitura **consome** os estados retornados antes de responder: repetir a chamada não devolve os mesmos registros, salvo nova atualização de entrega. Não há paginação nesta rota. `status: OK`, `code: 5` e `sent_status` indicam registros; `code: 6` e lista vazia indicam ausência de novos estados. `NO_RECORDS` é texto provisório. Datas são apresentadas em **America/Sao_Paulo** e também em Unix como string. `SENT` não significa `DELIVERED`. O ingresso de DLR real do fornecedor ainda precisa ser homologado; uma lista vazia não confirma entrega nem falha.\n\nExemplo: `curl -u 'USUARIO:SENHA' 'https://SEU_HOST/v1/status'`."
		setResponseExamples(op, "200", map[string]*huma.Example{
			"comEstado":       {Summary: "Um estado de entrega", Value: map[string]any{"status": "OK", "code": 5, "message": "REQUIRED_OK", "sent_status": []map[string]any{{"correlationId": "pedido-123", "destination": "5511987654321", "sentDate": "2026-10-27 18:00:00-03", "unixSentDate": "1793134800", "status": "DELIVERED"}}}},
			"semNovosEstados": {Summary: "Nenhum estado novo", Value: map[string]any{"status": "OK", "code": 6, "message": "NO_RECORDS", "sent_status": []any{}}},
		})
	}

	if path := spec.Paths["/v1/consultmo"]; path != nil && path.Get != nil {
		op := path.Get
		op.Summary = "Consultar respostas SMS de um dia"
		op.Description = "Informe `date` como **Unix timestamp em segundos** de qualquer instante do dia desejado em **America/Sao_Paulo**. A consulta cobre o dia inteiro, retorna respostas ligadas aos envios desta credencial e **não consome** os registros. `code: 5` indica respostas; `code: 6` e `list_mo: []` indicam ausência. `NO_RECORDS` é texto provisório. Use a grafia publicada `fromAdress` exatamente assim. Se o volume ultrapassar o limite defensivo atual de 1000 registros, o gateway responde indisponibilidade em vez de devolver uma lista truncada. O ingresso MO real do fornecedor ainda depende de homologação.\n\nExemplo: `curl -u 'USUARIO:SENHA' 'https://SEU_HOST/v1/consultmo?date=1793134800'`."
		setResponseExamples(op, "200", map[string]*huma.Example{
			"comResposta":  {Summary: "Uma resposta recebida", Value: map[string]any{"status": "OK", "code": 5, "message": "REQUIRED_OK", "list_mo": []map[string]any{{"fromAdress": "5511987654321", "receivedDate": "2026-10-27 18:00:00-03", "unixReceivedDate": "1793134800", "msg": "Confirmo", "idMo": "45", "idMt": "123"}}}},
			"semRespostas": {Summary: "Nenhuma resposta no dia", Value: map[string]any{"status": "OK", "code": 6, "message": "NO_RECORDS", "list_mo": []any{}}},
		})
	}
}

func setRequestExamples(op *huma.Operation, examples map[string]*huma.Example) {
	if op.RequestBody == nil {
		return
	}
	if content := op.RequestBody.Content["application/json"]; content != nil {
		content.Examples = examples
	}
}

func setResponseExamples(op *huma.Operation, status string, examples map[string]*huma.Example) {
	if response := op.Responses[status]; response != nil {
		if content := response.Content["application/json"]; content != nil {
			content.Examples = examples
		}
	}
}
