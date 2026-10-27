# Fluxo interno MO durável

## Estado e camadas

POST /internal/mo → validação → evento criptografado → associação por credencial/UUID → outbox → worker → POST legado ao cliente. Controller/service/model/DTO/repository/worker próprios do domínio MO. Reutiliza autenticação interna, transporte HTTP/SSRF, estados, backoff e constantes de lease/tentativas. Nenhum payload do white label foi presumido. Envio continua callback_level NONE; /v1/consultmo implementado sobre eventos locais conforme docs/legacy-mo-query.md.

## Configuração e contrato

MO_INGRESS_ENABLED=false por padrão. Habilitar exige CALLBACK_INGRESS_TOKEN de pelo menos 32 bytes e MO_CONTENT_KEY: chave de 32 bytes em base64, independente de PROVIDER_TOKEN_KEY, igual na API e worker. Gerar com openssl rand -base64 32 e guardar em ambiente seguro/gestor de segredos. Chave ausente impede persistência; chave inválida impede startup. Worker MO inicia quando a chave está configurada, mesmo com ingress desligado para drenar fila anterior. Compose mantém ingress false e passa chave à API/worker; nunca expor no frontend.

Contrato interno Bearer: eventKey (1–200 caracteres), credentialId interno, providerMessageId UUID do envio, phoneNumber (1–15 dígitos), message (UTF-8, 1–4096 bytes), receivedAt RFC3339 (até cinco minutos no futuro). Limites são defensivos; não representam limites Zoug. Sem normalização de texto/telefone. O adaptador interno autorizado resolve a conta; cliente externo não acessa esta rota.

204 após persistência. Repetição da chave na mesma conta com dados iguais é idempotente. Alterações retornam 409, assim como UUID conhecido de outra credencial. UUID não associado permanece pendente; reconciliação exige UUID e credencial iguais. Não atribuir conta por telefone ou correlationId. Respostas sem envio correlacionável exigem contrato adicional do fornecedor.

## Conteúdo e segurança

Migração 007 adiciona mo_events/mo_callback_jobs, sem alterar tentativas SMS ou armazenar texto de SMS enviados. MO precisa de conteúdo para reentregar: AES-GCM com nonce aleatório e AAD vinculado a credencial/eventKey; fingerprint HMAC-SHA256 da entrada canônica permite deduplicação sem hash público do texto. Texto só é aberto no worker autorizado; não expor conteúdo, chaves ou fingerprints em logs/DTOs administrativos.

Sem política definida, não há expurgo automático: conteúdo cifrado, telefone e datas ficam armazenados. Definir retenção antes da ativação de produção. Guardar chave junto do plano de backup; rotação exige recriptografia e atualização dos fingerprints. Multiversão/rotação ainda não implementadas: não trocar chave cegamente.

## Outbox e entrega

Evento, associação e job são transacionais; callbacks antecipados reconciliados no worker. URL fotografada no job. Antes de enviar, conta/empresa precisam estar ativas, MO habilitado e URL atual igual à registrada. Mudança de URL ou bloqueio impede entrega ao destino antigo; não redireciona jobs automaticamente. MO desabilitado na associação não gera job retroativamente ao habilitar depois.

SKIP LOCKED, lease de 30 segundos e token de posse; quatro workers MO adicionais aos quatro DLR, compartilhando transporte com limite de conexões por host. Até oito tentativas/backoff existentes; proposta de 24h ainda não adotada. Somente 2xx confirma, redirects e redes privadas bloqueados. Entrega pelo menos uma vez: timeout/crash pode repetir. Header X-21Mobile-Event-ID usa mo:<ID> para evitar colisão com DLR; Mobile21Identifier permanece número no JSON.

Conclusão é persistida com contexto independente do cancelamento HTTP. Conteúdo inválido/conta bloqueada falham sem entrega; conteúdo indisponível recebe CONTENT_UNAVAILABLE e tentativas limitadas. Falhas públicas não incluem corpo remoto. Listagem/reprocessamento manual MO ainda não expostos no painel.

## Migração e prova

Aplicar 007 antes da API/worker. Tabelas aditivas preservam leitores antigos. Rollback de binário mantém fila/conteúdo; retomar requer a mesma chave. Não apagar tabelas em rollback.

Testes HTTP de opt-in/Bearer/ciphertext; serviço de validação/nonces/fingerprints/vínculo conta-evento; worker de autorização/decrypt/payload/falha/conclusão após cancelamento; adaptador de JSON/Unicode/SSRF/sem retry próprio. PostgreSQL opcional ampliado com evento antecipado, deduplicação/conflito, reconciliação, lease/token antigo, bloqueio MO e conflito entre contas. Sem TEST_DATABASE_URL, não comprova SQL/transações reais.

Ativação externa depende de payload/auth Zoug, IDs e datas do legado, associação Master/Sub, retenção e homologação. Leia docs/mo-callback.md para ambiguidades de saída. Nenhum callback ou SMS real foi enviado.

Consulta legada MO adicionada em etapa posterior conforme docs/legacy-mo-query.md; não ativa ingresso real do fornecedor.

Evolução operacional: retenção de conteúdo MO definida em 90 dias e keyring/recriptografia implementados conforme docs/operations.md; prevalece sobre a ausência de política descrita na etapa inicial. Pendências ativas ficam protegidas; HMAC permanece estável na rotação da cifra.
