# Revisão responsiva — 30/10/2026

## Escopo verificado

Inspeção visual e de largura do DOM no painel de homologação, com sessão administrativa, em larguras CSS de 320, 390 e 768 px. Foram abertos login, visão geral da prévia, empresas e detalhe da empresa, usuários, credenciais e formulário, envios, agendamentos, callbacks, auditoria, notificações (incidentes e alertas por email) e perfil. A prévia local também foi percorrida nas seções principais. As tabelas foram avaliadas com sua rolagem própria, sem enviar SMS nem alterar dados.

## Resultado

- Menu, cabeçalho, filtros, cartões de notificações e formulários se reorganizam nas larguras inspecionadas. A coluna de ações permanece disponível nas tabelas operacionais.
- Listas extensas usam rolagem horizontal dentro da tabela. Em Credenciais, o documento inteiro alcançava 1059 px em um viewport de 390 px, permitindo que a página saísse lateralmente da tela. Um primeiro ajuste no contêiner do painel não impediu o deslize ao ser conferido em HML; a correção foi movida para o documento, preservando a rolagem da tabela.
- As abas do formulário de credencial espremiam o rótulo “Notifications” no celular. Os ícones decorativos e o padding horizontal das abas ficam ocultos abaixo de `sm`, dando espaço aos nomes inclusive em 320 px.
- Em 390 px, notificações e suas preferências de email ficaram dentro do viewport; em 768 px, a lista de usuários e as notificações também ficaram dentro do viewport. Em 320 px, a lista de usuários e seus filtros permaneceram legíveis.

## Limites da verificação

Esta revisão cobre apresentação e navegação em viewports simulados. Não equivale a teste físico em iOS/Android, a ensaio de todas as combinações de idioma e dados, nem valida envio SMS ou integrações externas. O ajuste de Credenciais deve ser conferido em homologação após a publicação do frontend.
