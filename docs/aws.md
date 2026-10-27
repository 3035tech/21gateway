# AWS: inventário e implantação

## Autorização e evidências

Em 14/10/2026, o usuário confirmou acesso à AWS e autorizou seguir com o trabalho. Identidade verificada por STS na conta 669241375557; região configurada us-west-2. Nenhum recurso foi criado ou alterado nesta inspeção.

Recursos encontrados nessa região:

| Serviço | Recurso | Estado observado |
| --- | --- | --- |
| EC2 | redmine-management-ec2 | running, t2.micro |
| EC2 | 30pay-api-prod | running, t3.small |
| ECS | 30grow-prod | cluster existente |
| RDS | general | PostgreSQL 18.3, db.t3.micro, available |

O RDS observado tem armazenamento criptografado, backup automático de sete dias, acesso público habilitado e não é Multi-AZ. Isso não confirma capacidade disponível, autorização de reutilização, acesso SQL ou isolamento adequado para a 21 Mobile.

A consulta `aws amplify list-apps` falhou por conectividade com o endpoint; existência de apps e permissões do Amplify não foram verificadas. Não declarar publicação concluída.

## Decisão pendente

Definir ambiente separado para a 21 Mobile ou indicar expressamente o servidor/banco existentes que podem ser reutilizados. Foi solicitada essa escolha ao usuário. Os nomes dos recursos indicam outros projetos; não assumir que podem receber aplicações, alterações de rede, schema ou carga da 21 Mobile.

Antes da implantação, registrar destino escolhido, domínio HTTPS da API, origem do frontend, método de distribuição do código e configuração de segredos. O frontend já possui amplify.yml; publicação depende de API funcional e VITE_API_URL válida.

## Separação de acessos

Usuários internos do backoffice terão login e permissões próprios. Credenciais externas da API serão vinculadas a empresas e permanecerão distintas das contas de sistema. Credenciais AWS, token do fornecedor e senhas de banco nunca devem ser incluídos em VITE_* ou no repositório.

## Opções para scheduler e alto volume — 15/10/2026

Usuário perguntou sobre usar serviços AWS durante a implementação do scheduler local. Não é decisão de substituir a fila nem autorização específica de criar recursos. Nenhum recurso AWS foi criado nesta etapa; inventário acima é histórico de 14/10, não consulta atual. Na avaliação inicial, o código usava PostgreSQL e worker Go, sem SDK SQS/EventBridge; a implementação SQS posterior está registrada no final deste documento.

Proposta inicial: Amplify para frontend, PostgreSQL em RDS dedicado/isolado conforme decisão de ambiente, API e worker como serviços ECS (Fargate ou EC2 a dimensionar), logs/métricas/alertas CloudWatch e segredos via mecanismo AWS com IAM mínimo. Não reutilizar os recursos de outros projetos sem escolha explícita. Não há orçamento calculado; custo depende de região, CPU/memória, ALB/NAT, banco, disponibilidade, logs e volume.

- [ECS/Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html): candidato a hospedar containers de API e worker contínuo. Isso preserva a consulta do scheduler a cada segundo; não requer cron externo por SMS.
- [CloudWatch](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/WhatIsCloudWatch.html): candidato para centralizar logs já em JSON e observar filas/atraso/desconhecidos. Coleta/alarmes ainda não configurados.
- [EventBridge Scheduler](https://docs.aws.amazon.com/scheduler/latest/UserGuide/schedule-types.html): alternativa para disparos de lotes/minutos, sem um recurso por SMS. Precisão documentada de 60 segundos; não substitui diretamente o loop de um segundo se essa latência for necessária. Não recomendar como melhoria garantida sem fechar tolerância de atraso e medir carga.
- [SQS](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-delay-queues.html): candidato para desacoplar despacho/callbacks após prova de necessidade. Delay próprio é limitado a 15 minutos; não resolve sozinho agendamentos para dias futuros. EventBridge pode disparar para SQS, mantendo IDs/estado no banco.
- [SQS Standard pode entregar novamente](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/standard-queues-at-least-once-delivery.html). Fila gerenciada não garante SMS exatamente uma vez; preservar guardas de estado, leases e reconciliação de resultado incerto. Retry de sinal interno não autoriza repetir chamada já iniciada ao fornecedor.

A sugestão preliminar de EventBridge para lotes exige a ressalva de precisão por minuto. Para o comportamento atual, a recomendação de partida é worker contínuo gerenciado pelo ECS + RDS + observabilidade. Essa era a proposta inicial; a implementação SQS posterior está registrada abaixo, ainda sem deployment. Comparar custo/carga/falhas de ambas antes de migrar a fila, mantendo o contrato V1.

## Scheduler Go e SQS — implementação de 15/10/2026

Decisão posterior autorizada e implementada: scheduler Go → outbox PostgreSQL → SQS Standard → consumidores limitados. Consulte [sqs-dispatch.md](sqs-dispatch.md) para fluxo, segurança de concorrência, configuração e recuperação. SDK oficial AWS v2, referências somente com UUID, um lote aberto por conta, reservas parciais e orçamento compartilhado; envios imediatos seguem síncronos. Migrações até 013 obrigatórias nos novos binários; SQS opt-in desligado. Nenhum recurso AWS criado. PostgreSQL local validado em 27/10/2026; SQS real/carga ainda pendentes. Testes simulados não homologam produção. Esta decisão substitui o estado anterior que descrevia SQS apenas como opção.

## Preparação GitOps para Dublin — 27/10/2026

O usuário escolheu preparar frontend e backend no GitOps do cluster Dublin com as Applications `21gateway-front` e `21gateway-back`. O backend usa um único Deployment/Service e um único contêiner: o processo `/api` executa HTTP, callbacks, retenção e, quando habilitado, MO e scheduler Go/SQS. Migrações versionadas são um Job PreSync.

Os manifests ficam em `../../3035/gitops/dublin/21gateway-front` e `../../3035/gitops/dublin/21gateway-back`, com ingress Cloudflare Tunnel e pull secret GHCR já utilizado pelo cluster. Os nomes públicos `21gateway-front.3035service.com` e `21gateway-back.3035service.com` e as imagens GHCR 0.1.0 são parâmetros de preparação, ainda sujeitos à confirmação/publicação. O frontend precisa ser construído com `VITE_API_URL` apontando ao backend; não basta variável de runtime no pod Nginx.

Nenhuma senha ou token de outro projeto foi copiado. O backend referencia Secret próprio `21gateway-back-runtime`, ainda a provisionar fora do GitOps, com `DATABASE_URL` de banco/usuário próprios e chaves independentes. Agendamento/SQS e ingress DLR/MO seguem desabilitados até seus contratos, chaves, permissões e testes serem fechados. Configuração GitOps não equivale a deployment nem homologação; não houve criação de imagem, banco, Secret, DNS ou sync ArgoCD nesta etapa.
