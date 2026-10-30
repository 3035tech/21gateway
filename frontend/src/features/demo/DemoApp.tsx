import { PanelShell } from '../../components/admin/PanelShell'
import { useState } from 'react'
import {
  Route,
  Routes,
  Link,
  Navigate,
  useNavigate,
  useParams,
  useSearchParams,
} from 'react-router-dom'
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
  Eye,
  Archive,
  RotateCcw,
} from 'lucide-react'
import { TableIconButton } from '../../components/admin/TableIconAction'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { initialRows, sections } from './data'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../../components/admin/LanguageSwitcher'
import { Dialog } from '../../components/admin/Dialog'
import { useFeedback } from '../../components/admin/feedbackContext'
import { AppSelect } from '../../components/admin/AppSelect'
import { RolePicker } from '../users/RolePicker'
import { CompanyAccessPicker } from '../users/CompanyAccessPicker'
import type { DemoRow, Section } from './data'

const demoSessionKey = '21gateway-demo-session-until'
const demoUserKey = '21gateway-demo-user-id'
const demoRememberDurationMs = 14 * 24 * 60 * 60 * 1000
const demoLoginUsers = ['usuarios-1', 'usuarios-2', 'usuarios-3', 'usuarios-4']

function visibleDemoRows(
  section: Section,
  rows: DemoRow[],
  user: DemoRow,
  selectedCompany?: string,
) {
  if (user.company === 'admin') return rows
  if (section === 'usuarios' || section === 'auditoria') return []
  const allowed = new Set(
    selectedCompany ? [selectedCompany] : (user.companyAccess ?? []),
  )
  return rows.filter((row) =>
    allowed.has(section === 'empresas' ? row.name : row.company),
  )
}

function Badge({ value }: { value: string }) {
  const { t } = useTranslation()
  const tone = ['active', 'accepted', 'delivered'].includes(value)
    ? 'bg-emerald-50 text-emerald-800'
    : ['failed', 'rejected', 'blocked'].includes(value)
      ? 'bg-rose-50 text-rose-800'
      : 'bg-amber-50 text-amber-800'
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ${tone}`}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {t(demoStatusKey(value))}
    </span>
  )
}
function demoStatusKey(value: string) {
  return value === 'recorded' ? 'demo.recorded' : `status.${value}`
}
function demoRoleKey(value: string) {
  return `roles.${value}`
}
function demoRoleValue(role: 'admin' | 'operator' | 'viewer') {
  return role
}
function Overview() {
  const { t } = useTranslation()
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.overview')}
          </h1>
          <p className="mt-2 text-sm text-slate-500">
            {t('overview.description')}
          </p>
        </div>
        <span className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
          <Activity size={16} /> {t('demo.simulatedOperation')}
        </span>
      </div>
      <section className="overflow-hidden rounded-2xl bg-brand-purple text-white">
        <div className="grid lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
          <div className="p-6 sm:p-8">
            <h2 className="flex items-center gap-2 text-sm font-semibold text-purple-100">
              <CalendarClock size={19} /> {t('overview.scheduleQueue')}
            </h2>
            <div className="mt-5 flex flex-wrap items-end gap-5">
              <strong className="text-6xl font-bold leading-none tabular-nums tracking-tight sm:text-7xl">
                8
              </strong>
              <p className="max-w-56 pb-1 text-sm leading-6 text-purple-100">
                {t('overview.overdue')}
              </p>
            </div>
            <Link
              to="/agendamentos"
              className="mt-7 inline-flex min-h-11 items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-bold text-brand-purple"
            >
              {t('overview.viewSchedules')} <ArrowRight size={17} />
            </Link>
          </div>
          <div className="grid grid-cols-3 border-t border-white/20 bg-white/10 lg:grid-cols-1 lg:border-l lg:border-t-0">
            {(
              [
                [t('overview.next24h'), '12'],
                [t('overview.processing'), '5'],
                [t('overview.queuedTotal'), '27'],
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
              <Send size={20} className="text-brand-purple" />{' '}
              {t('overview.recordedMessages')}
            </h2>
            <Link to="/envios" className="text-sm font-bold text-brand-purple">
              {t('overview.open')} <ArrowRight size={15} className="inline" />
            </Link>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            {t('overview.historyTotal')}
          </p>
          <dl className="mt-6 grid grid-cols-3 gap-3 border-t border-slate-100 pt-6">
            {(
              [
                [t('overview.accepted'), '32'],
                [t('overview.rejected'), '16'],
                [t('overview.unknown'), '16'],
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
            {t('overview.deliveryWarning')}
          </p>
        </section>
        <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
          <div className="flex items-center justify-between gap-3">
            <h2 className="flex items-center gap-2 text-lg font-bold">
              <Webhook size={20} className="text-brand-purple" />{' '}
              {t('overview.clientNotifications')}
            </h2>
            <Link
              to="/callbacks"
              className="text-sm font-bold text-brand-purple"
            >
              {t('overview.viewDlr')}{' '}
              <ArrowRight size={15} className="inline" />
            </Link>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            {t('overview.callbackJobs')}
          </p>
          <div className="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-6">
            <div>
              <h3 className="text-sm font-bold">{t('overview.dlrStatus')}</h3>
              <p className="mt-2 text-2xl font-bold">
                8{' '}
                <span className="text-xs font-normal text-slate-500">
                  {t('overview.pending')}
                </span>
              </p>
              <p className="mt-1 text-sm text-rose-700">
                {t('overview.failedCount', { value: 2 })}
              </p>
            </div>
            <div>
              <h3 className="text-sm font-bold">{t('overview.moReplies')}</h3>
              <p className="mt-2 text-2xl font-bold">
                4{' '}
                <span className="text-xs font-normal text-slate-500">
                  {t('overview.pending')}
                </span>
              </p>
              <p className="mt-1 text-sm text-rose-700">
                {t('overview.failedCount', { value: 1 })}
              </p>
            </div>
          </div>
        </section>
      </div>
      <section className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-3 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-sm">
        <span className="font-bold">{t('overview.activeAccess')}</span>
        <span className="inline-flex items-center gap-2">
          <Building2 size={17} className="text-brand-purple" />{' '}
          {t('overview.companyCount', { value: 12 })}
        </span>
        <span className="inline-flex items-center gap-2">
          <KeyRound size={17} className="text-brand-purple" />{' '}
          {t('overview.credentialCount', { value: 11 })}
        </span>
        <Link to="/credenciais" className="ml-auto font-bold text-brand-purple">
          {t('overview.manageAccess')}{' '}
          <ArrowRight size={15} className="inline" />
        </Link>
      </section>
      <div className="mt-6 grid gap-6 xl:grid-cols-[1.6fr_1fr]">
        <section className="rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-100 p-5">
            <h2 className="font-bold">{t('demo.recentMessages')}</h2>
            <Link
              to="/envios"
              className="flex items-center gap-1 text-sm font-semibold text-brand-purple"
            >
              {t('demo.viewAll')} <ArrowRight size={15} />
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
          <h2 className="text-lg font-bold">{t('demo.separateAccess')}</h2>
          <p className="mt-3 text-sm leading-6 text-slate-500">
            {t('demo.separateAccessHint')}
          </p>
          <div className="mt-6 space-y-3">
            <Link className="button-secondary w-full" to="/usuarios">
              {t('demo.manageTeam')}
            </Link>
            <Link className="button-primary w-full" to="/credenciais">
              {t('demo.manageCredentials')}
            </Link>
          </div>
        </section>
      </div>
      <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-5">
        <h2 className="font-semibold text-amber-900">
          {t('demo.operationWarningTitle')}
        </h2>
        <p className="mt-1 text-sm leading-6 text-amber-900">
          {t('demo.operationWarning')}
        </p>
      </div>
    </>
  )
}
function ScopedOverview({
  rows,
  user,
  selectedCompany,
}: {
  rows: Record<Section, DemoRow[]>
  user: DemoRow
  selectedCompany?: string
}) {
  const { t } = useTranslation()
  const companies = visibleDemoRows(
    'empresas',
    rows.empresas,
    user,
    selectedCompany,
  )
  const credentials = visibleDemoRows(
    'credenciais',
    rows.credenciais,
    user,
    selectedCompany,
  )
  const messages = visibleDemoRows('envios', rows.envios, user, selectedCompany)
  const schedules = visibleDemoRows(
    'agendamentos',
    rows.agendamentos,
    user,
    selectedCompany,
  )
  const callbacks = visibleDemoRows(
    'callbacks',
    rows.callbacks,
    user,
    selectedCompany,
  )
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          {t('nav.overview')}
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          {t('demo.scopedOverview', { name: user.name })}
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {companies.map((company) => (
          <span
            key={company.id}
            className="rounded-full bg-purple-50 px-3 py-1.5 text-sm font-semibold text-brand-purple"
          >
            {company.name}
          </span>
        ))}
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {(
          [
            ['nav.companies', companies.length, '/empresas', Building2],
            ['nav.credentials', credentials.length, '/credenciais', KeyRound],
            ['nav.messages', messages.length, '/envios', Send],
            ['nav.schedules', schedules.length, '/agendamentos', CalendarClock],
          ] as const
        ).map(([label, count, path, Icon]) => (
          <Link
            key={path}
            to={path}
            className="group rounded-2xl border border-slate-200 bg-white p-5 transition-colors hover:border-purple-300 hover:bg-purple-50"
          >
            <Icon size={20} className="text-brand-purple" aria-hidden="true" />
            <strong className="mt-5 block text-3xl tabular-nums">
              {count}
            </strong>
            <span className="mt-1 block text-sm font-semibold text-slate-600 group-hover:text-brand-purple">
              {t(label)}
            </span>
          </Link>
        ))}
      </div>
      <section className="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6">
        <div className="flex items-center justify-between gap-3">
          <h2 className="flex items-center gap-2 text-lg font-bold">
            <Webhook size={20} className="text-brand-purple" />
            {t('overview.clientNotifications')}
          </h2>
          <Link
            to="/callbacks"
            className="text-sm font-semibold text-brand-purple hover:underline"
          >
            {t('demo.viewAll')}
          </Link>
        </div>
        <p className="mt-3 text-sm text-slate-600">
          {t('demo.scopedCallbacks', { count: callbacks.length })}
        </p>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white">
        <div className="flex items-center justify-between gap-3 border-b border-slate-100 p-5">
          <h2 className="font-bold">{t('demo.recentMessages')}</h2>
          <Link
            to="/envios"
            className="text-sm font-semibold text-brand-purple hover:underline"
          >
            {t('demo.viewAll')}
          </Link>
        </div>
        {messages.slice(0, 5).map((row) => (
          <div
            key={row.id}
            className="flex items-center justify-between gap-3 border-b border-slate-100 px-5 py-4 last:border-0"
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
    </div>
  )
}
function DemoProfile({ user }: { user: DemoRow }) {
  const { t } = useTranslation()
  return (
    <div className="max-w-4xl space-y-6">
      <div className="flex items-center gap-3 text-brand-purple">
        <CircleUserRound size={30} aria-hidden="true" />
        <span className="text-xs font-bold uppercase tracking-[0.18em]">
          {t('auth.accountSecurity')}
        </span>
      </div>
      <div>
        <h1 className="text-3xl font-bold">{t('nav.profile')}</h1>
        <p className="mt-2 text-sm text-slate-500">{t('demo.profileHint')}</p>
      </div>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-bold">{t('profile.accountDetails')}</h2>
        <dl className="mt-5 grid gap-5 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-slate-500">{t('common.name')}</dt>
            <dd className="font-semibold">{user.name}</dd>
          </div>
          <div>
            <dt className="text-slate-500">{t('common.email')}</dt>
            <dd className="font-semibold">{user.detail}</dd>
          </div>
          <div>
            <dt className="text-slate-500">{t('roles.label')}</dt>
            <dd className="font-semibold">{t(demoRoleKey(user.company))}</dd>
          </div>
        </dl>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="flex items-center gap-2 text-lg font-bold">
          <KeyRound size={20} aria-hidden="true" />{' '}
          {t('profile.changePassword')}
        </h2>
        <p className="mt-2 text-sm text-slate-500">{t('demo.passwordHint')}</p>
        <div className="mt-5 max-w-md space-y-4">
          <label className="field">
            {t('profile.currentPassword')}
            <input
              className="form-control"
              type="password"
              disabled
              placeholder={t('demo.previewPlaceholder')}
            />
          </label>
          <label className="field">
            {t('auth.newPassword')}
            <input
              className="form-control"
              type="password"
              disabled
              placeholder={t('demo.previewPlaceholder')}
            />
          </label>
          <button className="button-primary" disabled>
            {t('auth.saveNewPassword')}
          </button>
        </div>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="flex items-center gap-2 text-lg font-bold">
          <ShieldCheck size={20} aria-hidden="true" /> {t('mfa.title')}
        </h2>
        <p className="mt-2 text-sm text-slate-500">{t('demo.mfaHint')}</p>
        <span className="mt-4 inline-flex rounded-full bg-amber-50 px-3 py-1 text-xs font-bold text-amber-800">
          {t('demo.mfaDisabled')}
        </span>
      </section>
    </div>
  )
}
function Listing({
  section,
  rows,
  companies,
  companyCredentials,
  companyUsers,
  canManage,
  admin,
  onSave,
  onArchive,
}: {
  section: Section
  rows: DemoRow[]
  companies: DemoRow[]
  companyCredentials: DemoRow[]
  companyUsers: DemoRow[]
  canManage: boolean
  admin: boolean
  onSave: (row: DemoRow) => void
  onArchive: (id: string) => void
}) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const fromCompany = searchParams.get('fromCompany')
  const { recordId } = useParams()
  const config = sections[section]
  const state = useListState(
    section === 'agendamentos' ? [{ id: 'scheduledAt', desc: true }] : [],
  )
  const [status, setStatus] = useState('')
  const [editing, setEditing] = useState<DemoRow | null>(null)
  const [formError, setFormError] = useState('')
  const [companySearch, setCompanySearch] = useState('')
  const [companyPage, setCompanyPage] = useState(1)
  const [demoPasswordMode, setDemoPasswordMode] = useState<
    'generated' | 'manual' | 'legacy'
  >('generated')
  const [demoPassword, setDemoPassword] = useState('')
  const [demoProviderToken, setDemoProviderToken] = useState('')
  const [demoClearProviderToken, setDemoClearProviderToken] = useState(false)
  const [demoGenerateUsername, setDemoGenerateUsername] = useState(false)
  const [demoSecret, setDemoSecret] = useState<{
    username: string
    password: string
  } | null>(null)
  const [viewing, setViewing] = useState<DemoRow | null>(
    () => rows.find((row) => row.id === recordId) ?? null,
  )
  const [archiving, setArchiving] = useState<DemoRow | null>(null)
  const feedback = useFeedback()
  const matchingCompanies = companies
    .filter((company) => company.status === 'active')
    .filter((company) =>
      `${company.name} ${company.detail}`
        .toLocaleLowerCase(i18n.language)
        .includes(companySearch.trim().toLocaleLowerCase(i18n.language)),
    )
    .sort((a, b) => a.name.localeCompare(b.name, i18n.language))
  const filtered = rows.filter(
    (row) =>
      (!status || row.status === status) &&
      Object.values(row).some((value) =>
        String(value)
          .toLocaleLowerCase(i18n.language)
          .includes(state.search.toLocaleLowerCase(i18n.language)),
      ),
  )
  const sort = state.sorting[0]
  if (sort)
    filtered.sort(
      (a, b) =>
        String(a[sort.id as keyof DemoRow]).localeCompare(
          String(b[sort.id as keyof DemoRow]),
          i18n.language,
          { numeric: true },
        ) * (sort.desc ? -1 : 1),
    )
  const columns: ColumnDef<DemoRow>[] = [
    {
      accessorKey: 'name',
      header: t(config.first),
      cell: ({ row }) => (
        <span className="font-semibold">{row.original.name}</span>
      ),
    },
    section === 'agendamentos'
      ? {
          accessorKey: 'scheduledAt',
          header: t(config.second),
          cell: ({ row }) => row.original.detail,
        }
      : {
          accessorKey: 'detail',
          header: t(config.second),
          cell: ({ row }) =>
            section === 'auditoria'
              ? t(`demo.auditActions.${row.original.detail}`)
              : row.original.detail,
        },
    {
      accessorKey: 'company',
      header: section === 'usuarios' ? t('roles.label') : t('common.company'),
      cell: ({ row }) =>
        section === 'usuarios'
          ? t(demoRoleKey(row.original.company))
          : row.original.company,
    },
    {
      accessorKey: 'status',
      header: t('common.status'),
      meta: { compact: true },
      cell: ({ row }) => <Badge value={row.original.status} />,
    },
    {
      accessorKey: 'date',
      meta: { compact: true },
      header:
        section === 'agendamentos'
          ? t('schedules.requestedAt')
          : t('common.updatedAt'),
    },
  ]
  if (section === 'empresas') columns.splice(2, 1)
  columns.push({
    id: 'actions',
    header: t('common.actions'),
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex w-max items-center gap-1">
        <TableIconButton
          icon={Eye}
          label={t('common.view')}
          onClick={() => setViewing(row.original)}
        />
        {canManage && config.entity ? (
          <>
            <TableIconButton
              icon={Pencil}
              label={t('common.edit')}
              onClick={() => {
                setDemoProviderToken('')
                setDemoClearProviderToken(false)
                setFormError('')
                setEditing(row.original)
              }}
            />
            <TableIconButton
              icon={Archive}
              label={t('common.archive')}
              danger
              onClick={() => setArchiving(row.original)}
            />
          </>
        ) : null}
        {canManage &&
        section === 'callbacks' &&
        row.original.status === 'failed' ? (
          <TableIconButton
            icon={RotateCcw}
            label={t('common.retry')}
            onClick={() => {
              onSave({ ...row.original, status: 'pending' })
              feedback.success(t('demo.callbackRetried'))
            }}
          />
        ) : null}
      </div>
    ),
  })
  return (
    <>
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t(config.title)}
          </h1>
          <p className="mt-2 text-sm text-slate-500">{t(config.description)}</p>
        </div>
        {canManage && config.entity && (admin || section !== 'empresas') ? (
          <button
            className="button-primary"
            onClick={() => {
              setDemoGenerateUsername(false)
              setDemoPasswordMode('generated')
              setDemoPassword('')
              setDemoProviderToken('')
              setDemoClearProviderToken(false)
              setFormError('')
              setEditing({
                id: '',
                name: '',
                detail: '',
                scheduledAt: '',
                company:
                  section === 'usuarios'
                    ? 'viewer'
                    : section === 'credenciais'
                      ? (companies.find(
                          (company) => company.status === 'active',
                        )?.name ?? '')
                      : 'Horizonte Saúde',
                companyAccess:
                  section === 'usuarios' ? ['Horizonte Saúde'] : [],
                status: 'active',
                date: '',
              })
            }}
          >
            <Plus size={17} />
            {t(`demo.newEntity.${section}`)}
          </button>
        ) : null}
      </div>
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
            {t('common.status')}
            <AppSelect
              className="min-w-40"
              value={status}
              label={t('common.status')}
              onValueChange={(value) => {
                setStatus(value)
                state.onPaginationChange({ ...state.pagination, pageIndex: 0 })
              }}
              options={[
                { value: '', label: t('demo.allStatuses') },
                ...Array.from(new Set(rows.map((row) => row.status))).map(
                  (value) => ({
                    value,
                    label: t(demoStatusKey(value)),
                  }),
                ),
              ]}
            />
          </label>
        }
      />
      {viewing ? (
        <Dialog
          title={t('demo.viewRecord')}
          onClose={() => {
            setViewing(null)
            if (recordId)
              navigate(fromCompany ? `/empresas/${fromCompany}` : `/${section}`)
          }}
        >
          <dl className="space-y-4 text-sm">
            {[
              [t(config.first), viewing.name],
              [
                t(config.second),
                section === 'auditoria'
                  ? t(`demo.auditActions.${viewing.detail}`)
                  : viewing.detail,
              ],
              ...(section === 'empresas'
                ? []
                : [
                    [
                      section === 'usuarios'
                        ? t('roles.label')
                        : t('common.company'),
                      section === 'usuarios'
                        ? t(demoRoleKey(viewing.company))
                        : viewing.company,
                    ],
                  ]),
              [t('common.status'), t(demoStatusKey(viewing.status))],
              ...(section === 'empresas'
                ? [
                    [
                      t('companies.providerAccess'),
                      t(
                        viewing.providerTokenConfigured
                          ? 'companies.providerConfigured'
                          : 'companies.providerMissing',
                      ),
                    ],
                  ]
                : []),
              [t('common.updatedAt'), viewing.date],
              ...(section === 'envios'
                ? [
                    [
                      t('messages.contentLabel'),
                      t('demo.sampleMessage', { company: viewing.company }),
                    ],
                  ]
                : []),
            ].map(([label, value]) => (
              <div
                key={label}
                className="border-b border-slate-100 pb-3 last:border-0"
              >
                <dt className="text-slate-500">{label}</dt>
                <dd className="mt-1 break-words font-semibold text-slate-900">
                  {value || '—'}
                </dd>
              </div>
            ))}
          </dl>
          {section === 'empresas' ? (
            <div className="mt-6 rounded-xl bg-purple-50 p-4">
              <h3 className="font-bold text-brand-purple">
                {t('companyDetail.credentialsTitle')}
              </h3>
              {admin ? (
                <p className="mt-1 text-xs text-slate-600">
                  {t('demo.adminUsersGlobal')}
                </p>
              ) : null}
              <ul className="mt-3 space-y-2 text-sm">
                {companyCredentials
                  .filter(
                    (credential) =>
                      credential.company === viewing.name &&
                      credential.status === 'active',
                  )
                  .map((credential) => (
                    <li
                      key={credential.id}
                      className="rounded-lg bg-white font-semibold"
                    >
                      <Link
                        to={`/credenciais/${credential.id}?fromCompany=${viewing.id}`}
                        className="flex min-h-11 items-center gap-2 rounded-lg px-3 py-2 text-brand-purple hover:bg-white/70 hover:underline"
                      >
                        <KeyRound size={16} aria-hidden="true" />{' '}
                        {credential.name}{' '}
                        <ArrowRight size={15} aria-hidden="true" />
                      </Link>
                    </li>
                  ))}
              </ul>
              {admin ? (
                <>
                  <h3 className="mt-5 font-bold text-brand-purple">
                    {t('users.companyUsersTitle')}
                  </h3>
                  <ul className="mt-3 space-y-2 text-sm">
                    {companyUsers
                      .filter(
                        (user) =>
                          user.status === 'active' &&
                          user.companyAccess?.includes(viewing.name),
                      )
                      .map((user) => (
                        <li key={user.id} className="rounded-lg bg-white">
                          <Link
                            to={`/usuarios/${user.id}?fromCompany=${viewing.id}`}
                            className="flex min-h-11 items-center gap-2 rounded-lg px-3 py-2 text-brand-purple hover:bg-white/70 hover:underline"
                          >
                            <CircleUserRound size={16} aria-hidden="true" />{' '}
                            {user.name} · {t(demoRoleKey(user.company))}{' '}
                            <ArrowRight size={15} aria-hidden="true" />
                          </Link>
                        </li>
                      ))}
                  </ul>
                </>
              ) : null}
            </div>
          ) : null}
          <div className="mt-6 flex justify-end">
            <button
              className="button-secondary"
              onClick={() => {
                setViewing(null)
                if (recordId)
                  navigate(
                    fromCompany ? `/empresas/${fromCompany}` : `/${section}`,
                  )
              }}
            >
              {t('common.close')}
            </button>
          </div>
        </Dialog>
      ) : null}
      {archiving ? (
        <Dialog
          title={t('demo.archiveTitle')}
          onClose={() => setArchiving(null)}
        >
          <p className="text-sm leading-6 text-slate-600">
            {t('demo.archiveHint', { name: archiving.name })}
          </p>
          <div className="mt-6 flex justify-end gap-3">
            <button
              className="button-secondary"
              onClick={() => setArchiving(null)}
            >
              {t('common.cancel')}
            </button>
            <button
              className="button-danger"
              onClick={() => {
                onArchive(archiving.id)
                setArchiving(null)
                feedback.success(t('demo.archived'))
              }}
            >
              {t('common.archive')}
            </button>
          </div>
        </Dialog>
      ) : null}
      {editing ? (
        <Dialog
          title={t(
            `demo.${editing.id ? 'editEntity' : 'newEntity'}.${section}`,
          )}
          onClose={() => {
            setEditing(null)
            setFormError('')
          }}
        >
          <p className="mt-1 text-sm text-slate-500">
            {section === 'usuarios' && !editing.id
              ? t('demo.inviteHint')
              : t('demo.recordHint')}
          </p>
          <form
            className="mt-6 space-y-4"
            noValidate
            onInput={() => setFormError('')}
            onSubmit={(event) => {
              event.preventDefault()
              if (!demoGenerateUsername && !editing.name.trim()) {
                setFormError(t('validation.name'))
                return
              }
              if (
                section === 'usuarios' &&
                !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(editing.detail.trim())
              ) {
                setFormError(t('validation.email'))
                return
              }
              if (section === 'credenciais' && !editing.company) {
                setFormError(t('validation.selectCompany'))
                return
              }
              if (section === 'credenciais' && editing.detail) {
                let url: URL
                try {
                  url = new URL(editing.detail)
                } catch {
                  setFormError(t('validation.httpsUrl'))
                  return
                }
                if (
                  url.protocol !== 'https:' ||
                  url.username ||
                  url.password ||
                  url.hash ||
                  (url.port && url.port !== '443')
                ) {
                  setFormError(t('validation.httpsUrl'))
                  return
                }
              }
              const username =
                section === 'credenciais' && !editing.id && demoGenerateUsername
                  ? `gw_${crypto.randomUUID().replaceAll('-', '').slice(0, 12)}`
                  : editing.name
              if (
                section === 'credenciais' &&
                !editing.id &&
                demoPasswordMode !== 'generated' &&
                !demoPassword
              ) {
                setFormError(t('validation.manualPasswordRequired'))
                return
              }
              if (
                section === 'credenciais' &&
                !editing.id &&
                demoPasswordMode === 'manual' &&
                Array.from(demoPassword.trim()).length < 15
              ) {
                setFormError(t('validation.passwordMinimumTrimmed'))
                return
              }
              if (
                section === 'credenciais' &&
                !editing.id &&
                demoPasswordMode === 'generated'
              )
                setDemoSecret({
                  username,
                  password: crypto.randomUUID(),
                })
              onSave({
                ...editing,
                name: username,
                providerTokenConfigured:
                  section === 'empresas'
                    ? demoClearProviderToken
                      ? false
                      : demoProviderToken
                        ? true
                        : editing.providerTokenConfigured
                    : editing.providerTokenConfigured,
                id: editing.id || `${section}-${crypto.randomUUID()}`,
                date: new Date().toLocaleString(i18n.language),
              })
              setEditing(null)
              setFormError('')
              setDemoPassword('')
              setDemoProviderToken('')
              feedback.success(
                section === 'usuarios' && !editing.id
                  ? t('demo.invitationSimulated')
                  : t('demo.recordSaved'),
              )
            }}
          >
            {section === 'credenciais' ? (
              <label className="flex flex-col gap-2 text-sm font-semibold">
                {t('common.company')}
                {admin && !editing.id ? (
                  <AppSelect
                    value={editing.company}
                    label={t('common.company')}
                    onValueChange={(value) =>
                      setEditing({ ...editing, company: value })
                    }
                    options={companies
                      .filter((row) => row.status === 'active')
                      .map((row) => ({ value: row.name, label: row.name }))}
                  />
                ) : (
                  <input
                    className="form-control"
                    value={editing.company}
                    readOnly
                  />
                )}
              </label>
            ) : null}
            {section === 'credenciais' ? (
              <div className="rounded-xl border border-purple-100 bg-purple-50 p-4">
                <h3 className="font-bold text-brand-purple">
                  {t('credentials.clientAccess')}
                </h3>
                <p className="mt-1 text-xs leading-5 text-slate-600">
                  {t('credentials.basicAuthHint')}
                </p>
              </div>
            ) : null}
            {section === 'credenciais' && !editing.id ? (
              <label className="flex min-h-11 items-center gap-2 text-sm font-semibold">
                <input
                  type="checkbox"
                  className="size-4 accent-brand-purple"
                  checked={demoGenerateUsername}
                  onChange={(event) =>
                    setDemoGenerateUsername(event.target.checked)
                  }
                />
                {t('credentials.generateUsername')}
              </label>
            ) : null}
            <label className="flex flex-col gap-2 text-sm font-semibold">
              {t(config.first)}
              <input
                autoFocus
                required={!demoGenerateUsername}
                maxLength={120}
                className="form-control"
                readOnly={
                  section === 'credenciais' &&
                  (!!editing.id || demoGenerateUsername)
                }
                placeholder={
                  section === 'credenciais' && demoGenerateUsername
                    ? t('credentials.generatedUsernamePlaceholder')
                    : undefined
                }
                value={editing.name}
                onChange={(event) =>
                  setEditing({ ...editing, name: event.target.value })
                }
              />
            </label>
            {section !== 'credenciais' ? (
              <label className="flex flex-col gap-2 text-sm font-semibold">
                {t(config.second)}
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
            ) : null}
            {section === 'empresas' ? (
              <div className="space-y-3 rounded-xl border border-purple-100 bg-purple-50 p-4">
                <h3 className="font-bold text-brand-purple">
                  {t('companies.providerAccess')}
                </h3>
                <p className="text-xs leading-5 text-slate-600">
                  {t(
                    editing.providerTokenConfigured
                      ? 'companies.providerConfigured'
                      : 'companies.providerMissing',
                  )}
                </p>
                {!demoClearProviderToken ? (
                  <label className="field">
                    {t('companies.providerToken')}
                    <input
                      type="password"
                      autoComplete="new-password"
                      className="form-control"
                      value={demoProviderToken}
                      onChange={(event) =>
                        setDemoProviderToken(event.target.value)
                      }
                    />
                  </label>
                ) : null}
                {editing.providerTokenConfigured ? (
                  <label className="flex min-h-11 items-center gap-2 text-sm text-rose-800">
                    <input
                      type="checkbox"
                      checked={demoClearProviderToken}
                      onChange={(event) => {
                        setDemoClearProviderToken(event.target.checked)
                        if (event.target.checked) setDemoProviderToken('')
                      }}
                    />
                    {t('companies.providerRemove')}
                  </label>
                ) : null}
              </div>
            ) : null}
            {section === 'credenciais' ? (
              editing.id ? (
                <p className="rounded-xl bg-slate-50 p-4 text-sm text-slate-600">
                  {t('credentials.passwordStoredHint')}
                </p>
              ) : (
                <fieldset className="space-y-3 rounded-xl border border-slate-200 p-4">
                  <legend className="px-1 text-sm font-bold">
                    {t('credentials.passwordSetup')}
                  </legend>
                  {(['generated', 'manual', 'legacy'] as const).map((mode) => (
                    <label
                      key={mode}
                      className="flex min-h-10 items-center gap-2 text-sm"
                    >
                      <input
                        type="radio"
                        className="size-4 accent-brand-purple"
                        name="demo-basic-password"
                        checked={demoPasswordMode === mode}
                        onChange={() => {
                          setDemoPasswordMode(mode)
                          setDemoPassword('')
                        }}
                      />
                      {t(`credentials.passwordMode.${mode}`)}
                    </label>
                  ))}
                  {demoPasswordMode !== 'generated' ? (
                    <label className="flex flex-col gap-2 text-sm font-semibold">
                      {t(
                        demoPasswordMode === 'manual'
                          ? 'credentials.manualPassword'
                          : 'credentials.currentApiPassword',
                      )}
                      <input
                        type="password"
                        autoComplete="new-password"
                        required
                        minLength={demoPasswordMode === 'manual' ? 15 : 1}
                        className="form-control"
                        value={demoPassword}
                        onChange={(event) =>
                          setDemoPassword(event.target.value)
                        }
                      />
                    </label>
                  ) : (
                    <p className="text-xs text-slate-500">
                      {t('credentials.newPasswordHint')}
                    </p>
                  )}
                </fieldset>
              )
            ) : null}
            {section === 'credenciais' ? (
              <label className="flex flex-col gap-2 text-sm font-semibold">
                {t('credentials.dlrUrl')}
                <input
                  type="url"
                  className="form-control"
                  value={editing.detail}
                  onChange={(event) =>
                    setEditing({ ...editing, detail: event.target.value })
                  }
                />
              </label>
            ) : null}
            {section === 'usuarios' ? (
              <>
                <RolePicker
                  value={
                    editing.company === 'admin'
                      ? 'admin'
                      : editing.company === 'operator'
                        ? 'operator'
                        : 'viewer'
                  }
                  onChange={(role) =>
                    setEditing({
                      ...editing,
                      company: demoRoleValue(role),
                      companyAccess:
                        role === 'admin'
                          ? []
                          : editing.companyAccess?.length
                            ? editing.companyAccess
                            : ['Horizonte Saúde'],
                    })
                  }
                />
                <p className="text-xs leading-5 text-slate-500">
                  {editing.company === 'admin'
                    ? t('roles.globalAccess')
                    : t('users.companyAccessHint')}
                </p>
                {editing.company !== 'admin' ? (
                  <div className="space-y-2">
                    <CompanyAccessPicker
                      options={matchingCompanies
                        .slice((companyPage - 1) * 10, companyPage * 10)
                        .map((company) => ({
                          id: company.name,
                          name: company.name,
                        }))}
                      selected={(editing.companyAccess ?? []).map((name) => ({
                        id: name,
                        name,
                      }))}
                      selectedIds={editing.companyAccess ?? []}
                      selectedCount={editing.companyAccess?.length ?? 0}
                      selectedPage={1}
                      selectedPageSize={5}
                      onSelectedPageChange={() => {}}
                      search={companySearch}
                      onSearchChange={(value) => {
                        setCompanySearch(value)
                        setCompanyPage(1)
                      }}
                      onToggle={(name) =>
                        setEditing({
                          ...editing,
                          companyAccess: editing.companyAccess?.includes(name)
                            ? editing.companyAccess.filter(
                                (companyName) => companyName !== name,
                              )
                            : [...(editing.companyAccess ?? []), name],
                        })
                      }
                      page={companyPage}
                      pageSize={10}
                      total={matchingCompanies.length}
                      onPageChange={setCompanyPage}
                    />
                    {!editing.companyAccess?.length ? (
                      <p className="text-xs text-rose-700">
                        {t('validation.companyAccess')}
                      </p>
                    ) : null}
                  </div>
                ) : null}
              </>
            ) : null}
            <label className="flex flex-col gap-2 text-sm font-semibold">
              {t('common.status')}
              <AppSelect
                value={editing.status}
                label={t('common.status')}
                onValueChange={(value) =>
                  setEditing({ ...editing, status: value })
                }
                options={[
                  { value: 'active', label: t('status.active') },
                  { value: 'blocked', label: t('status.blocked') },
                ]}
              />
            </label>
            {formError ? (
              <p role="alert" className="text-sm text-rose-700">
                {formError}
              </p>
            ) : null}
            <div className="flex justify-end gap-3 border-t border-slate-100 pt-5">
              <button
                className="button-secondary"
                type="button"
                onClick={() => {
                  setEditing(null)
                  setFormError('')
                }}
              >
                {t('common.cancel')}
              </button>
              <button
                className="button-primary"
                type="submit"
                disabled={
                  section === 'usuarios' &&
                  editing.company !== 'admin' &&
                  !editing.companyAccess?.length
                }
              >
                {section === 'usuarios' && !editing.id
                  ? t('demo.simulateInvitation')
                  : section === 'credenciais' && !editing.id
                    ? t('credentials.create')
                    : section === 'empresas' && !editing.id
                      ? t('companies.create')
                      : t('common.save')}
              </button>
            </div>
          </form>
        </Dialog>
      ) : null}
      {demoSecret ? (
        <Dialog
          title={t('credentials.secretTitle')}
          onClose={() => setDemoSecret(null)}
        >
          <p className="mb-4 text-sm text-slate-600">
            {t('demo.sampleSecretHint')}
          </p>
          <dl className="space-y-3 text-sm">
            <div>
              <dt className="font-semibold">{t('common.apiUsername')}</dt>
              <dd className="mt-1 rounded-lg bg-slate-50 p-3 font-mono">
                {demoSecret.username}
              </dd>
            </div>
            <div>
              <dt className="font-semibold">
                {t('credentials.passwordSetup')}
              </dt>
              <dd className="mt-1 rounded-lg bg-slate-50 p-3 font-mono break-all">
                {demoSecret.password}
              </dd>
            </div>
          </dl>
          <button
            className="button-primary mt-6 w-full"
            onClick={() => setDemoSecret(null)}
          >
            {t('common.close')}
          </button>
        </Dialog>
      ) : null}
    </>
  )
}
export default function DemoApp() {
  const { t } = useTranslation()
  const [rows, setRows] = useState(initialRows)
  const [demoUserId, setDemoUserId] = useState(
    () => localStorage.getItem(demoUserKey) ?? demoLoginUsers[0],
  )
  const [companySelection, setCompanySelection] = useState<{
    userId: string
    company: string
  }>()
  const [loggedOut, setLoggedOut] = useState(
    () => !(Number(localStorage.getItem(demoSessionKey)) > Date.now()),
  )
  const [rememberDemo, setRememberDemo] = useState(false)
  const [visible, setVisible] = useState(false)
  const demoUser =
    rows.usuarios.find((user) => user.id === demoUserId) ?? rows.usuarios[0]
  const isAdmin = demoUser.company === 'admin'
  const selectedCompany = !isAdmin
    ? companySelection?.userId === demoUserId &&
      demoUser.companyAccess?.includes(companySelection.company)
      ? companySelection.company
      : demoUser.companyAccess?.[0]
    : undefined
  const canManage = isAdmin || demoUser.company === 'operator'
  if (loggedOut)
    return (
      <div className="grid min-h-screen lg:grid-cols-2">
        <div className="hidden flex-col justify-between bg-brand-purple p-14 text-white lg:flex">
          <img src="/brand/21-mobile.png" alt="21 Mobile" className="w-48" />
          <div>
            <h1 className="max-w-md text-4xl font-semibold leading-tight">
              {t('auth.heroTitle')}
            </h1>
            <p className="mt-5 max-w-sm text-base leading-7 text-purple-100">
              {t('auth.heroDescription')}
            </p>
          </div>
          <p className="text-sm text-purple-200">{t('auth.brandFooter')}</p>
        </div>
        <main className="flex items-center justify-center px-6 py-10">
          <section className="w-full max-w-sm">
            <div className="mb-4 flex justify-end">
              <LanguageSwitcher />
            </div>
            <div className="mb-8 w-fit rounded-lg bg-brand-purple px-5 py-4 lg:hidden">
              <img
                src="/brand/21-mobile.png"
                alt="21 Mobile"
                className="h-10 w-auto"
              />
            </div>
            <h2 className="text-3xl font-bold">{t('auth.loginTitle')}</h2>
            <p className="mt-3 text-sm text-slate-500">{t('demo.loginHint')}</p>
            <form
              className="mt-8 space-y-5"
              onSubmit={(event) => {
                event.preventDefault()
                if (rememberDemo)
                  localStorage.setItem(
                    demoSessionKey,
                    String(Date.now() + demoRememberDurationMs),
                  )
                else localStorage.removeItem(demoSessionKey)
                localStorage.setItem(demoUserKey, demoUserId)
                setLoggedOut(false)
              }}
            >
              <label className="field">
                {t('demo.signInAs')}
                <AppSelect
                  value={demoUserId}
                  label={t('demo.signInAs')}
                  onValueChange={setDemoUserId}
                  options={demoLoginUsers.map((id) => {
                    const user = rows.usuarios.find((row) => row.id === id)!
                    return {
                      value: id,
                      label: `${user.name} · ${t(demoRoleKey(user.company))}`,
                    }
                  })}
                />
                <span className="text-xs font-normal text-slate-500">
                  {isAdmin
                    ? t('roles.globalAccess')
                    : t('demo.loginCompanyScope', {
                        companies: demoUser.companyAccess?.join(', '),
                      })}
                </span>
              </label>
              <label className="block text-sm font-semibold">
                {t('common.email')}
                <input
                  type="email"
                  autoComplete="username"
                  className="form-control mt-2 w-full"
                  value={demoUser.detail}
                  readOnly
                />
              </label>
              <label className="block text-sm font-semibold">
                {t('auth.password')}
                <div className="mt-2 flex">
                  <input
                    type={visible ? 'text' : 'password'}
                    autoComplete="current-password"
                    className="form-control min-w-0 flex-1"
                    defaultValue="demonstracao"
                  />
                  <button
                    type="button"
                    className="button-secondary ml-2"
                    onClick={() => setVisible((value) => !value)}
                  >
                    {visible ? t('auth.hidePassword') : t('auth.showPassword')}
                  </button>
                </div>
              </label>
              <label className="flex min-h-11 cursor-pointer items-center gap-3 text-sm text-slate-700">
                <input
                  type="checkbox"
                  className="h-4 w-4 accent-brand-purple"
                  checked={rememberDemo}
                  onChange={(event) => setRememberDemo(event.target.checked)}
                />
                {t('auth.rememberMe')}
              </label>
              <button className="button-primary w-full" type="submit">
                {t('demo.enterDemo')}
              </button>
            </form>
          </section>
        </main>
      </div>
    )
  return (
    <PanelShell
      name={demoUser.name}
      role={t(demoRoleKey(demoUser.company))}
      admin={isAdmin}
      demo
      companyView={
        demoUser.companyAccess && demoUser.companyAccess.length > 1
          ? {
              value: selectedCompany ?? demoUser.companyAccess[0],
              options: demoUser.companyAccess.map((company) => ({
                value: company,
                label: company,
              })),
              onChange: (company) =>
                setCompanySelection({
                  userId: demoUserId,
                  company: String(company),
                }),
            }
          : undefined
      }
      onLogout={() => {
        localStorage.removeItem(demoSessionKey)
        setLoggedOut(true)
      }}
    >
      {' '}
      <Routes>
        <Route
          path="/"
          element={
            isAdmin ? (
              <Overview />
            ) : (
              <ScopedOverview
                rows={rows}
                user={demoUser}
                selectedCompany={selectedCompany}
              />
            )
          }
        />
        <Route path="/perfil" element={<DemoProfile user={demoUser} />} />
        {(Object.keys(sections) as Section[])
          .filter(
            (section) =>
              isAdmin || (section !== 'usuarios' && section !== 'auditoria'),
          )
          .map((section) => (
            <Route
              key={section}
              path={`/${section}/:recordId?`}
              element={
                <Listing
                  key={`${section}-${selectedCompany ?? ''}`}
                  section={section}
                  rows={visibleDemoRows(
                    section,
                    rows[section],
                    demoUser,
                    selectedCompany,
                  )}
                  companies={visibleDemoRows(
                    'empresas',
                    rows.empresas,
                    demoUser,
                    selectedCompany,
                  )}
                  companyCredentials={visibleDemoRows(
                    'credenciais',
                    rows.credenciais,
                    demoUser,
                    selectedCompany,
                  )}
                  companyUsers={isAdmin ? rows.usuarios : []}
                  canManage={canManage}
                  admin={isAdmin}
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
                  onArchive={(id) =>
                    setRows((current) => ({
                      ...current,
                      [section]: current[section].filter(
                        (row) => row.id !== id,
                      ),
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
