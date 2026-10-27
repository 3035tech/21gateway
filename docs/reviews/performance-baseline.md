# Baseline local — 14/10/2026

Go 1.27.2, darwin/arm64, Apple M3 Pro. Uma execução de um segundo por benchmark, sem detector de races nesta medição. Comando: go test ./internal/services -run '^$' -bench BenchmarkSMS -benchtime=1s -benchmem.

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| SMSNormalization | 17.745 | 1.848 | 11 |
| SMSBatch200 | 468.831 | 414.700 | 2.427 |

Batch200 corresponde a um lote de 200 itens, com normalização, tradução e dependências em memória. Normalização usa texto de entrada próximo ao limite. Não inclui autenticação Argon2id, banco, HTTP/latência do fornecedor, callback ou réplicas concorrentes. Não é teste de carga integrado, SLA, capacidade de produção ou garantia de 200 TPS. Repetir apenas quando houver mudança relevante e comparar em condições equivalentes.

## Scheduler — 15/10/2026

Apple M3 Pro/darwin arm64, benchtime1s, repositórios em memória, sem banco/rede/auth:

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| ScheduledAdmission200 | 682217 | 344438 | 4204 |
| SMSBatch200 | 570566 | 433865 | 2427 |

Admissão de 200 itens inclui normalização, UUID e AES-GCM. Não mede throughput PostgreSQL nem vazão do fornecedor. Sem certificação de 200TPS; testes de fila/índices/locks e endpoints com histórico grande continuam pendentes.
