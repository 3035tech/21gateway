# Segurança da conta administrativa — 27/10/2026

## Escopo e fronteira

Há vários usuários do painel possíveis, inclusive mais de um administrador. A tela **Usuários** cadastra nome, e-mail, perfil (`admin`, `operator`, `viewer`) e status. Administradores têm acesso global; operadores e usuários de consulta exigem ao menos uma empresa vinculada e ficam limitados a elas. Credenciais Basic Auth das APIs de SMS são outro domínio e não usam cookies, 2FA ou “Lembrar de mim” do painel. Um futuro portal externo de clientes permanece fora do escopo atual.

## Convite, perfil e senha

O cadastro pela tela usa `POST /admin/users/invite`: gera uma senha aleatória inacessível, salva somente o hash Argon2id, cria token de convite de 256 bits e envia e-mail HTML 21 Mobile por SMTP com STARTTLS obrigatório. O usuário define a própria senha em `/resetar-senha`. O token vai no fragmento da URL e é removido da barra após o carregamento da página. O convite dura 24 horas; a recuperação de senha dura 15 minutos. Ambos são de uso único e um novo link invalida o anterior. O admin pode reenviar por `POST /admin/users/{id}/invite`, limitado a três por minuto por conta. A API administrativa antiga `POST /admin/users` mantém o contrato de senha explícita por compatibilidade, mas o painel não o usa.

`/perfil` mostra nome, e-mail e perfil. Trocar a própria senha exige a atual, aplica a política de 15 caracteres/128 bytes, audita o evento e incrementa `session_version`, revogando todas as sessões. O fluxo `/esqueci-senha` envia um link de recuperação com a mesma validade e uso único. A resposta é genérica para e-mails conhecidos e desconhecidos; há limites por IP e por e-mail. O e-mail de recuperação é enviado após a resposta, com concorrência limitada; em caso de falha, o usuário pode solicitar outro link. Redefinir não cria sessão automaticamente e também revoga as anteriores. Se 2FA estiver ativo, o link ainda exige um código do aplicativo ou de recuperação.

## Segundo fator

O administrador inicia com sua senha atual. O servidor gera segredo TOTP de 20 bytes com `pquerna/otp` e guarda apenas o segredo cifrado AES-GCM com `MFA_ENCRYPTION_KEY` (base64 de 32 bytes). A tela monta localmente um QR code `otpauth://` com `qrcode.react` e exibe chave manual. O segredo pendente expira em dez minutos. A ativação requer um TOTP válido e entrega dez códigos de recuperação aleatórios de 128 bits, mostrados uma vez; somente hashes SHA-256 são guardados. Código TOTP não pode ser repetido no mesmo intervalo de 30 segundos. Cada código de recuperação é consumido uma vez. A ativação e desativação renovam a sessão atual e revogam as demais. Desativar exige senha atual e segundo fator.

O login só cria sessão depois da senha e, quando configurado, do segundo fator. Tentativas de senha e fator são limitadas no PostgreSQL. Sem `MFA_ENCRYPTION_KEY`, novas ativações falham; uma conta já protegida não pode entrar se a chave necessária estiver ausente. Não perder/rotacionar a chave sem procedimento de recriptografia dos segredos existentes. Os códigos de recuperação devem ser guardados fora do painel; se e-mail, dispositivo e códigos forem perdidos, é necessário um processo operacional de identificação e recuperação, não um bypass automático.

O código do aplicativo é digitado em seis campos numéricos: cada dígito avança o foco, colar seis dígitos preenche os campos e o sexto envia o formulário automaticamente. Códigos de recuperação continuam em um campo separado.

## Sessões web

Sem a opção “Lembrar de mim”, a sessão tem prazo absoluto de oito horas, cookie de sessão e expira após 30 minutos de inatividade. Com a opção, o cookie persiste e o prazo absoluto é de 14 dias, com inatividade máxima de sete dias. O servidor impõe ambos os limites; o cookie é HttpOnly e, em produção, Secure. Mudanças de senha, perfil ou status revogam todas as sessões pela versão do usuário. Alterações de MFA renovam a sessão atual e revogam as demais. Estes prazos só se aplicam a `/admin`; os endpoints `/v1` de envio mantêm autenticação e limites independentes.

## Implantação e rollback

Aplicar migrações **015** (`user_mfa`) e **016** (`user_password_resets`) antes do binário novo. Configurar `MFA_ENCRYPTION_KEY`, `RECOVERY_SMTP_HOST`, `RECOVERY_SMTP_PORT`, `RECOVERY_SMTP_USER`, `RECOVERY_SMTP_PASSWORD`, `RECOVERY_SMTP_FROM` e `PUBLIC_APP_URL` HTTPS no backend. A conta SMTP pode usar credenciais SMTP do SES. Não registrar tokens, segredo TOTP, QR, senhas ou URLs de recuperação. O frontend é estático e não recebe segredos SMTP/MFA.

As migrações são expansivas e não alteram usuários existentes. Durante a transição, não habilitar 2FA enquanto pods antigos capazes de fazer login sem segundo fator estiverem atendendo. Para rollback, drenar binários novos, impedir logins de contas que ativaram 2FA e manter as tabelas/chave; um binário antigo não impõe MFA. Não remover colunas/tabelas/chave automaticamente. SMTP real e entrega de e-mail precisam de homologação com a conta remetente e DNS de produção; testes locais usam mailer falso e não enviam e-mails externos.

Referências de implementação: [OWASP MFA](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html), [OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html), [pquerna/otp](https://github.com/pquerna/otp), [SCS](https://pkg.go.dev/github.com/alexedwards/scs/v2), [qrcode.react](https://github.com/zpao/qrcode.react).
