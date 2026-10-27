# Backlog de decisões e compatibilidade

## Normalização de texto e telefone

Adiado explicitamente pelo usuário em 14/10/2026: seguir com outras implementações e retornar posteriormente a este tema. Não bloquear configurações de conta, whitelist, lote e consultas por essa decisão.

- Confirmar transformação de acentos/emojis: exemplo fictício “Olá, ação! 😊”, caracteres removidos/substituídos/rejeitados e quando contar os 160 caracteres.
- Confirmar inclusão do nono dígito em entradas de dez dígitos, possíveis exceções e exemplos do legado.
- Capturar evidência dos fontes/tráfego ou confirmação da 21 antes de ativar uma transformação.

Estado atual: texto SMS normalizado conforme docs/sms-normalization.md após autorização posterior. Entrada de dez dígitos continua rejeitada e onze dígitos válidos recebem 55. Não declarar compatibilidade integral das regras telefônicas nem transformar números sem evidência.

## Dependências externas ainda necessárias

Contratos Zoug de DLR/MO/autenticação, consultas/saldo e idempotência (agendamento local autorizado e implementado posteriormente em docs/sms-scheduling.md); export de contas/hashes/configurações do legado; herança Master/Sub; plano de migração/aceite e retenção. Estes itens não são resolvidos pela decisão de adiar a normalização.

## Compatibilidade das consultas V1

Confirmar mensagem de retorno vazio (code 6), semântica exata de consumo/atualizações, destino e horário retornados em /v1/status. Para /v1/status/date, obter exemplo com continuação: nome/formato do link, parâmetros de página e grafia correlationid/correlationId. Implementação de /v1/status descrita em docs/legacy-status.md; não considerar contrato integral homologado.

Consulta MO local implementada em docs/legacy-mo-query.md; confirmar fuso/IDs/erros do legado e volume diário/continuação para retirar ou ajustar o limite defensivo sem truncamento.

Levantamento externo atualizado em docs/contract-discovery.md: PHP/Java oficiais não mostram continuação; saldo/agendamento não obtidos nas fontes disponíveis. Pedidos técnicos objetivos em docs/contract-requests.md, ainda não enviados.

## Esclarecimento recebido sobre normalização — 14/10/2026

Usuário trouxe captura da resposta da 21 ao exemplo de texto: “Remover” e resultado “Ola, acao!”. Para “Olá, ação! 😊”, evidência confirma retirar diacríticos preservando letras e remover o emoji. Não interpretar “remover” como apagar toda letra acentuada. Regras gerais de símbolos, espaços finais e contagem de 160 antes/depois da transformação ainda não confirmadas. Nenhuma normalização ativada nesta etapa; usuário deixou implementação para retomada.

Telefone: interlocutor diz inserir 9 depois do DDD em números de dez dígitos dependendo do primeiro dígito do número local, mas ainda vai confirmar se o corte é 4 ou 5. Fixo não recebe o 9. Exemplos da captura: 513474044 permanece; 515474044 vira 5195474044. Os exemplos escritos têm nove e dez dígitos, respectivamente, portanto não comprovam uma entrada completa de dez dígitos/saída de onze. Precisamos de exemplos completos e confirmação do corte; não deduzir regra definitiva ou ativar transformação com essa evidência parcial.

## Evolução autorizada após a captura corrigida

Usuário autorizou implementar agora. Texto normalizado em single/batch conforme docs/sms-normalization.md; adiamento anterior do texto revogado. Contagem de 160 permanece na origem até confirmação. Telefone corrigido para exemplo 516474044 → 5196474044, ainda nove/dez dígitos; corte 4/5 não confirmado. Regra telefônica permanece pendente.

Retenção do conteúdo MO confirmada em 90 dias pelo usuário; implementação em docs/operations.md. Prazos de metadados/backups permanecem separados. Rotação da cifra e diagnóstico preparados; infraestrutura de alertas/backup automático e carga integrada ainda pendentes.

## Evidência de callbacks — 15/10/2026

Resposta 22 do formulário implementada com logs seguros: HTTP status/código/duração/evento nas falhas e conclusão do worker com tipo DLR/MO, job/evento/tentativas e outcome_persisted. Recuperação só após Finish bem-sucedido; sucesso inicial sem log por mensagem. Worker usa slog JSON em stdout. Sem URLs, texto, telefones, correlationId, tokens ou erros originais. Detalhes em docs/operations.md. Sem migration, dependência, alteração de payload ou política de retry. Coleta/retenção/alertas e PostgreSQL real continuam pendentes.

## Segurança e ataques controlados — 15/10/2026

Relatório em docs/reviews/security-assessment-2026-10-15.md. Corrigida falta de throttling pré-Basic Auth: runtime injeta AuthenticationAttempts PostgreSQL, API com orçamento por peer IP configurável (API_AUTH_IP_LIMIT_PER_MINUTE padrão 600, inclui tráfego válido), falhas por conta bloqueiam após dez até o próximo minuto; backoffice mantém dez logins/IP e ganha limite por e-mail. Sem confiança em XFF, janelas fixas e possível excesso por tentativas já em curso. Falha do limitador fecha acesso. Novas senhas internas/bootstrap mínimo 15 caracteres/máximo128 bytes, pequena denylist; login e senhas legadas preservados. Headers DENY/no-referrer adicionados. Testes de ataques HTTP e SQL DryRun; suíte PostgreSQL nova preparada, não executada. npm audit zero vulnerabilidades conhecidas; govulncheck não instalado. MFA, DAST/browser, infraestrutura/proxy/IP real e prova de banco ainda pendentes. Sem SMS ou tráfego externo de ataque.

## Agendamento local e volume — 15/10/2026

Usuário autorizou scheduler interno e reforçou alto volume/desempenho em todos os endpoints. Agendamento local implementado em docs/sms-scheduling.md, revogando a dependência de agendamento do fornecedor: migração010, chave AES independente/keyring, aceite durável/id_sms preservado, pedidos mistos, worker por conta até200, SKIP LOCKED/lease/CAS, limites compartilhados, nenhum retry após dispatch/resultado incerto e limpeza do conteúdo terminal. Reutiliza o sender existente via adaptador de tentativas. Opt-in desligado até configuração/prova PostgreSQL. Diagnóstico inclui scheduler; resumo agrega contagens em quatro consultas. Sem prova de TPS/SQL real; limites e pendências (TTL/quota/cancelamento/aviso de falha local) explícitos.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migração011 obrigatória nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. Prova PostgreSQL/SQS/carga ainda pendente; testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.
