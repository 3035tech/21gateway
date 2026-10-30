# Perguntas para fechamento do escopo

A 21 enviou respostas em 14/10/2026; consolidação e pendências em `docs/scope-answers.md`. As perguntas abaixo são o questionário original e devem ser preservadas como referência. Para cada funcionalidade: obrigatória agora, etapa futura ou dispensada; informar clientes que a utilizam. Respostas determinam se faremos envio básico, envio com acompanhamento ou compatibilidade completa.

## Objetivo e compatibilidade

1. Substituição completa ou apenas envio imediato nesta etapa?
2. Quais clientes serão migrados e quais recursos cada um usa?
3. Integrações precisam continuar sem alteração do cliente?
4. Preservar domínio, endpoints, campos, respostas e códigos?
5. Há exceções/comportamentos por cliente fora do manual?
6. Disponibilizar fontes PHP completos e exemplos reais sanitizados?

## Envio

7. Individual e lotes de até 200 são necessários?
8. Manter sucesso parcial com resultado individual?
9. correlationId deve ser preservado e pode se repetir?
10. Clientes dependem de id_sms e do seu formato numérico?
11. Agendamento scheduledDate é necessário e quem usa?
12. Suportar só 160 caracteres ou mensagens longas/acentos/emojis?
13. Preservar texto integral ou há normalização de caracteres?
14. Somente Brasil e quais formatos de telefone?
15. Regras de remetente, horário, bloqueio e duplicidade?

## Callbacks e respostas

16. DLR obrigatório nesta etapa?
17. MO obrigatório nesta etapa?
18. Preservar GET DLR e POST MO com os formatos atuais?
19. URL única ou separada por evento/cliente?
20. Autenticação exigida no endpoint do cliente?
21. Todos os estados ou só resultado final?
22. Prazo e política de repetição quando cliente indisponível?
23. Consulta/reprocessamento manual de falhas no painel?

## Consultas e créditos

24. Manter /v1/status e consumo dos não consultados?
25. Manter /v1/status/date com filtros/paginação?
26. Manter /v1/consultmo?
27. Manter /v1/credits?
28. Quem utiliza cada consulta?
29. Origem do saldo e unidade: SMS, segmentos ou dinheiro?
30. Histórico anterior e prazo de retenção?

## White label

31. Conta/token por cliente ou compartilhado?
32. Quem administra contas, tokens, rotas e saldo?
33. Callback em lote e exemplos completos de DLR/MO?
34. Autenticação, retries e ordenação dos callbacks do fornecedor?
35. Suporte/documentação para agendamento, status, MO e saldo?
36. Idempotência/consulta por referência para timeouts?
37. TPS por conta/token/rota e cobrança de mensagens longas?
38. Homologação, credenciais de teste e contato técnico?

## Painel e acessos

39. Só equipe 21 ou também clientes?
40. Perfis e permissões de cada um?
41. Ações sobre clientes e credenciais: cadastrar/editar/bloquear/excluir/trocar?
42. Manter credenciais antigas ou emitir novas?
43. Múltiplas credenciais por cliente?
44. Gestão de IPs, callbacks, tokens e limites no painel?
45. Consultas de mensagem/status/saldo/falhas ou só credenciais?
46. Auditoria e MFA necessários?

## Operação

47. Volume diário e pico por segundo?
48. Clientes simultâneos e tamanho de lote?
49. Manter uma requisição/segundo e quais exceções?
50. Latência máxima de resposta e callback?
51. Fornecedor indisponível: erro ou armazenamento para envio posterior?
52. Validade das mensagens por finalidade?
53. Responsável por alertas e falhas?
54. Hospedagem, infraestrutura, backups e manutenção?
55. Requisitos de segurança, privacidade e retenção?

## Migração e aceite

56. Prazo e data de desligamento da plataforma antiga?
57. Migração gradual ou simultânea?
58. Tratamento de agendados, envios em andamento e callbacks antigos?
59. Retorno à plataforma anterior por cliente?
60. Clientes piloto e responsável pela homologação?
61. Cenários obrigatórios para aceite?
62. Aprovador do escopo e funcionalidades futuras?

## Detalhes adicionais dos CRUDs indicados depois do questionário

Definir campos/validações de usuários e empresas; vínculo usuário-empresa; distinção entre administrador, cliente e credencial SMS; unicidade; permissões; exclusão/bloqueio; filtros disponíveis e ordenação padrão. As respostas recebidas estão em docs/scope-answers.md; algumas definições continuam pendentes.

## SMS para alertas críticos internos — proposta de 30/10/2026

- Destinatários: todos os super administradores ativos com telefone verificado ou uma lista operacional separada?
- Qual provedor/canal independente da rota Zoug de clientes fará o envio operacional?
- Quais tipos representam ação imediata, e quais ficam apenas em sininho/e-mail?
- Como verificar os telefones e tratar perda de acesso, plantão e atualização de números?
- Qual intervalo máximo de repetição, escalonamento e orçamento de envios?

Até essas decisões, nenhum SMS de alerta operacional deve ser ativado. Consultar docs/notifications.md.
