export type DemoRow = {
  id: string
  name: string
  detail: string
  scheduledAt: string
  company: string
  status: string
  date: string
}
export type Section =
  | 'empresas'
  | 'usuarios'
  | 'credenciais'
  | 'envios'
  | 'agendamentos'
  | 'callbacks'
  | 'auditoria'
export const sections: Record<
  Section,
  {
    title: string
    description: string
    entity?: string
    first: string
    second: string
  }
> = {
  empresas: {
    title: 'Empresas',
    description: 'Organize as empresas que utilizam a API de SMS.',
    entity: 'empresa',
    first: 'Empresa',
    second: 'Referência',
  },
  usuarios: {
    title: 'Usuários',
    description: 'Gerencie o acesso da equipe ao backoffice.',
    entity: 'usuário',
    first: 'Nome',
    second: 'E-mail',
  },
  credenciais: {
    title: 'Credenciais da API',
    description: 'Acessos externos para integração e envio de SMS.',
    entity: 'credencial',
    first: 'Usuário da API',
    second: 'Callback',
  },
  envios: {
    title: 'Envios',
    description: 'Consulte as tentativas de envio e o retorno do fornecedor.',
    first: 'ID do envio',
    second: 'Destinatário',
  },
  agendamentos: {
    title: 'Agendamentos',
    description:
      'Horários solicitados pelos clientes e andamento do despacho. O usuário da API identifica a credencial técnica.',
    first: 'ID do envio',
    second: 'Agendado para',
  },
  callbacks: {
    title: 'Callbacks',
    description: 'Acompanhe a entrega das notificações aos clientes.',
    first: 'ID do envio',
    second: 'Destino do callback',
  },
  auditoria: {
    title: 'Auditoria',
    description: 'Acompanhe as alterações realizadas pela equipe.',
    first: 'Responsável',
    second: 'Ação',
  },
}
const companies = [
  'Horizonte Saúde',
  'Vértice Logística',
  'Nexo Educação',
  'Aurora Varejo',
  'Portal Serviços',
  'Conecta Tecnologia',
]
export const initialRows: Record<Section, DemoRow[]> = Object.fromEntries(
  Object.keys(sections).map((key) => [
    key,
    Array.from(
      {
        length:
          key === 'envios'
            ? 64
            : key === 'callbacks'
              ? 32
              : key === 'agendamentos'
                ? 36
                : 14,
      },
      (_, i) => {
        const company = companies[i % companies.length]
        const status =
          key === 'envios'
            ? ['Aceito', 'Aceito', 'Rejeitado', 'Resultado incerto'][i % 4]
            : key === 'agendamentos'
              ? i >= 24
                ? 'Na fila'
                : [
                    'Na fila',
                    'Na fila',
                    'Reservado',
                    'Aceito pelo fornecedor',
                    'Resultado incerto',
                  ][i % 5]
              : key === 'callbacks'
                ? ['Entregue', 'Entregue', 'Pendente', 'Falhou'][i % 4]
                : key === 'auditoria'
                  ? 'Registrado'
                  : i % 5 === 4
                    ? 'Bloqueado'
                    : 'Ativo'
        const name =
          key === 'empresas'
            ? `${company}${i >= companies.length ? ` ${i + 1}` : ''}`
            : key === 'usuarios' || key === 'auditoria'
              ? [
                  'Marina Costa',
                  'Lucas Almeida',
                  'Beatriz Santos',
                  'Pedro Lima',
                ][i % 4] + (i > 3 ? ` ${i + 1}` : '')
              : key === 'credenciais'
                ? `integracao_${i + 1}`
                : `SMS-${String(10482 + i)}`
        const detail =
          key === 'empresas'
            ? `CLI-${String(i + 1).padStart(3, '0')}`
            : key === 'usuarios'
              ? `usuario${i + 1}@21mobile.example`
              : key === 'credenciais' || key === 'callbacks'
                ? `https://cliente${i + 1}.example/notificacoes`
                : key === 'agendamentos'
                  ? `${String(26 + Math.floor(i / 12)).padStart(2, '0')}/10/2026 ${String(8 + (i % 12)).padStart(2, '0')}:00 · integracao_${(i % 6) + 1}`
                  : key === 'auditoria'
                    ? [
                        'Empresa atualizada',
                        'Credencial bloqueada',
                        'Usuário criado',
                        'Senha da API alterada',
                      ][i % 4]
                    : '(11) 9••••-4321'
        return {
          id: `${key}-${i + 1}`,
          name,
          detail,
          scheduledAt:
            key === 'agendamentos'
              ? `2026-10-${String(26 + Math.floor(i / 12)).padStart(2, '0')}T${String(8 + (i % 12)).padStart(2, '0')}:00:00`
              : '',
          company,
          status,
          date:
            key === 'agendamentos'
              ? `${String(25 + Math.floor(i / 12)).padStart(2, '0')}/10/2026 ${String(9 + Math.floor(i / 12)).padStart(2, '0')}:${String(59 - (i % 60)).padStart(2, '0')}`
              : `14/10/2026 ${String(14 - Math.floor(i / 12)).padStart(2, '0')}:${String(59 - (i % 60)).padStart(2, '0')}`,
        }
      },
    ),
  ]),
) as Record<Section, DemoRow[]>
