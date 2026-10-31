# Fila operacional

O painel `/pendencias` mostra registros individuais que requerem investigação: envio com resultado incerto sem DLR terminal, agendamento em fila há mais de cinco minutos depois do prazo, callback DLR falho e callback MO falho. A listagem oferece busca por ID ou empresa, filtros de empresa/tipo/estado, paginação, tamanho de página e ordenação. Operadores e usuários de consulta veem somente empresas vinculadas; administradores veem todas. O backend aplica esse escopo, inclusive nas mutações e no histórico.

Um item aparece como **aberto** enquanto a condição existir. Operador ou administrador pode assumi-lo, tornando-o **em análise** com responsável. Somente o responsável pode concluir ou devolver o caso à fila; ambas as ações exigem nota. Cada mudança cria evento imutável com ator, estado, nota e hora. Casos já acompanhados permanecem consultáveis quando a condição original deixa de existir; a interface indica isso sem marcar automaticamente o tratamento como concluído. O estado do caso descreve o acompanhamento humano, não o estado do SMS, do agendamento ou da entrega.

Se um callback DLR/MO concluído pelo operador for reprocessado e falhar novamente, a nova atualização do job o apresenta como aberto e permite nova posse, preservando os eventos anteriores. Agendamentos e envios incertos não são reenviados nem reabertos automaticamente por esse mecanismo.

O link da ocorrência abre a listagem de origem com busca pelo ID. Para MO, abre o envio relacionado e seu histórico, pois o painel atual não oferece reprocessamento de MO. Para DLR, o reprocessamento já existente permanece na tela de callbacks e conserva suas próprias regras. A fila nunca reenvia SMS. A nota aceita até 1.000 caracteres; a interface orienta a não incluir telefone, texto, URL ou credenciais. Ela não aplica redação automática de conteúdo digitado pelo usuário.

Migração **027** cria `operation_cases`, `operation_case_events` e a view `operation_case_sources`. Aplicá-la antes do backend novo. A migração é aditiva e não envia, cancela ou altera mensagens/callbacks. Em rollback do binário, as tabelas podem permanecer sem uso; apagá-las descartaria o histórico humano e exige decisão explícita. O prazo de cinco minutos reutiliza `models.OperationScheduleGrace`, também empregado no alerta de agendamento. A view preserva apenas a condição de fila; o prazo é aplicado pela consulta Go.

O conjunto novo usa controllers, services, repositories, DTOs e models próprios. Autorização/sessão e escopo de empresa são transversais compartilhados. Sem novos serviços externos ou SMS real. Teste PostgreSQL isolado cobre escopo, posse, devolução, conclusão, histórico e bloqueio de escrita por outro operador.

## Publicação em HML — 31/10/2026

O commit `2bfac6b` entrou em `main`; os workflows de frontend e backend passaram. A tela `/pendencias` foi publicada, mas a listagem retornou HTTP 503 porque `operation_cases` e `operation_case_sources` ainda não existiam em `gateway21_hml`. A migração 026 estava registrada; a 027 não. A migração 027 foi aplicada no banco HML em transação com `pg_advisory_xact_lock(210014)`, guarda de banco/versões e registro em `migrations`. A consulta posterior confirmou tabela, view e versão 027. A tela passou a listar três envios incertos da 3035TECH; o detalhe abriu com o aviso de não reenviar sem investigação. Nenhum caso foi assumido ou alterado na validação.

A execução automática da migração no PreSync do GitOps precisa ser investigada antes da próxima alteração de schema. A publicação do binário novo, por si só, não provou a aplicação da migração.
