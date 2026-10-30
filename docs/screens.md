# Telas

> Escopo respondido em 14/10/2026: consulte docs/scope-answers.md. Este documento descreve o código existente; token por cliente, whitelist, MO e compatibilidade V1 completa ainda exigem ajustes. Painel solicitado pela 21 é focado em credenciais/configurações com perfil administrativo único. Não confundir recursos extras implementados com requisitos confirmados.
## Painel real

Na ficha da empresa, administradores podem abrir **Diagnosticar respostas Zoug** quando há token da empresa configurado. A ação dispara 16 casos fixos de autenticação, payload, lote, rota e método, sem destinatário válido. Uma opção desmarcada por padrão habilita o número de teste autorizado e acrescenta dois casos com destino válido e campos obrigatórios inválidos; há risco de até duas mensagens se o fornecedor ignorar os erros. Qualquer HTTP 2xx interrompe a bateria. A tela mostra status HTTP, headers, corpo e duração por caso, além do Run ID. O token salvo é usado no backend e nunca exibido. A execução é explícita e limitada a uma por minuto por empresa no processo. Resultados são transitórios na tela e registrados em JSON/stdout pelo Run ID, com token e número redigidos.

Feedback das ações reais usa aviso compartilhado de sucesso ou falha, com texto traduzido, anúncio acessível e fechamento automático. Cadastros, edição, arquivamento, reenvio de convite, callback e teste de SMS usam o mesmo padrão. No teste de SMS, aceite do fornecedor e resultado incerto são avisados separadamente da entrega final; detalhes e erro permanecem na tela da ação.

Login, visão geral, empresas, usuários internos, credenciais externas, envios, callbacks e auditoria conectados à API administrativa. Campos e permissões em docs/backoffice.md. Usuários/auditoria acessíveis somente a administradores; operação de cadastros/credenciais/callbacks disponível a operadores e administradores; consulta não permite alterações.

Empresas e usuários: formulários próprios, status ativo/bloqueado e confirmação de arquivamento. Usuários escolhem Consulta, Operador ou Administrador por opções com ícone e resumo das permissões, com Consulta como sugestão inicial. Administradores têm acesso global; para os demais, o formulário exige uma ou mais empresas. O convite por e-mail permite definir a própria senha, que o admin não digita no cadastro/edição. A prévia usa o mesmo seletor de perfil. Credenciais têm empresa, username, status e configuração de callback; criação/rotação exibe a senha uma vez. Arquivar preserva histórico.

O administrador pode abrir **Testar credencial** após salvar: instrução para checagem Basic Auth/IP da rede do cliente, campo de celular com máscara, mensagem padrão ou livre e consulta posterior de DLR. Aceite do fornecedor e entrega são exibidos separadamente. Detalhes e limites em [credential-testing.md](credential-testing.md).

Cadastro de credenciais: permite várias credenciais por empresa, username gerado ou informado, senha gerada, manual ou legada e validade opcional no fuso do navegador. A listagem mostra validade e o status vencido; a edição pode definir ou remover a validade. O usuário e o vínculo da credencial não mudam após criação. O painel oferece português e inglês no login e no cabeçalho, com preferência salva no navegador; textos de telas reais, validações e erros vêm dos catálogos em `frontend/src/locales/`.

O seletor de idioma exibe as bandeiras do Brasil e dos Estados Unidos ao lado do nome de cada opção; o nome acessível permanece no controle.

Na listagem de envios, a primeira coluna usa apenas “ID” para reservar espaço horizontal aos demais campos; o detalhe continua identificando explicitamente “ID do envio”.

O detalhe de Envios inclui histórico cronológico sob demanda: registro, início do processamento quando conhecido, resultado atual da submissão, eventos DLR/MO e estado das notificações ao cliente. A hora exata da resposta Zoug anterior à mudança não está disponível. A ficha da empresa mostra indicadores de integração e últimos horários; administradores veem avisos operacionais na Visão geral, atualizados a cada minuto. Esses avisos não enviam e-mail automaticamente.

O estado atual da submissão aparece separado dos eventos datados no detalhe do SMS. Sem eventos DLR/MO/notificações, a tela informa a ausência de registros. Avisos operacionais mostram o horário da leitura e abrem a listagem relacionada para investigação; são uma entrada para consulta, sem filtro ou resolução automática.

Credencial Zoug: a ficha da empresa exibe o estado e o cadastro/substituição/remoção do token fornecido pela Zoug. O gateway gera apenas usuário e senha Basic Auth das credenciais de API; não gera o token do fornecedor. Várias credenciais Basic Auth da mesma empresa compartilham seu token Zoug. A edição de credencial da API abre a empresa para gerenciar esse token. Tokens antigos individuais continuam em fallback temporário até a configuração do token da empresa; nenhum valor secreto é devolvido pelo painel.

O formulário de nova credencial e edição usa abas **Acesso**, **Notificações** e **IPs autorizados**. Na aba Acesso, busca de empresa precede os campos alinhados de empresa e usuário; a criação abre com usuário e senha Basic Auth manuais editáveis, enquanto geração automática e preservação legada são escolhas explícitas. O ID da conta legada fica em uma seção opcional recolhida. A senha é persistida apenas como hash Argon2id. Status e validade compartilham uma linha quando há espaço. A gestão do token Zoug é uma ação contextual que abre a ficha da empresa e fecha o modal; o token é cifrado no banco. O editor de IPs aceita entradas individuais e redes CIDR, normaliza e agrupa ambas as listas, e permite removê-las separadamente. Erros de validação levam à aba correspondente.

Na aba Notificações, DLR e MO ficam em blocos verticais para exibir URLs longas. Cada destino tem autenticação independente e opcional: nenhuma (padrão), cabeçalho personalizado ou Basic Auth. O segredo é campo de escrita; a tela indica quando já existe um valor salvo e permite substituí-lo sem revelá-lo.

Dentro de cada destino, nome e valor do cabeçalho ou usuário e senha Basic Auth ficam lado a lado em telas largas e empilhados no celular.

Polimento de 27/10/2026: a prévia ordena o formulário de credencial por empresa e acesso Basic Auth, permite gerar username e exibe a senha gerada uma vez em diálogo. O formulário real mostra a empresa fixa em campo somente leitura. Os links diretos de usuário e credencial apresentam estados de carregamento, erro e ID inválido. A prévia continua apenas em memória.

Listagens compartilham DataTable, filtros, paginação/tamanho e ordenação no servidor. Envios separam submissão/entrega e resultados incertos. Callbacks separam eventos recebidos da Zoug (SMS e MO, sem conteúdo bruto) da fila/falhas das notificações aos clientes; o reprocessamento manual não reenvia SMS. Auditoria não inclui senhas ou conteúdo SMS. Resumo usa dados reais e não inventa saldo ou contagens.

## Prévia fictícia

/demo no Vite de desenvolvimento permanece disponível sem backend para revisão visual. A prévia começa em um login simulado e inclui dashboard populado e listagens de empresas, usuários, credenciais, envios, agendamentos, callbacks e auditoria. Os exemplos são fictícios e as alterações em memória são descartadas ao recarregar. O botão Sair volta ao login simulado, sem credenciais reais. Nenhum SMS é enviado. Nunca declarar essas alterações como persistidas. Prévia não é habilitada em build de produção.

## Agendamentos e visão geral — 27/10/2026

O administrador interno abre agora em uma visão geral com contagens de agendamentos vencidos na fila, próximos em 24 horas, em processamento e total na fila; resultados históricos de envio; jobs DLR/MO pendentes ou falhos; empresas e credenciais ativas. Os números vêm de `/admin/summary` e são atualizados manualmente. Aceite do fornecedor não é entrega final, e resultado incerto não recomenda reenvio.

A tela `/agendamentos` lista registros reais com empresa, usuário técnico da API, destinatário, referência, solicitação, horário previsto, despacho e estados. Abre com os horários agendados mais recentes primeiro; a ordem pode ser alterada pelo cabeçalho. Tem busca, filtros de empresa/credencial/estado/período, paginação, tamanho de página e ordenação nas colunas suportadas. Os horários são exibidos no fuso do navegador. O SMS não é exposto. O sistema identifica a credencial que solicitou o envio, mas não a pessoa física que a utilizou. Consulta administrativa somente leitura; não altera a fila.

## Pendências

Portal de clientes, campanha/envio pelo painel, saldo e consulta MO no painel não implementados. Perfil, convite de usuários, recuperação por e-mail e MFA TOTP estão descritos em [admin-security.md](admin-security.md). O código TOTP tem seis campos numéricos, avanço automático e envio após o sexto dígito; códigos de recuperação usam campo separado. O agendamento interno existe com ativação opt-in; a nova tela oferece visibilidade sem operação de cancelamento/edição. DLR real do fornecedor depende de contrato. Sem PostgreSQL/API disponíveis, o painel real apresenta falha de conexão; não substituir por sucesso fictício.

Convites e recuperação de senha usam e-mails HTML com identidade 21 Mobile, ações próprias, prazo de 15 minutos e link alternativo visível. Cada envio inclui uma parte em texto simples no mesmo MIME; o token continua no fragmento do link e não é registrado em logs.

## Acabamento visual posterior

Painel real e prévia agora compartilham PanelShell: navegação compacta no celular, menu que fecha ao trocar de tela, indicação da seção atual e acesso ao perfil pelo nome/avatar no rodapé lateral, sem item duplicado no menu. Há foco para pular ao conteúdo. Raleway e cores da marca preservadas; tabelas têm espaçamento consistente, datas sem quebra, ações com alvo de toque e estados de carregamento/vazio/falha identificáveis. Status de entrega são apresentados em português.

Botões do frontend usam variantes próprias da marca para ação principal, secundária, perigosa, discreta, ordenação e barra lateral. Todos têm aparência removida do navegador, alvo de toque, foco visível e estados de hover/pressionado/desabilitado; a prévia compartilha as variantes do painel. O seletor de perfil usa cartões com rádio acessível oculto e indicador visual próprio.

Os selects nativos foram substituídos por `AppSelect` compartilhado no painel real e na prévia: filtros, paginação, idioma e formulários usam menu, seta, seleção e foco com visual da marca. O menu usa o contêiner do diálogo quando aberto dentro dele, para não ficar oculto atrás da camada modal. Checkboxes e rádios visíveis também têm marcação e foco próprios, preservando seus controles semânticos e interação por teclado.

O vínculo de empresas no formulário de usuário usa um seletor compartilhado no painel e na prévia: busca por nome ou referência, resultados em área rolável com páginas de 20 no painel (10 na prévia) e empresas já escolhidas em chips removíveis. A consulta real é paginada no servidor; a busca não fica limitada às primeiras 100 empresas. O perfil administrador mantém acesso global sem vínculo obrigatório.

A prévia `/demo` permite selecionar no login Marina (administradora global), Lucas (operador vinculado a Horizonte Saúde e Vértice Logística) ou Beatriz (consulta vinculada a Nexo Educação). A navegação, as listas, o resumo e os detalhes simulados usam o vínculo escolhido; consulta não exibe ações de edição, e usuários/auditoria não aparecem para perfis não administradores. Esse login continua fictício, sem autenticação real ou dados do backend.

Login tem marca visível no celular, opção de mostrar/ocultar senha e erros associados aos campos. Verificação de sessão e falha de conexão têm apresentação própria, com tentativa manual. Formulários reais não permitem fechar o diálogo enquanto a gravação está em andamento; exibição da senha de API espera o fechamento do formulário/confirmação, evitando diálogos sobrepostos.

Revisão no Chrome: prévia desktop e tela pequena, menu móvel e ausência de overflow da página verificados. Backend indisponível impede homologação visual de dados reais nesta sessão; testes frontend continuam verificando componentes reais. Nenhuma regra SMS, permissão ou contrato de backend alterado pelo polish.

## Configurações após o escopo respondido

Painel real inicia em credenciais, com navegação para empresas, usuários e acessos externos. Formulário da credencial permite token Zoug (somente entrada), status de configuração, substituição/remoção explícita, whitelist/IPs, destinos distintos DLR/MO e ID legado. Cadastro pode preservar senha conhecida de integração existente, sem gerar outra. MO tem fluxo interno durável preparado; recebimento real do fornecedor e consulta legada ainda pendentes. Prévia fictícia continua separada.

## Polish de credenciais e notificações

Formulário amplo/responsivo por seções: acesso do cliente, envio Zoug, notificações DLR/MO e restrição por IP. Senha legada aparece somente ao escolher preservar acesso; lista de IPs aparece com restrição habilitada e mantém valores quando alternada. Remoção de token limpa substituição digitada e explica efeito antes de salvar. Campos ficam bloqueados durante gravação; footer permanece visível na rolagem. Criação usa ação Criar credencial. Empresas não são consultadas novamente ao editar vínculo imutável.

Listagem distingue configuração DLR/MO e nomeia restrição por IP. Revisão do componente real em Chrome desktop/tela pequena com dados fictícios, sem API real: sem overflow horizontal, callbacks/IPs e footer alcançáveis por teclado. Screenshots em docs/reviews/credentials-polish-desktop.png e credentials-polish-mobile.png. Página temporária de revisão removida; produção não ganhou rota de dados fictícios.

## Auditoria de padronização — 27/10/2026

Critérios, matriz de listagens, ajustes e riscos em `docs/reviews/ui-standardization-2026-10-27.md`. As sete listagens reais usam filtro, paginação/tamanho e ordenação em todos os cabeçalhos de dados. Envios e callbacks exibem nome de cliente quando disponível; a busca ocupa a largura do celular e tabelas largas indicam rolagem. Ações não são campos ordenáveis.

Polish posterior: Auditoria mostra nome atual do responsável mais ID, distingue registros do sistema e oferece filtros por responsável, ação e recurso. Abre pelos eventos recentes. A consulta continua restrita ao administrador e sem senhas/conteúdo SMS. Tabelas com colunas ocultas pela largura do celular indicam rolagem horizontal.

## Visão da empresa — 27/10/2026

Contas não administradoras com dois ou mais vínculos exibem um seletor estilizado de empresa no cabeçalho. A empresa escolhida filtra visão geral, cadastros, credenciais, mensagens, agendamentos e callbacks; a prévia pode ser explorada com Lucas Almeida e duas empresas. O seletor não aparece para administrador global nem para usuário com vínculo único.
No cadastro de credenciais, operadores usam automaticamente a empresa em visualização; quando há só um vínculo, esse vínculo é assumido. O modal mostra o nome em campo somente leitura. A prévia inclui Pedro Lima, operador de uma empresa, para verificar esse caso.

A listagem de empresas abre `/empresas/{id}` pelo nome. A página mostra cadastro, estado, credenciais da API e usuários de operação/consulta vinculados, com criação e edição no próprio contexto; a seleção da empresa fica fixa. A listagem geral de credenciais e usuários permanece disponível. Administradores não possuem vínculo e, por isso, não entram na lista de usuários de uma empresa. Essa seção de usuários só aparece para administradores.

As listagens oferecem Visualizar para consultar os campos disponíveis sem entrar em edição. Empresas abrem a página de detalhes; usuários, credenciais, envios, agendamentos, callbacks e auditoria mostram detalhes em diálogo. Empresas, usuários e credenciais oferecem Arquivar para perfis autorizados, com confirmação e preservação do histórico. Registros de envio, agendamento e auditoria continuam somente leitura; callbacks falhos mantêm a operação de tentativa manual. A prévia fictícia também permite visualizar e arquivar cadastros em memória.

O cabeçalho e os controles da coluna Ações são centralizados pelo componente compartilhado de tabela, incluindo listas com uma ou várias ações.

Atualização de 30/10: as colunas Ações das listagens reais e da prévia usam botões de ícone de 44 px em uma linha, com dica nativa no hover, nome acessível e foco visível. Visualizar, testar credencial, editar, trocar senha, reenviar convite, tentar novamente e arquivar têm ícones distintos; arquivar mantém cor de perigo e confirmação. Ações principais de formulário, diálogo e recuperação de erro continuam com texto para preservar clareza.

Polish adicional de 30/10: o menu lateral do painel pode ser recolhido em desktop para uma faixa de 76 px, exibindo apenas ícones de navegação, perfil e saída. O controle de expansão fica na borda superior do menu, com alvo de 44 px e seta direcional em círculo discreto. Cada ação mantém nome acessível, foco visível e dica nativa; a preferência é lembrada no navegador. O menu móvel conserva a navegação expansível anterior. A área de trabalho usa até 1600 px quando o menu está recolhido. Botões compartilhados de busca, salvar e tentar novamente combinam ícone e texto; comandos principais permanecem explícitos.

Os nomes das credenciais e usuários na visão da empresa abrem URLs próprias (`/credenciais/{id}` e `/usuarios/{id}`), carregadas por GET administrativo do registro real e autorizadas pelo domínio. Ao fechar o detalhe, o usuário volta para a empresa de origem. A prévia fictícia usa o mesmo fluxo com registros simulados.

## Revisão visual com dados de homologação — 29/10/2026

Visão geral, empresas e detalhe, usuários, credenciais e detalhe, envios, agendamentos, callbacks recebidos/notificações, auditoria e perfil foram inspecionados no Chrome com dados de homologação. Envios, agendamentos e credenciais mantêm ações fixas na borda direita durante a rolagem horizontal; as ações de credenciais quebram em linhas compactas. A lista de credenciais dentro da empresa omite a coluna Empresa, redundante nesse contexto. O seletor de idioma mantém bandeira e nome na mesma linha no menu. Os campos de autenticação Basic Auth e cabeçalho personalizado dos destinos DLR/MO ficam lado a lado a partir da largura `sm` e empilham no celular.

O seletor de empresas autorizadas no formulário de usuário usa busca e paginação no servidor (20 resultados por página). A seleção permanece no formulário entre páginas. Para usuários vinculados a muitas empresas, somente cinco selecionadas aparecem por página e apenas esses cinco nomes são consultados; o total continua visível. Ao selecionar uma empresa, a página das selecionadas avança para mostrar o novo vínculo. A remoção permanece possível em qualquer página.

Perfil usa a largura do conteúdo administrativo: dados da conta ocupam uma faixa, e os cartões de senha e verificação em duas etapas ficam lado a lado em telas largas e empilhados nas demais. Campos preenchem a largura do respectivo cartão. Erros de validação aparecem somente quando presentes, sem reservar espaços vazios entre os campos.
# Notificações internas — 30/10/2026

Tela real `/notificacoes` exclusiva do administrador: histórico paginado/filtrado e configuração global de e-mail por tipo. Sininho no cabeçalho mostra não lidas e últimas ocorrências. Todos os tipos aparecem no painel; somente tipos ativados também geram e-mail para administradores ativos.
