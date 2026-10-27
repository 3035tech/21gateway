# Avaliação de segurança — 15/10/2026

## Escopo e conclusão

Rodada local autorizada pelo usuário sobre API legada e backoffice, com revisão de código e testes de ataques controlados. Não é pentest completo nem certificação. Nenhum ambiente externo, fornecedor, SMS real, credencial de produção ou teste destrutivo foi utilizado.

Lacuna confirmada: o limite de envio era posterior ao Basic Auth; portanto não impedia tentativas sequenciais de senha. O semáforo Argon2id limitava concorrência, não número de palpites ao longo do tempo. Backoffice já tinha limite por IP, mas faltava limite de falhas por conta. Correções implementadas abaixo; SQL compartilhado novo ainda precisa de PostgreSQL real.

## Correções e comportamento

| Tema | Implementação | Limite material |
| --- | --- | --- |
| API, força bruta | Runtime usa AuthenticationAttempts compartilhado no PostgreSQL antes do hash: orçamento de IP e consulta de falhas por username; dez falhas bloqueiam novas verificações da conta até mudar o minuto | Conta pode sofrer bloqueio temporário provocado por terceiro; tentativas já em andamento podem concluir após o limite |
| API, troca de usuários/IPs | Orçamento por peer IP cobre todos os pedidos de autenticação, inclusive inválidos; falhas de conta são independentes do IP | Ataque distribuído por muitas contas/IPs exige proteção de borda/monitoramento adicional |
| Backoffice | Mantidos dez pedidos de login/minuto/IP; adicionadas dez falhas/minuto por e-mail normalizado, mesmo mudando origem | O segundo controle depende de SQL real ainda não exercitado |
| Falha do limitador | Erro de consulta/registro recusa autenticação com 503; não permite acesso por indisponibilidade do controle | Pode indisponibilizar login durante problema de banco; comportamento deliberado |
| Senhas internas | Novas senhas e bootstrap: mínimo 15 caracteres Unicode, máximo 128 bytes; bloqueio de exemplos comuns e repetição de um único caractere | Pequena denylist local, não base abrangente de senhas vazadas ou garantia de força |
| Senhas existentes | Login continua verificando hash sem impor política nova; senhas API legadas não trocadas nem rejeitadas por comprimento mínimo novo | Senha legada fraca ainda pode ser acertada antes do bloqueio; whitelist/rotação voluntária continuam importantes |
| Headers HTTP | nosniff/no-store existentes; adicionados X-Frame-Options DENY e Referrer-Policy no-referrer, antes de CORS/sessão | Headers da API não configuram o hosting Amplify do frontend |

Sem dependências novas ou migração adicional: reutiliza login_limits da migração 003 e cleanup do worker. A política de senha usa Argon2id existente; não implementa algoritmo de hash próprio, pontuação de entropia ou consulta externa de senhas.

## Política e impacto operacional

- Janelas fixas de minuto do relógio PostgreSQL, não janela móvel de 60 segundos. Pode haver burst na virada do minuto. Não há bloqueio permanente, CAPTCHA, MFA ou reset manual novo.
- API_AUTH_IP_LIMIT_PER_MINUTE padrão 600, configurável entre 1 e 1.000.000. É orçamento adicional de autenticação compartilhado por peer IP, incluindo pedidos válidos de envio e consultas. Não substitui 1 envio/s por conta nem orçamento de 200 mensagens/s.
- NAT/proxy pode concentrar vários clientes no mesmo orçamento. Medir consumo real e ajustar antes do deployment; não inferir capacidade a partir desse valor. Na AWS, definir proxy confiável/IP real/whitelist antes de liberar tráfego. Não confiar automaticamente em Forwarded/X-Forwarded-For recebidos do usuário.
- IP é normalizado sem porta, incluindo IPv4 mapeado em IPv6. IP inválido usa bucket comum unknown-peer. E-mail interno é normalizado por trim/lowercase; username API mantém comparação legada exata.
- Chaves de limite são namespaces + SHA-256 da identidade, sem armazenar username/e-mail/IP abertos nesse campo. Isso não é anonimização contra enumeração e não substitui proteção do banco.
- Falhas por conta são consultadas antes de verificar senha e incrementadas depois de falha. Sucesso não limpa contadores. Requisições concorrentes já aprovadas podem ultrapassar dez falhas; o limite de quatro verificações Argon2id simultâneas por serviço/réplica limita custo local, não DDoS distribuído.
- Contadores IP usam UPSERT atômico e saturam para evitar overflow. Novos registros não têm TTL automático do banco: cleanup depende do worker em funcionamento.
- Após bloqueio da conta, até a senha correta recebe 429 até o próximo minuto. Monitorar abuso que provoque esse bloqueio. Payloads de sucesso e credenciais existentes permanecem iguais, mas 429 pré-autenticação é comportamento novo a incluir na homologação.

## Testes executados

| Cenário | Prova e resultado |
| --- | --- |
| API: múltiplas senhas e múltiplos peers | Router/controlador/Argon2id reais, repositório e orçamento em memória; dez falhas 401, pedidos seguintes 429, nenhum SMS persistido/enviado |
| API: peer bloqueado e headers falsos | Orçamento simulado pré-esgotado; troca de X-Forwarded-For/Forwarded não contorna 429 |
| Backoffice: brute force | Controller/middleware HTTP reais com serviço de login negado e orçamento simulado; treze pedidos, apenas dez chegam à verificação; nenhuma sessão criada |
| Enumeração API | Usuário conhecido com senha errada, desconhecido e username com SQL malicioso recebem o mesmo corpo 401 e challenge Basic |
| CPU/indisponibilidade | Testes de serviço: quatro slots ocupados impedem nova verificação; falhas nas três etapas do limitador recusam acesso |
| SQL injection | GORM real em DryRun: payloads OR/UNION/pg_sleep e metacaracteres ficam em Vars, SQL mantém placeholders; sort malicioso recusado. Não executa SQL |
| Acesso sem sessão/CSRF/CORS | PUT/DELETE administrativos sem sessão recusados, login/logout de origem maliciosa recusados; origem não liberada pelo CORS |
| Payload excessivo | /v1/send autenticado rejeita corpo acima de 1 MiB com 413, sem persistir envio |
| Senhas | Frases longas aceitas; curtas, exemplos comuns, caractere repetido e espaços de preenchimento recusados; UI acompanha mínimo/caracteres/bytes |
| Dependências frontend | npm audit --json: zero vulnerabilidades conhecidas reportadas, 331 dependências avaliadas nesta execução |

Suíte existente também verifica permissões, fixação/revogação/logout de sessão, cookies HttpOnly/Secure/SameSite, whitelist, SSRF em callbacks, isolamento e privacidade. Parte da prova de isolamento/transações continua dependente de PostgreSQL; não confundir testes isolados com execução integrada.

Revisão estática frontend não localizou dangerouslySetInnerHTML, eval, innerHTML ou armazenamento de sessão em localStorage/sessionStorage nos fontes consultados. Isso não equivale a teste dinâmico de XSS no navegador.

## Prova PostgreSQL preparada, não executada

TestPostgresSecurityAuthenticationAndInjection: bloqueio por e-mail normalizado, expiração de janela, vinte reservas concorrentes com somente dez aceitas, payloads SQL contra repositories reais, ordenação recusada e senha legada preservada. Usa schema exclusivo da fixture compartilhada e não altera banco/contas existentes.

Docker permanece com permission denied no socket; TEST_DATABASE_URL ausente. Não foi comprovado o UPSERT/contagem concorrente, expiração real ou autenticação interna com as consultas novas. Executar:

```sh
docker compose --profile verification run --build --rm backend-test
```

Modo obrigatório não permite ignorar falta do DSN. O arquivo do teste não é evidência de aprovação no banco.

## Validação desta execução

Go: go test -race ./..., go vet ./... e go build ./... passaram. Frontend: lint/build e 15 testes passaram. Configuração Compose válida. Testes PostgreSQL sinalizaram skip explícito pela ausência do DSN; não foram aprovados em banco real.

## Pendências priorizadas

1. Executar PostgreSQL real e validar concorrência/janelas/cleanup, carga de autenticação e capacidade de pool.
2. Definir topologia AWS, TLS, proxy/IP real, limites por IP compartilhado e proteção/alertas de borda. Validar HSTS e headers/CSP do hosting frontend sem quebrar Amplify/API.
3. Implementar MFA interno com enrollment/recuperação/revogação definidos; ainda ausente.
4. Revisar senhas legadas e ampliar denylist com fonte adequada, sem registrar/enviar senhas ou forçar mudança automática dos clientes.
5. Rodar scanner Go de vulnerabilidades: govulncheck não instalado nesta máquina; npm audit não cobre backend. Executar DAST/browser em ambiente isolado, incluindo XSS, permissões/IDs com banco, sessão entre origens e configurações de hosting.
6. Homologar 429 pré-autenticação e impacto de NAT, monitoramento de bloqueios temporários e liberação/recuperação operacional.

Não executados: scan ZAP/sqlmap contra serviço publicado, fuzzing amplo, teste de invasão de infraestrutura, DDoS, exploração de dependências Go ou testes de MFA inexistente. Não afirmar ausência geral de vulnerabilidades.

## Referências

- [OWASP — Authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html): throttling, mensagens genéricas e política de senhas.
- [OWASP — SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html): parâmetros e allowlists.
- [OWASP — Credential Stuffing Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Credential_Stuffing_Prevention_Cheat_Sheet.html): limites da proteção baseada somente em IP.

Polish de segurança em 15/10/2026: frontend agora conta mínimo da senha interna sem espaços nas pontas como backend, com limites centralizados e mensagens claras; senha original preservada na validação/envio. Regressão reproduzida antes da correção. Testes de configuração fixam orçamento de autenticação no ambiente de teste para evitar influência do terminal. Sem alteração de senha legada ou limite de autenticação em produção.
