# Escolha do framework Go

Decisão: manter Chi + Huma + net/http para o gateway e backoffice. Reavaliar somente com evidência de limitação funcional ou profiling do fluxo completo.

Chi é router compatível com net/http; permite reutilizar sessões, middlewares e testes existentes. Huma cobre contratos tipados, validação e geração de OpenAPI. MVC/separação por domínio pertence à organização do código, não depende de trocar de framework.

Referências primárias verificadas em 14/10/2026: [Chi](https://github.com/go-chi/chi), [Huma OpenAPI](https://huma.rocks/features/openapi-generation/), [Huma validação](https://huma.rocks/features/request-validation/). Fiber é alternativa, mas sua integração com handlers net/http passa por adaptação; sua documentação explicita overhead dessa camada: [Fiber adaptor](https://docs.gofiber.io/middleware/adaptor/). Não trocar somente por rankings de benchmark sintético.

Não há teste de carga real comprovando 200 TPS neste ambiente. Medir autenticação Argon2id, banco, persistência de lotes, conexões/latência do fornecedor, filas de callback e p95/p99. Limites e concorrência são necessários independentemente do router.

Teto conservador atual: 200 mensagens/s agregado e uma requisição/s por credencial, controlados em janela compartilhada no PostgreSQL. O fornecedor ainda deve confirmar o escopo do teto por token/rota/global. A implementação não transforma requisições limitadas em envio assíncrono posterior.
