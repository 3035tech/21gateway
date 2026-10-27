# Validação dos fluxos de aceite

Atualização de 15/10/2026. Requisito: envio SMS e entrega de DLR/MO ao cliente, conforme questionário final. Nenhum SMS ou callback real foi enviado.

## Prova executada sem rede

`backend/internal/services/acceptance_test.go`, teste `TestAcceptanceSMSDLRAndMOAdapters`: normalização SMS → resolução de token por conta → cliente white label real com transporte HTTP simulado → persistência simulada do UUID; tradução GET DLR com erro HTTP e nova tentativa explícita; ingresso MO normalizado → criptografia → worker → POST legado preservando texto, correlação e IDs.

HTTP simulado com httpmock existente; destinos não registrados falham, sem abertura de sockets. Não utiliza payload DLR/MO presumido do fornecedor, ingress HTTP, autenticação HTTP ou banco real. Autenticação HTTP possui testes separados existentes. A nova tentativa explícita de DLR neste teste não comprova agendamento/backoff do worker.

## Prova preparada para PostgreSQL, ainda não executada

`TestPostgresSMSCallbackAcceptance` usa schema exclusivo, migrations reais e repositories/services/workers/adaptadores de produção. Apenas HTTP externo é simulado. Cobre:

- SMS com token da conta, tradução de texto, UUID persistido e ID legado.
- DLR e MO repetidos sem jobs duplicados; destino retorna 503 primeiro e 204 na tentativa seguinte.
- Estado pendente/backoff após falha, ausência de tentativa antecipada e conclusão após liberação controlada do horário, sem sleep.
- MO associado à conta errada rejeitado; polling DLR e consulta MO isolados por credencial; entrega de callback não consome polling.
- Bloqueio de jobs DLR/MO se a URL cadastrada mudar antes da entrega.

Fixture PostgreSQL compartilhada com `TestPostgresBackofficeAndOutbox`: schema independente por teste, cleanup registrado mesmo em falhas, execução idempotente das migrations. A suíte anterior cobre leases, eventos antecipados, rotação e retenção MO de 90 dias. Não declarar esses cenários comprovados sem execução com banco real.

Executar na raiz do projeto, com Docker acessível:

```sh
docker compose --profile verification run --build --rm backend-test
```

O perfil define `REQUIRE_DATABASE_TESTS=true`; falta de DSN falha em vez de resultar em skip. Essa proteção foi exercitada localmente e falhou como esperado sem TEST_DATABASE_URL. Sem modo obrigatório, ambos os testes PostgreSQL sinalizam skip explícito.

## Correção de autorização DLR

`repositories/callbacks.go`: DeliveryData agora exige que URL do job coincida com URL DLR atual da credencial. Com URL alterada, worker encerra job com ACCESS_BLOCKED, sem enviar ao destino antigo nem redirecionar automaticamente ao novo. É o mesmo comportamento existente para MO. Administradores precisam considerar jobs pendentes ao trocar URL; não há migração automática dos jobs antigos.

## Resultado desta execução

- Go: `go test -race ./...`, `go vet ./...`, `go build ./...` passaram. Testes PostgreSQL explicitamente pulados por ausência de TEST_DATABASE_URL.
- Frontend: lint, build e 14 testes passaram.
- Compose: configuração válida. Scripts backup/restauração: três testes com ferramentas simuladas passaram.
- Docker: permission denied no socket; PostgreSQL real, SQL da correção, backup criptografado e restauração reais permanecem sem validação. Testes de scripts não equivalem a uma restauração real.
- Nenhuma dependência, migração, infraestrutura ou tela nova adicionada.

Contratos DLR/MO do fornecedor, paginação status/date, saldo/agendamento e critério do nono dígito continuam pendentes. Esta validação não substitui homologação com fornecedor e piloto.
