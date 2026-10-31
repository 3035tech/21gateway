# Revisão de robustez — 31/10/2026

## Escopo e correções

Revisão estática de entrada HTTP, consultas de maior volume, dependências, carregamento do painel e componentes compartilhados. Nenhum SMS, callback ou e-mail real foi enviado. O defeito de escala em Credenciais foi reproduzido em emulação móvel: uma tela de 390 px adotava largura de layout de 1060 px. O texto acessível dos marcadores DLR/MO, com posição absoluta, escapava da tabela rolável. O marcador agora é seu contexto de posicionamento; após publicação em HML, largura de layout e visual ficaram em 390 px, escala 1; em 320 px, ambas ficaram em 320 px.

- `follow-redirects` foi atualizado no lockfile de 1.16.1 para 1.16.2. O alerta [GHSA-mrwp-f3q8-3g9p](https://github.com/advisories/GHSA-mrwp-f3q8-3g9p) afeta as versões anteriores. O aviso informa que a combinação exata com Axios 1.20.0 não reproduziu a falha; a atualização remove o alerta conhecido da árvore de produção.
- O workflow do frontend executa `npm audit --omit=dev --audit-level=high` após `npm ci`, antes de publicar a imagem.
- O workflow do backend executa `govulncheck` v1.9.0 após testes com PostgreSQL real, `go vet` e build, antes de publicar a imagem.
- O projeto já reutiliza `DataTable`, `PageTabs`, `FormFooter`, cliente HTTP, paginação/erros e camadas por domínio. Nenhuma extração adicional apresentou ganho claro que compensasse o risco de mudar contratos de envio, callback ou autenticação nesta revisão.

## Verificações de segurança

- `npm audit --omit=dev --audit-level=high`: zero vulnerabilidades conhecidas na árvore de produção após a atualização.
- Auditoria completa do npm: 12 avisos moderados encadeados pelo `minimatch` 10.2.6 em ferramentas de desenvolvimento (ESLint e TypeScript ESLint). A correção automática sugerida exige mudança incompatível; não foi aplicada sem validar o novo conjunto de ferramentas. Não são dependências do bundle de produção.
- `govulncheck ./...`: nenhuma vulnerabilidade em funções chamadas. Há um aviso de módulo `GO-2026-5932` sobre `golang.org/x/crypto/openpgp`, pacote não importado pelo projeto. O resultado não substitui pentest, revisão de infraestrutura ou testes dinâmicos.

## Desempenho e limites ainda abertos

- O resumo administrativo em `repositories/summary.go` conta registros históricos de `sms_attempts`, `company_test_sends` e callbacks. O número de consultas é pequeno, mas o trabalho de contagem pode crescer com o histórico. Antes de substituir por agregados incrementais ou materializados, medir planos SQL e latência com volume representativo, índices efetivos e concorrência; essa mudança exigiria migração e reconciliação dos contadores.
- O frontend já carrega domínios por rota e pagina listas no servidor. A revisão de código não demonstrou um gargalo de renderização que justifique adicionar estado global, cache próprio ou nova abstração.
- A suíte PostgreSQL do perfil de verificação (`docker compose --profile verification run --build --rm backend-test`) passou com detector de corrida nesta revisão; ela exercita persistência e isolamento, sem substituir teste de carga.
- Não foram medidos TPS, latência em carga integrada, Safari físico, falhas de rede prolongadas nem entrega real do fornecedor. A aplicação não pode ser declarada “inquebrável” com base nestas verificações.
