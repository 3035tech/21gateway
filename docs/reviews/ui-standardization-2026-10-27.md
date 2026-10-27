# Auditoria de padronização da interface — 27/10/2026

## Escopo e método

Revisadas as telas reais de visão geral, empresas, usuários, credenciais, envios, agendamentos, callbacks e auditoria; `PanelShell`, `Listing`, `DataTable`, estilos e contratos de listagem no backend. Conferência visual no Chrome com API simulada em largura comum e largura de celular para dashboard/agendamentos. A prévia `/demo` é fictícia e não substitui homologação do painel real.

## Resultado por critério

| Critério | Resultado |
| --- | --- |
| Identidade visual | Roxo 21 Mobile, magenta de seleção, Raleway, ícones Lucide e superfícies claras consistentes. O dashboard concentra o destaque de cor e as demais páginas preservam leitura de dados. |
| Hierarquia e tipografia | Títulos de página 24/30 px; texto de apoio, filtros, tabela e ações em 14 px; avisos e metadados em 12 px. Botões e controles têm altura mínima de 44 px. |
| Alinhamento | Cabeçalhos, células, filtros e rodapé usam componentes compartilhados. Tabelas largas passam a indicar rolagem horizontal; a região pode receber foco para rolagem por teclado. Busca ocupa linha inteira no celular. |
| Filtros | Todas as sete listagens reais têm busca. Empresas, usuários e credenciais também têm status; credenciais, envios e callbacks têm empresa; agendamentos têm empresa, credencial, estado e período. A busca textual é aplicada por botão; os outros controles filtram ao alterar. |
| Paginação | Todas as sete usam paginação do servidor, total e opções de 10, 25, 50 ou 100 itens. Troca de filtro, busca, tamanho ou ordenação volta à primeira página. |
| Ordenação | Todos os cabeçalhos de dados das sete listagens enviam a coluna e direção ao servidor; apenas Ações não é ordenável. Backend usa lista fechada de expressões/colunas e desempate por ID. |
| Informação relevante | Envios e callbacks passam a mostrar nome do cliente quando disponível, com ID como referência alternativa. Em agendamentos, datas de solicitação e programação aparecem antes dos detalhes de destino. Mensagens e agendamentos não expõem texto SMS. |

## Riscos e próximos cuidados

- A pessoa física que usou uma credencial técnica de API não é identificada pelos dados atuais. A tela informa isso expressamente.
- Ordenações por nome da empresa, URL, campos derivados de credenciais e buscas por trecho podem exigir índices, filtros mais seletivos ou outra estratégia quando houver volume de produção. Medir com dados representativos antes de prometer tempo de resposta; não criar índices para todas as colunas sem medir custo de escrita.
- Auditoria passou a mostrar o nome atual do responsável junto ao ID e oferece filtros por responsável, ação e recurso. O nome não é um snapshot histórico: mudanças posteriores aparecem na leitura, enquanto o ID preserva a identidade da conta. Recursos continuam vinculados por ID; seus nomes históricos não são inferidos.
- A revisão visual usou dados simulados no navegador; não prova integração com fornecedor, carga real ou aparência de todos os estados de formulário em todos os navegadores.
