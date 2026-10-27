# Plano de migração e aceite

Atualizado em 15/10/2026 a partir do questionário final. Este é um roteiro documental; não autoriza corte de tráfego, descarte, disparos reais ou deployment.

## Decisões confirmadas

- Operação indicada: 3035 e Mozhtech. A resposta não distribui hospedagem, manutenção e backups entre as equipes, nem escolhe região/recursos AWS. Amplify/AWS continuam sendo a direção já autorizada no projeto.
- Prazo desejado: conclusão em outubro. Outubro de 2026 é inferido do contexto; dia e data obrigatória de desligamento do legado não informados.
- Perguntas 57 e 60 lidas em conjunto: teste com cliente interno, primeiro cliente de produção e, após validação, demais clientes em conjunto. Infinitech e grupoamaro são candidatos ao piloto, sem escolha definitiva.
- Aceite funcional: enviar SMS; receber status do fornecedor e entregar ao cliente; receber resposta MO do fornecedor e entregar ao cliente. A 21 deseja uma entrega completa, preservando as integrações V1.
- Sem requisitos adicionais informados na pergunta 55. Isso não revoga segurança, privacidade ou retenção MO de 90 dias aprovada pelo usuário. Retenção de metadados, backups e histórico importado não foi fixada por essa decisão.

## Sequência proposta para execução

1. Fechar contratos externos e importação: payload/autenticação DLR/MO, saldo, paginação status/date e aceite do agendamento local, normalização telefônica, credenciais e relação Master/Sub.
2. Validar migrations 001–009 e fluxos reais em PostgreSQL; executar backup criptografado e restauração em banco isolado. Os testes com mocks existentes não comprovam estes passos.
3. Preparar conta interna, destinos controlados e evidências sanitizadas dos três fluxos completos. Envio real exige escopo de homologação definido; não duplicar SMS para comparar plataformas.
4. Escolher piloto de produção, aprovador, janela, critérios de interrupção e tratamento dos eventos antigos. Preparar retorno antes de alterar roteamento.
5. Após aceite interno e do piloto, migrar as demais contas na janela acordada, preservando credenciais, domínio e contrato V1. Acompanhar falhas de envio, jobs DLR/MO, eventos sem vínculo e limites.

Não há período mínimo de observação, limites numéricos de aceite ou janela aprovados; definir com a 21 antes do corte.

## Evidências de aceite

| Cenário | Evidência necessária | Estado atual |
| --- | --- | --- |
| SMS unitário/lote | Requisição V1, resposta por item/ID e recebimento controlado, sem duplicação | Adaptador implementado; homologação externa pendente |
| DLR completo | Evento autenticado do fornecedor, vínculo com envio, tradução e GET no destino correto | Ingresso normalizado/outbox/worker implementados; contrato real pendente |
| MO completo | Resposta controlada, evento do fornecedor, vínculo e POST legado no destino correto | Ingresso normalizado/outbox/worker implementados; contrato real pendente |
| Compatibilidade V1 | Credenciais existentes, whitelist, IDs/erros, consultas, saldo e agendamento conforme uso/contrato | Parcial; status/date, saldo e agendamento aguardam contratos |
| Isolamento e recuperação | Eventos da conta correta, repetição sem entrega indevida, indisponibilidade do destino, backup/restauração | Testes locais existentes; prova PostgreSQL e operação real pendentes |

Aceite técnico adicional preserva o escopo V1 já confirmado; não transforma telas extras do backoffice em requisito. Aprovação final deve identificar pessoa responsável e evidências, sem conteúdo ou segredos em logs/documentos.

## Decisões abertas antes do corte

- Escolher piloto e aprovador; confirmar data/janela e divisão de responsabilidades entre 3035/Mozhtech.
- Definir destino de agendamentos, envios em andamento e callbacks antigos. A sugestão de simplesmente virar a chave é provisória, não autorização de perda ou descarte. Identificar volume/estados e decidir drenagem, convivência ou migração.
- Definir retorno: quem altera roteamento, gatilhos e tratamento dos envios já aceitos pela nova plataforma. Voltar o domínio não devolve automaticamente eventos ou mensagens ao legado; nunca reenviar SMS de resultado desconhecido.
- Obter contratos e exemplos faltantes do fornecedor/legado e dados de importação por canal seguro. Política de callback por 24h continua proposta; configuração atual não equivale a esse prazo.

Nenhuma migração, homologação real ou contato externo foi executado nesta atualização.

Validação de aceite ampliada em 15/10/2026: consultar `docs/acceptance-validation.md` para cobertura executada, suíte PostgreSQL pendente e regra de bloqueio de callbacks DLR/MO após troca da URL. Mantida separação por domínio; sem camadas ou dependências novas.
