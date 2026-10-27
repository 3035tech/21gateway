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
    title: 'nav.companies',
    description: 'demo.sections.companies',
    entity: 'companies',
    first: 'common.company',
    second: 'common.reference',
  },
  usuarios: {
    title: 'nav.users',
    description: 'demo.sections.users',
    entity: 'users',
    first: 'common.name',
    second: 'common.email',
  },
  credenciais: {
    title: 'nav.credentials',
    description: 'demo.sections.credentials',
    entity: 'credentials',
    first: 'common.apiUsername',
    second: 'demo.callback',
  },
  envios: {
    title: 'nav.messages',
    description: 'demo.sections.messages',
    first: 'common.messageId',
    second: 'common.destination',
  },
  agendamentos: {
    title: 'nav.schedules',
    description: 'demo.sections.schedules',
    first: 'common.messageId',
    second: 'schedules.scheduledFor',
  },
  callbacks: {
    title: 'nav.callbacks',
    description: 'demo.sections.callbacks',
    first: 'common.messageId',
    second: 'demo.callbackDestination',
  },
  auditoria: {
    title: 'nav.audit',
    description: 'demo.sections.audit',
    first: 'audit.actor',
    second: 'audit.action',
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
            ? ['accepted', 'accepted', 'rejected', 'unknown'][i % 4]
            : key === 'agendamentos'
              ? i >= 24
                ? 'queued'
                : ['queued', 'queued', 'claimed', 'done', 'unknown'][i % 5]
              : key === 'callbacks'
                ? ['delivered', 'delivered', 'pending', 'failed'][i % 4]
                : key === 'auditoria'
                  ? 'recorded'
                  : i % 5 === 4
                    ? 'blocked'
                    : 'active'
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
                        'companyUpdated',
                        'credentialBlocked',
                        'userCreated',
                        'apiPasswordChanged',
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
          company:
            key === 'usuarios'
              ? ['admin', 'operator', 'viewer'][i % 3]
              : company,
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
