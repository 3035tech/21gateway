import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  ArrowUpRight,
  Building2,
  CalendarClock,
  KeyRound,
  RefreshCw,
  Send,
  Webhook,
  TriangleAlert,
} from 'lucide-react'
import { getSummary, getOperations } from './api'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
import { useCompanyView } from '../companies/CompanyViewContext'
import { useUser } from '../auth/context'
import { DateCell } from '../../components/admin/DateCell'

function olderThan(value: string | null, minutes: number, now: number) {
  return value !== null && now - new Date(value).getTime() >= minutes * 60_000
}

function OperationsAlerts() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['admin', 'operations'],
    queryFn: ({ signal }) => getOperations(signal),
    refetchInterval: 60_000,
  })
  const data = query.isError ? undefined : query.data
  const now = data ? new Date(data.observedAt).getTime() : 0
  const alerts: { message: string; to: string }[] = []
  if (data) {
    if (data.unresolvedSends > 0)
      alerts.push({
        message: t('overview.alertUnknownSends', {
          count: data.unresolvedSends,
        }),
        to: '/envios',
      })
    if (data.dlr.expiredLeases + data.mo.expiredLeases > 0)
      alerts.push({
        message: t('overview.alertExpiredLeases'),
        to: '/callbacks',
      })
    if (olderThan(data.dlr.oldestPendingAt, 15, now))
      alerts.push({ message: t('overview.alertDlrPending'), to: '/callbacks' })
    if (olderThan(data.mo.oldestPendingAt, 15, now))
      alerts.push({ message: t('overview.alertMoPending'), to: '/callbacks' })
    if (olderThan(data.dlr.oldestUnmatchedAt, 15, now))
      alerts.push({
        message: t('overview.alertDlrUnmatched'),
        to: '/callbacks',
      })
    if (olderThan(data.mo.oldestUnmatchedAt, 15, now))
      alerts.push({ message: t('overview.alertMoUnmatched'), to: '/callbacks' })
    if (olderThan(data.scheduler.oldestDueAt, 5, now))
      alerts.push({
        message: t('overview.alertSchedulerDue'),
        to: '/agendamentos',
      })
    if (data.dlr.failedLastHour + data.mo.failedLastHour > 0)
      alerts.push({
        message: t('overview.alertCallbackFailed'),
        to: '/callbacks',
      })
  }
  return (
    <section
      className="mb-6 rounded-2xl border border-slate-200 bg-white p-5"
      aria-label={t('overview.operationsHealth')}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-base font-bold text-slate-900">
          {t('overview.operationsHealth')}
        </h2>
        <span className="text-xs text-slate-500">
          {t('overview.operationsRefresh')}
          {data ? (
            <>
              {' '}
              · {t('overview.operationsObservedAt')}{' '}
              <DateCell value={data.observedAt} />
            </>
          ) : null}
        </span>
      </div>
      {query.isPending ? (
        <p className="mt-2 text-sm text-slate-600">{t('common.loading')}</p>
      ) : null}
      {query.isError ? (
        <p className="mt-2 text-sm text-amber-800" role="alert">
          {t('overview.operationsUnavailable')}
        </p>
      ) : null}
      {data && alerts.length === 0 ? (
        <p className="mt-2 text-sm text-emerald-800">
          {t('overview.operationsClear')}
        </p>
      ) : null}
      {alerts.length > 0 ? (
        <ul className="mt-3 grid gap-2" role="alert">
          {alerts.map((alert) => (
            <li
              key={alert.message}
              className="flex flex-wrap items-start gap-x-3 gap-y-1 border-l-2 border-amber-500 bg-amber-50 px-3 py-2 text-sm text-amber-900"
            >
              <span className="flex min-w-0 flex-1 items-start gap-2">
                <TriangleAlert
                  size={16}
                  className="mt-0.5 shrink-0"
                  aria-hidden="true"
                />
                {alert.message}
              </span>
              <Link
                to={alert.to}
                className="ml-6 font-semibold underline underline-offset-2 hover:text-amber-950 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-800"
              >
                {t('overview.inspectAlert')}
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}

export function OverviewPage() {
  const { t, i18n } = useTranslation()
  const companyId = useCompanyView()
  const isAdmin = useUser()?.role === 'admin'
  const number = (value: number) => value.toLocaleString(i18n.language)
  const query = useQuery({
    queryKey: ['admin', 'summary', companyId],
    queryFn: ({ signal }) => getSummary(companyId, signal),
  })
  const data = query.data
  return (
    <>
      <header className="mb-7 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">
            {t('nav.overview')}
          </h1>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            {t('overview.description')}
          </p>
        </div>
        <button
          className="button-secondary"
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <RefreshCw size={16} aria-hidden="true" /> {t('overview.refresh')}
        </button>
      </header>
      {isAdmin && !companyId ? <OperationsAlerts /> : null}
      {query.isPending ? (
        <p role="status" className="rounded-xl bg-white p-8">
          {t('overview.loading')}
        </p>
      ) : query.isError ? (
        <p role="alert" className="rounded-xl bg-rose-50 p-6 text-rose-800">
          {errorMessage(query.error)}
        </p>
      ) : data ? (
        <div className="space-y-6">
          <section className="overflow-hidden rounded-2xl bg-brand-purple text-white">
            <div className="grid lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
              <div className="p-6 sm:p-8">
                <h2 className="flex items-center gap-2 text-sm font-semibold text-purple-100">
                  <CalendarClock size={19} aria-hidden="true" />{' '}
                  {t('overview.scheduleQueue')}
                </h2>
                <div className="mt-5 flex flex-wrap items-end gap-x-5 gap-y-2">
                  <strong className="text-6xl font-bold leading-none tabular-nums tracking-tight sm:text-7xl">
                    {number(data.scheduledDue)}
                  </strong>
                  <p className="max-w-56 pb-1 text-sm leading-6 text-purple-100">
                    {t('overview.overdue')}
                  </p>
                </div>
                <Link
                  to="/agendamentos"
                  className="mt-7 inline-flex min-h-11 items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-bold text-brand-purple hover:bg-purple-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
                >
                  {t('overview.viewSchedules')}{' '}
                  <ArrowUpRight size={17} aria-hidden="true" />
                </Link>
              </div>
              <div className="grid grid-cols-3 border-t border-white/20 bg-white/10 lg:grid-cols-1 lg:border-l lg:border-t-0">
                {[
                  [t('overview.next24h'), data.scheduledNext24h],
                  [t('overview.processing'), data.scheduledActive],
                  [t('overview.queuedTotal'), data.scheduledQueued],
                ].map(([label, value]) => (
                  <div
                    key={label}
                    className="flex flex-col justify-center border-r border-white/15 p-4 last:border-r-0 lg:border-b lg:border-r-0 lg:px-7 lg:py-5 lg:last:border-b-0"
                  >
                    <span className="text-xs leading-5 text-purple-100 sm:text-sm">
                      {label}
                    </span>
                    <strong className="mt-1 text-2xl font-bold tabular-nums sm:text-3xl">
                      {number(Number(value))}
                    </strong>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <div className="grid gap-5 lg:grid-cols-2">
            <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="flex items-center gap-2 text-lg font-bold text-slate-950">
                    <Send
                      size={20}
                      className="text-brand-purple"
                      aria-hidden="true"
                    />{' '}
                    {t('overview.recordedMessages')}
                  </h2>
                  <p className="mt-1 text-xs text-slate-500">
                    {t('overview.historyTotal')}
                  </p>
                </div>
                <Link
                  to="/envios"
                  className="inline-flex min-h-11 items-center gap-1 text-sm font-bold text-brand-purple hover:underline"
                >
                  {t('overview.open')}{' '}
                  <ArrowUpRight size={16} aria-hidden="true" />
                </Link>
              </div>
              <dl className="mt-6 grid grid-cols-3 gap-3 border-t border-slate-100 pt-6">
                {[
                  [t('overview.accepted'), data.accepted],
                  [t('overview.rejected'), data.rejected],
                  [t('overview.unknown'), data.unknown],
                ].map(([label, value]) => (
                  <div key={label}>
                    <dt className="text-xs text-slate-500 sm:text-sm">
                      {label}
                    </dt>
                    <dd className="mt-1 text-2xl font-bold tabular-nums text-slate-900">
                      {number(Number(value))}
                    </dd>
                  </div>
                ))}
              </dl>
              <p className="mt-6 rounded-lg bg-amber-50 px-4 py-3 text-xs leading-5 text-amber-900">
                {t('overview.deliveryWarning')}
              </p>
            </section>

            <section className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-7">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="flex items-center gap-2 text-lg font-bold text-slate-950">
                    <Webhook
                      size={20}
                      className="text-brand-purple"
                      aria-hidden="true"
                    />{' '}
                    {t('overview.clientNotifications')}
                  </h2>
                  <p className="mt-1 text-xs text-slate-500">
                    {t('overview.callbackJobs')}
                  </p>
                </div>
                <Link
                  to="/callbacks"
                  className="inline-flex min-h-11 items-center gap-1 text-sm font-bold text-brand-purple hover:underline"
                >
                  {t('overview.viewDlr')}{' '}
                  <ArrowUpRight size={16} aria-hidden="true" />
                </Link>
              </div>
              <div className="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-6">
                {[
                  [
                    t('overview.dlrStatus'),
                    data.pendingCallbacks,
                    data.failedCallbacks,
                  ],
                  [
                    t('overview.moReplies'),
                    data.pendingMoCallbacks,
                    data.failedMoCallbacks,
                  ],
                ].map(([label, pending, failed]) => (
                  <div key={label}>
                    <h3 className="text-sm font-bold text-slate-700">
                      {label}
                    </h3>
                    <p className="mt-2 text-2xl font-bold tabular-nums text-slate-900">
                      {number(Number(pending))}{' '}
                      <span className="text-xs font-normal text-slate-500">
                        {t('overview.pending')}
                      </span>
                    </p>
                    <p className="mt-1 text-sm text-rose-700">
                      {t('overview.failedCount', {
                        value: number(Number(failed)),
                      })}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          </div>

          <section className="flex flex-wrap items-center gap-x-6 gap-y-3 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-sm">
            <span className="font-bold text-slate-900">
              {t('overview.activeAccess')}
            </span>
            <span className="inline-flex items-center gap-2 text-slate-700">
              <Building2
                size={17}
                className="text-brand-purple"
                aria-hidden="true"
              />{' '}
              {t('overview.companyCount', { value: number(data.companies) })}
            </span>
            <span className="inline-flex items-center gap-2 text-slate-700">
              <KeyRound
                size={17}
                className="text-brand-purple"
                aria-hidden="true"
              />{' '}
              {t('overview.credentialCount', {
                value: number(data.activeCredentials),
              })}
            </span>
            <Link
              to="/credenciais"
              className="ml-auto inline-flex min-h-11 items-center gap-1 font-bold text-brand-purple hover:underline"
            >
              {t('overview.manageAccess')}{' '}
              <ArrowUpRight size={16} aria-hidden="true" />
            </Link>
          </section>
        </div>
      ) : null}
    </>
  )
}
