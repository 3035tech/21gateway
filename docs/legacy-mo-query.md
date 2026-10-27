# Consulta legada de respostas MO

GET /v1/consultmo?date=<Unix em segundos> implementado com Basic Auth e whitelist do LegacyGuard. Controller/service/DTO próprios para consulta legada, reutilizando model/repository MO, criptografia e infraestrutura de datas. Manual páginas 8–9 confirma que date é obrigatório e representa o dia a consultar. Não foram inventados parâmetros de paginação.

## Comportamento

Dia em America/Sao_Paulo, com tzdata embutido; intervalo UTC calculado por limites de calendário [início, próximo dia), respeitando dias históricos de 23/25 horas. Filtro por received_at e credential_id, somente eventos associados a um envio. Ordem estável received_at/id. Nenhuma chamada ao fornecedor, alteração de estado, consumo de eventos ou modificação da fila.

Consulta funciona independentemente de MO habilitado e de callback entregue/falho; conta/empresa ainda precisam estar ativas para autenticar. Eventos antecipados não associados são expostos depois da reconciliação. Texto é descriptografado com MO_CONTENT_KEY apenas para a conta autenticada. Falha de chave/conteúdo aborta a resposta inteira com indisponibilidade, sem lista parcial ou metadados criptográficos.

Resposta: status OK, code 5, message REQUIRED_OK, list_mo. Campos exatos: fromAdress, receivedDate, unixReceivedDate, msg, idMo, idMt. IDs/Unix em string, datas no formato 2017-04-18 16:05:01-03. Conteúdo Unicode/linha nova preservado. Lista vazia: code 6, message NO_RECORDS e list_mo vazio; texto de NO_RECORDS continua provisório porque o manual não o informa.

## Proteções e limites a homologar

- date obrigatório, não negativo, limitado ao último segundo do ano 9999; datas inválidas retornam erro HTTP de validação. O envelope de erro -20 mencionado no manual não tem formato/mensagem completos e permanece pendente de comparação com legado.
- Timeout de consulta cinco segundos e limite defensivo de 1000 registros por dia. Busca 1001 para detectar excesso; nesse caso retorna indisponibilidade, nunca lista truncada. Isso é proteção interna, não limite informado pelo manual. Homologar volume/continuação antes de ativar clientes com dias maiores. Não adicionar page/next arbitrários à API V1.
- Fuso São Paulo é consistente com os exemplos -03, mas precisa ser confirmado contra tráfego. IDs usam os registros locais, não IDs históricos importados; equivalência da migração ainda pendente.
- Conteúdo MO é revelado somente para o cliente autorizado, sem entrada em logs administrativos. HTTP middleware usa Cache-Control no-store. Retenção/rotação de chave permanecem pendentes conforme docs/mo-flow.md.
- Saldo, agendamento e /v1/status/date não foram implementados nesta etapa. Normalização segue no backlog. Ingresso real do fornecedor continua sem contrato e desligado.

## Migração e prova

Migração aditiva 008: índice parcial credential_id/received_at/id para eventos associados. Aplicar explicitamente antes do binário novo; rollback de binário mantém índice e dados, sem apagar conteúdo/filas.

Testes de serviço/HTTP cobrem day bounds, horários de verão, nome/tipo de campos, Unicode, requisições repetidas, autenticação/IP com header forjado, date ausente/inválido, conteúdo corrompido, isolamento e excesso sem truncamento. Integração PostgreSQL opcional inclui início/fim de dia, consulta com MO desabilitado, repetição e isolamento; depende de TEST_DATABASE_URL. Sem banco real não declarar SQL/transações homologados.
