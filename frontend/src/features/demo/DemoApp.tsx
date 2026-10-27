import { PanelShell } from '../../components/admin/PanelShell'
import { useState } from 'react'
import { Route, Routes, Link, Navigate } from 'react-router-dom'
import {
  Activity,
  Plus,
  Pencil,
  ShieldCheck,
  ArrowRight,
  CalendarClock,
  Send,
  Webhook,
  Building2,
  KeyRound,
  CircleUserRound,
} from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { initialRows, sections } from './data'
import { roleDescriptions } from '../../lib/admin'
import type { DemoRow, Section } from './data'

function Badge({ value }: { value: string }) {
  const tone = ['Ativo', 'Aceito', 'Entregue'].includes(value)
    ? 'bg-emerald-50 text-emerald-800'
    : ['Falhou', 'Rejeitado', 'Bloqueado'].includes(value)
      ? 'bg-rose-50 text-rose-800'
      : 'bg-amber-50 text-amber-800'
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ${tone}`}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {value}
    </span>
  )
}
function Overview() {
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Visão geral</h1>
          <p className="mt-2 text-sm text-slate-500">
            Acompanhe a fila e encontre o que precisa de atenção.
          </p>
        </div>
        <span className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
          <Activity size={16} /> Operação simulada
        </span>
      </div>
      <section className="overflow-hidden rounded-2xl bg-brand-purple text-white">
        <div className="grid lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
          <div className="p-6 sm:p-8">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-purple-100">
              <CalendarClock size={19} /> Fila de agendamentos
            </h2>
            <div className="mt-5 flex flex-wrap items-end gap-5">
              <strong className="text-6xl font-bold leading-none tabular-nums tracking-tight sm:text-7xl">
                8
              </strong>
              <p className="max-w-56 pb-1 text-sm leading-6 text-purple-100">
                aguardando despacho após o horário previsto
              </p>
            </div>
            <Link
              to="/agendamentos"
              className="mt-7 inline-flex min-h-11 items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-bold text-brand-purple"
            >
              Ver agendamentos <ArrowRight size={17} />
            </Link>
          </div>
          <div className="grid grid-cols-3 border-t border-white/20 bg-white/10 lg:grid-cols-1 lg:border-l lg:border-t-0">
            {(
              [
                ['Próximas 24 h', '12'],
                ['Em processamento', '5'],
                ['Total na fila', '27'],
              ] as const
            ).map(([label, value]) => (
              <div
                key={label}
                className="flex flex-col justify-center border-r border-white/15 p-4 last:border-r-0 lg:border-b lg:border-r-0 lg:px-7 lg:py-5 lg:last:border-b-0"
              >
                <span className="text-xs leading-5 text-purple-100 sm:text-sm">
                  {label}
                </span>
                <strong className="mt-1 text-2xl font-bold tabular-nums sm:text-3xl">
                  {value}
                </strong>
              </div>
            ))}
          </div>
        </div>
      </section>
      <div className="mt-6 grid gap-5 lg:grid-cols-2">
        <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
          <div className="flex items-center justify-between gap-3">
            <h2 className="flex items-center gap-2 text-lg font-bold">
              <Send size={20} className="text-brand-purple" /> Envios
              registrados
            </h2>
            <Link to="/envios" className="text-sm font-bold text-brand-purple">
              Abrir <ArrowRight size={15} className="inline" />
            </Link>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            Acumulado do histórico mantido no banco
          </p>
          <dl className="mt-6 grid grid-cols-3 gap-3 border-t border-slate-100 pt-6">
            {(
              [
                ['Aceitos', '32'],
                ['Rejeitados', '16'],
                ['Incerto', '16'],
              ] as const
            ).map(([label, value]) => (
              <div key={label}>
                <dt className="text-xs text-slate-500 sm:text-sm">{label}</dt>
                <dd className="mt-1 text-2xl font-bold tabular-nums">
                  {value}
                </dd>
              </div>
            ))}
          </dl>
          <p className="mt-6 rounded-lg bg-amber-50 px-4 py-3 text-xs leading-5 text-amber-900">
            Aceite pelo fornecedor não confirma entrega. Resultado incerto exige
            reconciliação antes de qualquer reenvio.
          </p>
        </section>
        <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
          <div className="flex items-center justify-between gap-3">
            <h2 className="flex items-center gap-2 text-lg font-bold">
              <Webhook size={20} className="text-brand-purple" /> Notificações
              aos clientes
            </h2>
            <Link
              to="/callbacks"
              className="text-sm font-bold text-brand-purple"
            >
              Ver DLR <ArrowRight size={15} className="inline" />
            </Link>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            Jobs pendentes e falhos de DLR e MO
          </p>
          <div className="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-6">
            <div>
              <h3 className="text-sm font-bold">Status DLR</h3>
              <p className="mt-2 text-2xl font-bold">
                8{' '}
                <span className="text-xs font-normal text-slate-500">
                  pendentes
                </span>
              </p>
              <p className="mt-1 text-sm text-rose-700">2 com falha</p>
            </div>
            <div>
              <h3 className="text-sm font-bold">Respostas MO</h3>
              <p className="mt-2 text-2xl font-bold">
                4{' '}
                <span className="text-xs font-normal text-slate-500">
                  pendentes
                </span>
              </p>
              <p className="mt-1 text-sm text-rose-700">1 com falha</p>
            </div>
          </div>
        </section>
      </div>
      <section className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-3 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-sm">
        <span className="font-bold">Acessos ativos</span>
        <span className="inline-flex items-center gap-2">
          <Building2 size={17} className="text-brand-purple" /> 12 empresas
        </span>
        <span className="inline-flex items-center gap-2">
          <KeyRound size={17} className="text-brand-purple" /> 11 credenciais
        </span>
        <Link to="/credenciais" className="ml-auto font-bold text-brand-purple">
          Gerenciar acessos <ArrowRight size={15} className="inline" />
        </Link>
      </section>
      <div className="mt-6 grid gap-6 xl:grid-cols-[1.6fr_1fr]">
        <section className="rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-100 p-5">
            <h2 className="font-bold">Últimos envios</h2>
            <Link
              to="/envios"
              className="flex items-center gap-1 text-sm font-semibold text-brand-purple"
            >
              Ver todos <ArrowRight size={15} />
            </Link>
          </div>
          {initialRows.envios.slice(0, 5).map((row) => (
            <div
              key={row.id}
              className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-4 last:border-0"
            >
              <div>
                <p className="text-sm font-semibold">{row.company}</p>
                <p className="mt-1 text-xs text-slate-500">
                  {row.name} · {row.date}
                </p>
              </div>
              <Badge value={row.status} />
            </div>
          ))}
        </section>
        <section className="rounded-xl border border-slate-200 bg-white p-6">
          <ShieldCheck className="mb-4 text-brand-purple" size={26} />
          <h2 className="text-lg font-bold">Acessos separados</h2>
          <p className="mt-3 text-sm leading-6 text-slate-500">
            Sua equipe entra no backoffice com usuários de sistema. Seus
            clientes integram com credenciais próprias da API.
          </p>
          <div className="mt-6 space-y-3">
            <Link className="button-secondary w-full" to="/usuarios">
              Gerenciar equipe
            </Link>
            <Link className="button-primary w-full" to="/credenciais">
              Gerenciar credenciais
            </Link>
          </div>
        </section>
      </div>
      <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-5">
        <h2 className="font-semibold text-amber-900">Atenção na operação</h2>
        <p className="mt-1 text-sm leading-6 text-amber-900">
          Um envio com resultado incerto precisa ser conferido antes de uma nova
          tentativa. O aceite do fornecedor ainda não confirma a entrega ao
          destinatário.
        </p>
      </div>
    </>
  )
}
function DemoProfile() {
  return (
    <div className="max-w-4xl space-y-6">
      <div className="flex items-center gap-3 text-brand-purple">
        <CircleUserRound size={30} aria-hidden="true" />
        <span className="text-xs font-bold uppercase tracking-[0.18em]">
          Conta e segurança
        </span>
      </div>
      <div>
        <h1 className="text-3xl font-bold">Meu perfil</h1>
        <p className="mt-2 text-sm text-slate-500">
          Dados fictícios para prévia da interface.
        </p>
      </div>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-bold">Dados da conta</h2>
        <dl className="mt-5 grid gap-5 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-slate-500">Nome</dt>
            <dd className="font-semibold">Marina Costa</dd>
          </div>
          <div>
            <dt className="text-slate-500">E-mail</dt>
            <dd className="font-semibold">marina@21mobile.com.br</dd>
          </div>
          <div>
            <dt className="text-slate-500">Perfil</dt>
            <dd className="font-semibold">Administradora</dd>
          </div>
        </dl>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="flex items-center gap-2 text-lg font-bold">
          <KeyRound size={20} aria-hidden="true" /> Alterar senha
        </h2>
        <p className="mt-2 text-sm text-slate-500">
          No painel real, esta ação encerra as sessões anteriores.
        </p>
        <div className="mt-5 max-w-md space-y-4">
          <label className="field">
            Senha atual
            <input
              className="form-control"
              type="password"
              disabled
              placeholder="Prévia fictícia"
            />
          </label>
          <label className="field">
            Nova senha
            <input
              className="form-control"
              type="password"
              disabled
              placeholder="Prévia fictícia"
            />
          </label>
          <button className="button-primary" disabled>
            Salvar nova senha
          </button>
        </div>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="flex items-center gap-2 text-lg font-bold">
          <ShieldCheck size={20} aria-hidden="true" /> Verificação em duas
          etapas
        </h2>
        <p className="mt-2 text-sm text-slate-500">
          No painel real, o aplicativo autenticador é configurado por QR code e
          confirmado com um código.
        </p>
        <span className="mt-4 inline-flex rounded-full bg-amber-50 px-3 py-1 text-xs font-bold text-amber-800">
          Desativada nesta prévia
        </span>
      </section>
    </div>
  )
}
function Listing({
  section,
  rows,
  onSave,
}: {
  section: Section
  rows: DemoRow[]
  onSave: (row: DemoRow) => void
}) {
  const config = sections[section]
  const state = useListState(
    section === 'agendamentos' ? [{ id: 'scheduledAt', desc: true }] : [],
  )
  const [status, setStatus] = useState('')
  const [editing, setEditing] = useState<DemoRow | null>(null)
  const [notice, setNotice] = useState('')
  const filtered = rows.filter(
    (row) =>
      (!status || row.status === status) &&
      Object.values(row).some((value) =>
        value
          .toLocaleLowerCase('pt-BR')
          .includes(state.search.toLocaleLowerCase('pt-BR')),
      ),
  )
  const sort = state.sorting[0]
  if (sort)
    filtered.sort(
      (a, b) =>
        String(a[sort.id as keyof DemoRow]).localeCompare(
          String(b[sort.id as keyof DemoRow]),
          'pt-BR',
          { numeric: true },
        ) * (sort.desc ? -1 : 1),
    )
  const columns: ColumnDef<DemoRow>[] = [
    {
      accessorKey: 'name',
      header: config.first,
      cell: ({ row }) => (
        <span className="font-semibold">{row.original.name}</span>
      ),
    },
    section === 'agendamentos'
      ? {
          accessorKey: 'scheduledAt',
          header: config.second,
          cell: ({ row }) => row.original.detail,
        }
      : { accessorKey: 'detail', header: config.second },
    {
      accessorKey: 'company',
      header: section === 'usuarios' ? 'Perfil' : 'Empresa',
      cell: ({ row }) => row.original.company,
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <Badge value={row.original.status} />,
    },
    {
      accessorKey: 'date',
      header: section === 'agendamentos' ? 'Solicitado em' : 'Atualização',
    },
  ]
  if (section === 'empresas') columns.splice(2, 1)
  if (config.entity)
    columns.push({
      id: 'actions',
      header: 'Ações',
      enableSorting: false,
      cell: ({ row }) => (
        <button
          className="inline-flex min-h-11 items-center gap-2 text-sm font-semibold text-brand-purple"
          onClick={() => setEditing(row.original)}
        >
          <Pencil size={15} />
          Editar
        </button>
      ),
    })
  if (section === 'callbacks')
    columns.push({
      id: 'actions',
      header: 'Ações',
      enableSorting: false,
      cell: ({ row }) =>
        row.original.status === 'Falhou' ? (
          <button
            className="text-sm font-semibold text-brand-purple"
            onClick={() => {
              onSave({ ...row.original, status: 'Pendente' })
              setNotice('Callback recolocado na fila de demonstração.')
            }}
          >
            Tentar novamente
          </button>
        ) : null,
    })
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{config.title}</h1>
          <p className="mt-2 text-sm text-slate-500">{config.description}</p>
        </div>
        {config.entity ? (
          <button
            className="button-primary"
            onClick={() =>
              setEditing({
                id: '',
                name: '',
                detail: '',
                scheduledAt: '',
                company:
                  section === 'usuarios' ? 'Consulta' : 'Horizonte Saúde',
                status: 'Ativo',
                date: '',
              })
            }
          >
            <Plus size={17} />
            {section === 'usuarios' ? 'Novo usuário' : `Nova ${config.entity}`}
          </button>
        ) : null}
      </div>
      {notice ? (
        <p
          role="status"
          className="mb-4 rounded-lg bg-emerald-50 p-3 text-sm text-emerald-800"
        >
          {notice}
        </p>
      ) : null}
      <DataTable
        columns={columns}
        data={filtered.slice(
          state.pagination.pageIndex * state.pagination.pageSize,
          (state.pagination.pageIndex + 1) * state.pagination.pageSize,
        )}
        total={filtered.length}
        getRowId={(row) => row.id}
        {...state}
        filters={
          <label className="flex flex-col gap-1 text-sm font-semibold">
            Status
            <select
              className="form-control min-w-40"
              value={status}
              onChange={(event) => {
                setStatus(event.target.value)
                state.onPaginationChange({ ...state.pagination, pageIndex: 0 })
              }}
            >
              <option value="">Todos os status</option>
              {Array.from(new Set(rows.map((row) => row.status))).map(
                (value) => (
                  <option key={value}>{value}</option>
                ),
              )}
            </select>
          </label>
        }
      />
      {editing ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-slate-950/40 p-4">
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="edit-title"
            className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl"
          >
            <h2 id="edit-title" className="text-xl font-bold">
              {editing.id ? 'Editar' : section === 'usuarios' ? 'Novo' : 'Nova'}{' '}
              {config.entity}
            </h2>
            <p className="mt-1 text-sm text-slate-500">
              Cadastro de demonstração.
            </p>
            <form
              className="mt-6 space-y-4"
              onSubmit={(event) => {
                event.preventDefault()
                onSave({
                  ...editing,
                  id: editing.id || `${section}-${crypto.randomUUID()}`,
                  date: new Date().toLocaleString('pt-BR'),
                })
                setEditing(null)
                setNotice('Cadastro salvo na demonstração.')
              }}
            >
              <label className="flex flex-col gap-2 text-sm font-semibold">
                {config.first}
                <input
                  autoFocus
                  required
                  maxLength={120}
                  className="form-control"
                  value={editing.name}
                  onChange={(event) =>
                    setEditing({ ...editing, name: event.target.value })
                  }
                />
              </label>
              <label className="flex flex-col gap-2 text-sm font-semibold">
                {config.second}
                <input
                  type={section === 'usuarios' ? 'email' : 'text'}
                  required={section === 'usuarios'}
                  className="form-control"
                  value={editing.detail}
                  onChange={(event) =>
                    setEditing({ ...editing, detail: event.target.value })
                  }
                />
              </label>
              {section === 'credenciais' ? (
                <label className="flex flex-col gap-2 text-sm font-semibold">
                  Empresa
                  <select
                    className="form-control"
                    value={editing.company}
                    onChange={(event) =>
                      setEditing({ ...editing, company: event.target.value })
                    }
                  >
                    {initialRows.empresas.slice(0, 6).map((row) => (
                      <option key={row.id}>{row.name}</option>
                    ))}
                  </select>
                </label>
              ) : null}
              {section === 'usuarios' ? (
                <div className="space-y-2">
                  <label className="flex flex-col gap-2 text-sm font-semibold">
                    Perfil
                    <select
                      className="form-control"
                      value={editing.company}
                      onChange={(event) =>
                        setEditing({ ...editing, company: event.target.value })
                      }
                    >
                      <option>Consulta</option>
                      <option>Operador</option>
                      <option>Administrador</option>
                    </select>
                  </label>
                  <p className="rounded-lg border border-purple-100 bg-purple-50 px-4 py-3 text-sm leading-6 text-purple-950">
                    {roleDescriptions[
                      editing.company === 'Administrador'
                        ? 'admin'
                        : editing.company === 'Operador'
                          ? 'operator'
                          : 'viewer'
                    ]}{' '}
                    Todos os perfis são da equipe interna.
                  </p>
                </div>
              ) : null}
              <label className="flex flex-col gap-2 text-sm font-semibold">
                Status
                <select
                  className="form-control"
                  value={editing.status}
                  onChange={(event) =>
                    setEditing({ ...editing, status: event.target.value })
                  }
                >
                  <option>Ativo</option>
                  <option>Bloqueado</option>
                </select>
              </label>
              <div className="flex justify-end gap-3 border-t border-slate-100 pt-5">
                <button
                  className="button-secondary"
                  type="button"
                  onClick={() => setEditing(null)}
                >
                  Cancelar
                </button>
                <button className="button-primary" type="submit">
                  Salvar alterações
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}
    </>
  )
}
export default function DemoApp() {
  const [rows, setRows] = useState(initialRows)
  const [loggedOut, setLoggedOut] = useState(true)
  const [visible, setVisible] = useState(false)
  if (loggedOut)
    return (
      <div className="grid min-h-screen lg:grid-cols-2">
        <div className="hidden flex-col justify-between bg-brand-purple p-14 text-white lg:flex">
          <img src="/brand/21-mobile.png" alt="21 Mobile" className="w-48" />
          <div>
            <h1 className="max-w-md text-4xl font-semibold leading-tight">
              Sua operação de SMS, conectada.
            </h1>
            <p className="mt-5 max-w-sm text-base leading-7 text-purple-100">
              Gerencie empresas, acessos e acompanhe os envios em um único
              painel.
            </p>
          </div>
          <p className="text-sm text-purple-200">21 Mobile · Backoffice</p>
        </div>
        <main className="flex items-center justify-center px-6 py-10">
          <section className="w-full max-w-sm">
            <div className="mb-8 w-fit rounded-lg bg-brand-purple px-5 py-4 lg:hidden">
              <img
                src="/brand/21-mobile.png"
                alt="21 Mobile"
                className="h-10 w-auto"
              />
            </div>
            <h2 className="text-3xl font-bold">Entrar no backoffice</h2>
            <p className="mt-3 text-sm text-slate-500">
              Prévia com dados fictícios. Nenhuma credencial real é necessária.
            </p>
            <form
              className="mt-8 space-y-5"
              onSubmit={(event) => {
                event.preventDefault()
                setLoggedOut(false)
              }}
            >
              <label className="block text-sm font-semibold">
                E-mail
                <input
                  type="email"
                  autoComplete="username"
                  className="form-control mt-2 w-full"
                  defaultValue="marina@21mobile.example"
                />
              </label>
              <label className="block text-sm font-semibold">
                Senha
                <div className="mt-2 flex">
                  <input
                    type={visible ? 'text' : 'password'}
                    autoComplete="current-password"
                    className="form-control min-w-0 flex-1"
                    defaultValue="demonstracao"
                  />
                  <button
                    type="button"
                    className="ml-2 min-h-11 rounded-lg border border-slate-200 px-3 text-sm"
                    onClick={() => setVisible((value) => !value)}
                  >
                    {visible ? 'Ocultar' : 'Mostrar'}
                  </button>
                </div>
              </label>
              <button className="button-primary w-full" type="submit">
                Entrar na demonstração
              </button>
            </form>
          </section>
        </main>
      </div>
    )
  return (
    <PanelShell
      name="Marina Costa"
      role="Administradora"
      admin
      demo
      onLogout={() => setLoggedOut(true)}
    >
      {' '}
      <Routes>
        <Route path="/" element={<Overview />} />
        <Route path="/perfil" element={<DemoProfile />} />
        {(Object.keys(sections) as Section[]).map((section) => (
          <Route
            key={section}
            path={`/${section}`}
            element={
              <Listing
                key={section}
                section={section}
                rows={rows[section]}
                onSave={(row) =>
                  setRows((current) => ({
                    ...current,
                    [section]: current[section].some(
                      (value) => value.id === row.id,
                    )
                      ? current[section].map((value) =>
                          value.id === row.id ? row : value,
                        )
                      : [row, ...current[section]],
                  }))
                }
              />
            }
          />
        ))}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </PanelShell>
  )
}
