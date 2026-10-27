# Operação do gateway

## Estado desta etapa

Usuário autorizou os itens 3 (contratos), 4 (telefone) e 5 (operação) e definiu retenção de **90 dias para conteúdo MO**. Contratos de paginação/saldo/agendamento e corte telefônico 4/5 continuam aguardando evidência; nenhuma regra foi presumida. Implementações operacionais abaixo não equivalem a deployment/homologação de produção.

## Retenção confirmada

MO_RETENTION_DAYS=90 é o padrão. Worker remove apenas encrypted_message de eventos recebidos há mais de 90×24 horas, já associados/processados e sem job pending/processing. Marca content_purged_at; mantém IDs, fingerprint, telefone/datas e histórico de entrega. /v1/consultmo não retorna conteúdo expurgado. As senhas/tokens e o texto de respostas recentes não são alterados.

Eventos sem associação ou com entrega ativa ficam protegidos mesmo após 90 dias. Diagnóstico inclui quantidade e idade dos eventos não associados para resolver exceções; não interpretar o prazo como destruição incondicional de pendências. Prazo dos metadados, arquivos de backup e eventuais cópias externas ainda precisa de política própria.

Limpeza inicia ao subir o worker e repete a cada cinco minutos, até dez lotes de cem por ciclo com contexto limitado. Só grava IDs/contagens no log, sem conteúdo. MO_RETENTION_DAYS=0 desativa a rotina para manutenção. Migração 009 deve ser aplicada antes do novo worker/API.

Prévia manual, sem escrita:

```bash
docker compose run --build --rm --no-deps --entrypoint /maintenance backend -action purge-mo
```

Com -apply, aplica **um lote** de até cem; o worker executa a limpeza contínua. Não remover dados manualmente para resolver falhas de entrega.

## Chaves versionadas

Tokens do fornecedor e conteúdo MO continuam separados. Compatibilidade v1:<ciphertext> preservada. Formato novo v2:<key-id>:<ciphertext> identifica a chave; AES-GCM/nonce aleatório e AAD continuam na biblioteca padrão Go.

| Configuração | Uso |
| --- | --- |
| PROVIDER_TOKEN_KEY | Chave legada v1 de tokens |
| PROVIDER_ENCRYPTION_KEYS | JSON secreto: IDs e chaves AES-256 base64 |
| PROVIDER_ACTIVE_KEY_ID | ID usado em novas gravações; vazio mantém v1 |
| MO_CONTENT_KEY | Chave legada v1 de conteúdo |
| MO_ENCRYPTION_KEYS / MO_ACTIVE_KEY_ID | Keyring e ID ativo para conteúdo MO |
| MO_FINGERPRINT_KEY | Chave estável de HMAC de deduplicação; por compatibilidade usa MO_CONTENT_KEY se omitida |

Não inserir segredos em CLI, documentação, frontend ou arquivos versionados. Usar ambiente/gestor de segredos. JSON/key IDs inválidos e chave ativa ausente falham no startup. Key IDs aceitam letras/números/_/- até 32 caracteres.

### Procedimento de rotação

1. Fazer backup e garantir recuperação das chaves e do arquivo age. Validar primeiro em banco isolado.
2. Distribuir binários novos **com todas as chaves de leitura**, mantendo active IDs vazios e chaves legadas. Todos os leitores precisam entender v2 antes de ativar escrita v2.
3. Configurar nova chave/ID ativo no keyring de todas as réplicas API/worker. Manter chaves antigas para leitura; não mudar a chave HMAC de deduplicação.
4. Executar prévia: `docker compose run --build --rm --no-deps --entrypoint /maintenance backend -action rotate-keys`.
5. Com resultado correto, repetir com `-apply`. Comando percorre tokens/conteúdos em lotes, verifica decrypt, recriptografa e usa comparação do ciphertext antigo para não sobrescrever edição/expurgo concorrente.
6. Repetir prévia até eligible=0/conflicts=0, validar leitura/callbacks e confirmar réplicas sem escrita antiga. Falhas retornam completed=false/contagens e exit code não zero; pode retomar sem repetir linhas já migradas.
7. Só retirar chaves antigas de leitura depois de verificar dados, backups e janela de rollback. HMAC estável precisa permanecer: rotação do fingerprint é outra migração e não é feita por este comando.

Nunca reutilizar ID existente com outro material. Binário antigo não lê v2; rollback requer compatibilidade de leitores/dados, não apagar tabelas. Expurgo de conteúdo é irreversível pelo fluxo normal e é a política autorizada de retenção. Backups mais antigos podem conter conteúdo já expirado: aplicar retenção antes de reabrir serviço restaurado e definir ciclo de vida desses backups.

## Diagnóstico e alertas

/healthz verifica processo; /readyz verifica banco. GET /internal/operations é opt-in com OPERATIONS_ENABLED=true e Bearer OPERATIONS_TOKEN próprio (mínimo 32 bytes). Desativado por padrão; manter rota em rede operacional/HTTPS no deployment. Não reutilizar senha de cliente ou token do fornecedor.

Retorna pending, processing, leases expirados, falhas na última hora, data do job pendente mais antigo, eventos não associados/data mais antiga e instante da observação, separados DLR/MO. Sem URLs, telefones, payloads, chaves ou hashes. Timeout cinco segundos; índices parciais para filas e falhas recentes. Snapshot não é transação consistente entre todas as consultas nem histórico de séries temporais.

Plano para monitor externo: observar /readyz, crescimento/idade das filas, leases vencidos persistentes e falhas. Alertas devem distinguir indisponibilidade HTTP do cliente, falha de banco e falta de associação de UUID. Destino desejado continua relacionamento@21mobile.com.br, mas nenhum e-mail/alarme AWS foi ativado. Definir limiares e infraestrutura antes de automatizar alertas; não enviar conteúdo SMS ou segredos no alerta.

## Evidência das tentativas de callback

Implementado em 15/10/2026 a partir da resposta 22 do formulário: registrar falhas para diagnóstico da entrega DLR/MO. Usa `log/slog` da biblioteca padrão, com atributos tipados via `LogAttrs`; worker escreve JSON em stdout. Sem nova dependência, tabela, tela ou serviço.

Falha HTTP/transporte gera `Callback HTTP attempt failed` com `event_id`, `method`, `http_status`, `failure_code` e `duration_ms`. `http_status=0` significa que não foi obtida resposta HTTP, não um status retornado pelo cliente. Erros de conexão também podem envolver DNS, TLS, cancelamento ou bloqueio de segurança; não atribuir automaticamente a falha ao cliente. Redirect não conta como entrega e continua sem seguimento automático.

Conclusão do worker gera `Callback attempt failed` (WARN), `Callback delivery recovered` (INFO, sucesso após mais de uma tentativa) ou `Callback outcome persistence failed` (ERROR). Campos: `callback_kind` (`dlr`/`mo`), `job_id`, `event_id`, `attempts`, `delivered`, `failure_code`, `outcome_persisted`. Recuperação só é registrada após sucesso de Finish; HTTP aceito com falha de banco tem `delivered=true` e `outcome_persisted=false`. Isso não significa conclusão persistida nem impede duplicação na retomada: entrega permanece pelo menos uma vez.

O evento HTTP usa o identificador enviado ao destinatário (`19` para DLR, `mo:1370` para MO). O evento de conclusão usa ID numérico junto com `callback_kind`; `job_id` permite correlacionar com outbox. Tentativas diferentes podem ter o mesmo ID de evento. Eventos HTTP e persistência são registros separados; o status HTTP não foi adicionado à tabela de jobs nem à API do painel.

Não registrar URL/host/query, correlationId, destinatário, texto, corpos/headers da resposta, tokens ou mensagem original do erro de rede/banco. São dados externos que podem carregar conteúdo e credenciais. Entrega bem-sucedida na primeira tentativa não gera registro individual nem aloca atributos de conclusão, reduzindo volume no caminho comum.

Consultar localmente sem ativar comunicação externa:

```sh
docker compose logs --since 1h worker
```

Logs dependem da coleta/retenção configurada pela operação; não são trilha imutável ou armazenamento de histórico completo. Coleta centralizada, limiares e alertas por e-mail/AWS continuam pendentes. Não foi definido prazo de retenção dos logs; 90 dias refere-se ao conteúdo MO.

Testes sem rede verificam DLR/MO com HTTP 503, redirect, erro de transporte e sucesso, retorno sem retry automático, ausência de dados sensíveis, falha de persistência e recuperação. Nenhum contrato externo, política de oito tentativas/backoff, credencial ou payload legado foi alterado. Prova PostgreSQL continua pendente.

## Backups e restauração

Scripts reutilizam pg_dump/pg_restore do PostgreSQL no Compose e [age](https://github.com/FiloSottile/age). O dump flui direto para criptografia; não há dump aberto em disco. Saída com permissão 0600, arquivo temporário privado, rename só após sucesso. Requer age instalado, Docker acessível e destinatário público de backup no ambiente.

```bash
# BACKUP_RECIPIENT contém a chave pública age configurada pela operação.
./scripts/database-backup.sh ./backups
# BACKUP_IDENTITY aponta para a identidade privada age, fora do repositório.
./scripts/database-restore-check.sh ./backups/ARQUIVO.dump.age
```

Restauração cria um banco scratch com nome próprio; usa --exit-on-error e remove só esse banco na saída. Não restaura por cima do banco ativo. Verifica estrutura básica; não comprova decrypt de chaves, aceite de aplicação, RPO ou RTO. Guardar identidades age/chaves AES/HMAC em plano de recuperação independente; perder a chave torna o backup/conteúdo inutilizável.

Scripts não criam agendamento, bucket ou política AWS. Para produção ainda definir destino protegido, frequência/PITR, retenção de backups e teste periódico de recuperação. Não versionar .dump/.dump.age/identidades; padrões adicionados ao .gitignore. Testes dos scripts usam CLIs falsos para provar isolamento/cleanup/atomicidade; não representam backup/restauração reais nem prova criptográfica do age.

## Desempenho

Benchmarks locais de normalização e lote de 200 em backend/internal/services/sms_benchmark_test.go. Rodar:

```bash
cd backend
go test ./internal/services -run '^$' -bench BenchmarkSMS -benchmem
```

Dependências simuladas: sem Argon2id, PostgreSQL, rede ou SMS reais. Baseline em docs/reviews/performance-baseline.md; não afirmar capacidade de 200 TPS a partir dele.

Roteiro [k6](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/) em ops/load/ready.k6.js: 50 GET /readyz por segundo por 30 segundos, somente loopback HTTP, sem redirects, credenciais ou envios. `k6 run ops/load/ready.k6.js`. Mede disponibilidade com banco pronto; não mede SMS. k6 não estava instalado, então roteiro apenas validado sintaticamente.

Teste completo ainda necessário: ambiente isolado com banco e fornecedor simulado, oito contas, limites de 1 requisição/s por conta e 200 mensagens/s agregado, mistura single/lote, DLR/MO, p95/p99, timeouts e recuperação de lease. Usar destinatários fictícios e bloquear saída real. Não apontar carga de envio para produção.

## Prova disponível

Testes Go: keyring legado/novo, binding, deduplicação estável, dry-run/retomada/CAS, retenção 90 dias e diagnóstico opt-in/Bearer. PostgreSQL opcional ampliado para SQL da rotação, expurgo preservando pending e contagens operacionais. Banco real continua necessário para executar essas provas. Nenhuma chave, banco, fila, backup ou infraestrutura de produção alterados nesta tarefa.

## Limites de autenticação

API_AUTH_IP_LIMIT_PER_MINUTE=600 limita pedidos antes do Basic Auth por peer IP e inclui pedidos válidos/consultas. Dez falhas/minuto por conta provocam 429 até o próximo minuto; backoffice mantém dez logins/IP e limita falhas por e-mail. Não substituir IP por XFF sem proxy confiável configurado. Validar NAT/ALB/capacidade antes de publicar e acompanhar login_limits/cleanup. Relatório completo: docs/reviews/security-assessment-2026-10-15.md. Não altera limite de envio nem troca senhas legadas.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migração011 obrigatória nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. Prova PostgreSQL/SQS/carga ainda pendente; testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.
