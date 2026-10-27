# Respostas da 21 e impacto no escopo

Fonte recebida em 14/10/2026: docs/references/perguntas-migracao-respondidas.docx, cópia do arquivo fornecido pelo usuário. Extraídos texto completo e tabela de contas; não há comentários ou alterações rastreadas. Este registro é análise de escopo, não prova de funcionalidades implementadas ou homologadas.

Versão final recebida em 15/10/2026: `docs/references/perguntas-migracao-respondidas-finalizado.docx`. Comparação textual com a versão anterior: somente respostas 54–61 alteradas; sem comentários ou alterações rastreadas. Esta versão prevalece nesses itens. Plano de execução e aceite em `docs/migration-plan.md`.

## Conclusão

O objetivo passa a ser preservar a API V1 utilizada pelos clientes sem mudança nas integrações. Não basta envio unitário. Manter os recursos existentes até comprovar por tráfego que algum recurso pode ser dispensado. A 21 pretende uma entrega funcional completa, não contratar etapas distintas de funcionalidades.

A plataforma white label concentra gestão operacional e saldo. O painel novo é interno e focado em credenciais/configurações. A implementação atual é base parcial; não atende ainda ao escopo respondido.

## Confirmações e ressalvas por tema

| Tema | Resposta da 21 | Consequência |
| --- | --- | --- |
| Compatibilidade (1, 3–5) | Preservar integrações sem alteração; manter o necessário do contrato atual | Validar domínio, rotas, campos, códigos, respostas e credenciais contra legado real |
| Clientes (2) | Tabela com 26 contas: 20 Master e 6 Sub | Planejar importação e identificação de conta; IDs compostos não cabem em ID inteiro simples |
| Fontes (6) | Solicitar os fontes à 3035 | PHP/tráfego real permanecem necessários; não enviar solicitação externa sem instrução explícita |
| Envio (7–8) | Individual e lote até 200; fornecedor deve informar itens inválidos e mostrá-los no relatório | Adaptar resultados por item e preservar ordem; evitar descartar itens localmente sem confirmar contrato |
| Referência/ID (9–10) | Usuário confirmou em 15/10/2026 que os clientes usam correlationId; dependência/formato de id_sms ainda a verificar | Preservar correlationId em envio, resposta e DLR/MO. Repetição não esclarecida; não tratar como chave de idempotência |
| Agendamento (11) | Acredita necessário; suspeita uso pelo grupoamaro | Incluir como requisito a confirmar por tráfego; mecanismo e suporte do fornecedor pendentes |
| Texto (12–13) | Até 160, sem acentos/emojis; há remoção no sistema, localização desconhecida | Reproduzir normalização real; ainda não decidir rejeição, transliteração, remoção e momento de contar 160 |
| Telefone (14) | Brasil, DDD+número com/sem nono dígito; legado acrescenta nono quando ausente | Aceitar a entrada documentada de 10/11 dígitos após validar regra exata de transformação |
| Outras regras (15) | Sem regras adicionais informadas | Não adicionar regras de remetente/horário/duplicidade sem evidência |
| DLR/MO (16–18) | Ambos necessários; GET de status e POST de respostas preservados | Implementar os dois adaptadores de callback; outbox DLR existente é apenas preparação |
| URLs (19–20) | Depende do cliente; autenticação adicional desconhecida | Levantar destinos DLR/MO reais por conta; tabela 2 não contém URLs ou uso de recursos |
| Estados (21) | DLR pode ser apenas resultado final | Definir estados finais com fornecedor/legado; não confundir aceite com entrega |
| Falhas callback (22–23) | Sugere até 24h se simples; deseja evidência da indisponibilidade; reprocessamento manual nunca existiu | Prazo de 24h é proposta, não política aprovada; logs seguros/status HTTP úteis; retry manual não obrigatório pelo documento |
| Consultas (24–28) | Manter status, status/date, consultmo e credits; dispensar só com evidência de desuso | API de consulta continua necessária mesmo sem telas operacionais novas |
| Saldo (29) | Preferência por saldo no white label; limites de segurança a discutir | Tradução de /v1/credits depende de API/unidade do fornecedor; não inventar saldo local |
| Histórico (30) | Mínimo; propõe arquivar 2026 separadamente e API desde setembro | Recorte e retenção são propostas; confirmar ano, campos e prazo antes de importar/apagar |
| Token/gestão (31–32) | Token próprio por cliente, contas legadas administradas pela 21 | Substituir token global por resolução de token da conta; esclarecer herança Master/Sub |
| White label (33–38) | Consultar documentação ou fornecedor | DLR/MO, lote com callback, agendamento, consultas, saldo, idempotência e homologação continuam sem contrato confirmado |
| Painel/perfil (39–40) | Equipe interna; um perfil com plenos poderes, provavelmente duas pessoas | Simplificar o produto para administradores; perfis extras atuais não são requisito do escopo respondido |
| Cadastro (41) | Cadastrar clientes que retornarem; editar credenciais; bloquear/excluir | Ajustar cadastro/configurações; definir exclusão lógica para manter histórico e unicidade de credencial |
| Credenciais (42–43) | Manter usuário/senha da API 21; novos acessos do white label; uma credencial por cliente | Importar autenticação existente; não gerar nova senha legada automaticamente; restringir multiplicidade de credenciais externas |
| Configurações (44) | Whitelist de IP, URLs callback e tokens; limites podem interessar | Estes campos entram no painel; limites adicionais permanecem discussão |
| Telas (45) | Apenas gestão de credenciais; usar Zoug para mensagens/status/saldo/falhas | Resumo/envios/callbacks/auditoria atuais são extras; não tratá-los como requisitos do novo painel |
| Auditoria/MFA (46) | Auditoria não solicitada; considera 2FA interessante | MFA é interesse expresso, método/enrollment/recuperação a fechar; auditoria interna mínima pode permanecer sem obrigar tela |

| Volume/TPS (47–49) | Média na faixa de 50k, pico citado 300k, teto Zoug 200 TPS; cerca de 8 clientes simultâneos; 1 requisição/s por cliente | Projetar limite de mensagens separado do limite de requisições; confirmar se 200 TPS é agregado/token/rota e período do pico |
| Latência/indisponibilidade (50–52) | Resposta rápida; callback pode demorar; envio posterior e validade não decididos; acredita que hoje retorna erro | Manter falha imediata/sem retry de SMS como base até decisão; não introduzir fila para enviar SMS tardiamente |
| Alertas (53) | Pode disparar e-mail para relacionamento@21mobile.com.br | Registrar destino desejado; configurar mecanismo/gatilhos antes de ativar, sem enviar e-mail nesta análise |
| Infra/segurança (54–55) | 3035 e Mozhtech; sem requisitos adicionais informados | Divisão de responsabilidades e configuração da infraestrutura ainda precisam ser definidas. Mantém segurança existente e retenção MO de 90 dias aprovada pelo usuário |
| Migração (56–60) | Conclusão em outubro; teste interno, primeiro cliente de produção e demais em conjunto; infinitech/grupoamaro são candidatos | Confirmar dia e piloto escolhido. Pendências antigas e rollback continuam a discutir; não há autorização para descartar mensagens |
| Aceite (61–62) | Envio de SMS; receber DLR/MO do white label e entregar ambos ao cliente da 21; entrega única | Homologar os três fluxos completos e compatibilidade V1. Responsável aprovador ainda não identificado |

Atualização de 27/10/2026: o usuário confirmou que pode haver vários usuários administradores. O cadastro interno pela tela envia convite para o novo usuário definir a própria senha; 2FA por aplicativo autenticador com QR code, recuperação e sessões lembradas estão implementados conforme [admin-security.md](admin-security.md). Administradores têm acesso global; operadores e usuários de consulta precisam de uma ou mais empresas vinculadas e só podem consultar/operar dados dessas empresas. O acesso por empresa já foi autorizado para perfis não administradores do painel; um portal externo de clientes permanece decisão futura.

## Diferenças identificadas na análise inicial (histórico; consultar status.md para estado atual)

1. Token global do fornecedor precisa ser substituído por configuração protegida por conta. Tokens do white label são segredos recuperáveis para uso pelo backend: precisam de criptografia/gerenciador de segredos, não somente hash. Hash continua adequado para senhas de autenticação externa.
2. Envio unitário, telefone estritamente de 11 dígitos e texto Unicode preservado não reproduzem as regras agora relatadas. Lotes, nono dígito e normalização precisam de adaptação com exemplos reais.
3. Agendamento, consultas V1, saldo e MO não implementados. DLR tem estrutura interna/outbox, mas nenhuma tradução homologada do fornecedor.
4. Painel ainda não configura token por cliente, whitelist ou destinos distintos DLR/MO. Permite várias credenciais por empresa; isso precisa ser alinhado ao significado de cliente/conta Master/Sub e à regra de uma credencial.
5. Usuário/senha externos existentes precisam continuar funcionando. Bootstrap/geração de novas credenciais atuais não são ferramenta de migração; rotação deve ser voluntária, não consequência automática da mudança de plataforma.
6. Perfis operator/viewer e telas operacionais são extras existentes. Não removê-los nem apagar dados nesta análise; ajustar a navegação/escopo posteriormente. MFA ainda ausente.
7. Limitador SMS é local ao processo e por requisição, sem orçamento de mensagens agregado. Lote de 200 por segundo por oito clientes pode superar 200 TPS; confirmar teto e tratar sem enfileirar SMS implicitamente.
8. Worker de callbacks atual faz oito tentativas com backoff de segundos/minutos; não equivale à proposta de 24h. Não ativar política final ou contratos não confirmados.

## Pendências necessárias para fechar a execução

- Legado/3035: fontes completos ou amostras sanitizadas de tráfego; export de usuários e formato de verificação das senhas; whitelist, URLs DLR/MO, dependências de consultas/agendamento e regras de normalização/IDs. Não pedir senhas/tokens em conversa ou arquivos versionados.
- Cadastro Master/Sub: cada uma das 26 contas terá token próprio? Subconta herda token/saldo/configuração do Master? Qual conta define cliente para a regra de uma credencial? Qual domínio atual fará cutover?
- Zoug: exemplos e autenticação reais de DLR/MO, estados finais, callbacks de lotes, APIs de saldo/status/MO/agendamento, idempotência/reconciliação, TPS por token/rota/global e homologação.
- Operação: política callback (prazo, intervalos, tentativas), indisponibilidade de SMS, latências de aceite, histórico desde setembro e retenção, método/recuperação de MFA, alertas e infraestrutura/backups.
- Migração: confirmar dia em outubro (ano 2026 inferido do contexto), escolher piloto/aprovador, definir corte de pendências antigas, importação e retorno. Critérios ponta a ponta confirmados; roteiro em docs/migration-plan.md.

## Orientação para próximas implementações

Preservar a separação por domínio exigida pelo usuário. Usar este documento como escopo respondido, docs/status.md como estado do código e o manual legado como contrato documental. Não interpretar preferência ou resposta provisória como decisão fechada. Não remover endpoints por suposição de desuso, não substituir credenciais existentes e não assumir que o fornecedor implementa recursos ausentes da documentação.

A análise atual modifica somente a documentação do projeto. Nenhuma conta/token/URL foi importada, nenhum segredo incluído, nenhum envio/alerta real disparado e nenhuma funcionalidade nova declarada pronta.

## Evolução posterior à análise

O usuário autorizou iniciar as implementações. Token por conta, whitelist, configurações DLR/MO, cadastro preservador de senha conhecida, lote documentado e limitador compartilhado foram adicionados em código após esta análise. As diferenças listadas acima descrevem a situação original; estado corrente em docs/status.md, docs/account-settings.md e docs/sms-batch.md. Nenhum contrato ausente do fornecedor foi presumido.

Decisão posterior do usuário em 15/10/2026: implementar scheduledDate por scheduler local, sem depender de contrato de agendamento do fornecedor. Estado/limitações em docs/sms-scheduling.md; alto volume é requisito explícito.
