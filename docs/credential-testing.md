# Homologação de credenciais

O administrador pode usar **Testar credencial** depois de salvar uma credencial. As três verificações são distintas:

1. `GET /v1/auth/check` usa o mesmo Basic Auth, orçamento de autenticação e whitelist da API legada. Responde HTTP 204 após autenticar, sem enviar SMS ou consumir consultas. Deve ser chamado **da rede do cliente**. Uma chamada do painel/servidor não comprova o IP do cliente. Conferir qual IP o gateway vê atrás de proxy/NAT antes do deployment.
2. `POST /admin/credentials/{id}/test-send` requer sessão de administrador e confirmação na interface. O administrador informa um celular brasileiro com DDD e 11 dígitos. Por padrão, `TEST_SMS_DESTINATION` restringe o destino a um único número; vazio desabilita o teste. Em homologação, `TEST_SMS_ALLOW_ANY_DESTINATION=true` permite escolher qualquer celular válido. A interface oferece texto padrão com o nome da empresa ou texto livre de até 160 caracteres. O backend usa o token da empresa, limitador compartilhado e `SMSService.Send`, registrando a tentativa normal e um evento de auditoria `test_send_requested`. Uma chamada autorizada pode custar um SMS. Resultado incerto **não** é reenviado automaticamente.
3. O painel consulta a tentativa pelo `correlationId` e apresenta DLR somente se registrado. Aceite do white label não é entrega. O ingresso DLR do fornecedor continua dependente de contrato e homologação; esta tela não inventa um status final.

Não há endpoint de validação de token do white label confirmado na documentação disponível. A etapa 2 é portanto uma tentativa real opt-in. Testes automatizados usam provedor simulado; não enviar SMS real em CI. Não colocar número de homologação, tokens ou senhas no repositório. A lista de tentativas já existente mostra o histórico; não foi criada tabela nova.

Operação: habilitar uma das configurações de destino apenas no backend, guardar a senha gerada no cadastro, executar a etapa 1 da rede do cliente e só então usar a etapa 2. Em Compose, as variáveis são repassadas ao backend. Sem destino fixo e sem o opt-in para escolha livre, a rota administrativa fica indisponível.
