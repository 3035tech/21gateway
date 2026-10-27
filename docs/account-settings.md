# Configurações da conta de API

Migração 004 adiciona token Zoug criptografado, whitelist de IPs, configuração MO e ID legado. Os campos pertencem ao domínio de credenciais; controllers/services/models/DTOs/repositories permanecem separados.

PROVIDER_TOKEN_KEY é base64 de 32 bytes aleatórios, fornecida pelo ambiente e separada do banco. Gerar com ferramenta padrão como openssl rand -base64 32 e guardar no gerenciador de segredos ou .env local ignorado pelo Git. Não armazenar a chave nem tokens em VITE_* ou documentos versionados. Sem chave/token válido, o envio falha fechado. Manter a chave protegida junto ao plano de recuperação: backup do banco sozinho não recupera tokens. Rotação de chave requer procedimento de recriptografia ainda não implementado.

Tokens usam AES-GCM da biblioteca padrão Go, nonce aleatório e username imutável como dados autenticados. Cada requisição tem visão própria do cliente HTTP com token da conta, compartilhando o pool sem mutar headers globais. O frontend recebe somente providerTokenConfigured. Token omitido mantém o atual; clearProviderToken remove explicitamente; substituir e remover na mesma chamada é inválido. Tokens/passwords de entrada são writeOnly no OpenAPI.

Whitelist aceita até cem IPs/CIDRs IPv4/IPv6, normalizados/deduplicados. Habilitar exige ao menos uma entrada. Verifica RemoteAddr da conexão; não confia em X-Forwarded-For. Uma implantação atrás de proxy precisa política de proxies confiáveis antes de importar whitelists reais. Contas preexistentes ficam com whitelist desabilitada após migração, evitando alterar acesso sem importar a configuração antiga.

callbackUrl/callbacksEnabled continuam DLR. moCallbackUrl/moCallbacksEnabled armazenam destino de respostas MO; não habilitam por si só recebimento/entrega de MO, que continua dependente do contrato do fornecedor. URLs exigem HTTPS/443 sem credenciais/fragmento. Não copiar automaticamente um destino DLR para MO.

legacyAccountId aceita IDs como 2340 e 2340_779 e é único entre credenciais não arquivadas. Uma vez preenchido, não muda por edição. Não inferir token/herança de Master/Sub nem impor unicidade por empresa enquanto a identificação de cliente/subconta estiver pendente.

Cadastro novo pode gerar senha externa ou preservar senha legada conhecida, via legacyPassword. Importação preservadora armazena Argon2id da senha informada e não retorna outra senha. Atualização de uma conta não aceita legacyPassword; rotação continua explícita e voluntária. Isso é uma ferramenta para senha conhecida, não importação comprovada de exports/hashes antigos. Nenhuma conta real das 26 foi importada.

Painel real inicia em /credenciais e navegação foca empresas, usuários internos e credenciais. Telas operacionais existentes continuam acessíveis por suas rotas para compatibilidade/revisão, mas não são itens principais do produto. Prévia fictícia /demo continua exibindo o conjunto anterior. Novos usuários internos têm admin como perfil inicial; suporte anterior a outros perfis não foi removido silenciosamente.
