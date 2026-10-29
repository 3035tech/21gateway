import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeft,
  Building2,
  CheckCircle2,
  KeyRound,
  Pencil,
  Send,
  ShieldCheck,
  TriangleAlert,
} from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { DateCell } from '../../components/admin/DateCell'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { errorMessage } from '../../lib/admin'
import { useUser } from '../auth/context'
import { CredentialsPage } from '../credentials/CredentialsPage'
import { CredentialTestDialog } from '../credentials/CredentialTestDialog'
import { CompanyForm } from './CompanyForm'
import { ZougDiagnosticDialog } from './ZougDiagnosticDialog'
import {
  getCompany,
  getCompanyByPublicRef,
  getCompanyTestSends,
  getCompanyHealth,
} from './api'
import { UsersPage } from '../users/UsersPage'
import { isPublicRef } from '../../lib/publicRef'

function CompanyTestHistory({ publicRef }: { publicRef: string }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['admin', 'company-test-sends', publicRef],
    queryFn: ({ signal }) => getCompanyTestSends(publicRef, signal),
  })
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="text-lg font-bold text-slate-900">
        {t('companyDetail.testHistory')}
      </h2>
      <p className="mt-1 text-sm text-slate-600">
        {t('companyDetail.testHistoryHint')}
      </p>
      {query.isPending ? (
        <p className="mt-4 text-sm" role="status">
          {t('common.loading')}
        </p>
      ) : null}
      {query.isError ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {errorMessage(query.error)}
        </p>
      ) : null}
      {query.data?.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">
          {t('companyDetail.noTests')}
        </p>
      ) : null}
      {query.data?.length ? (
        <ul className="mt-4 divide-y divide-slate-100">
          {query.data.map((send) => (
            <li
              key={send.correlationId}
              className="flex flex-wrap items-center justify-between gap-3 py-3 text-sm"
            >
              <div className="min-w-0">
                <p className="font-medium text-slate-900">
                  {send.destination.replace(/(\d{4})\d{4}(\d{4})$/, '$1••••$2')}
                </p>
                <p className="break-all text-xs text-slate-500">
                  {send.correlationId}
                </p>
                {send.providerMessageId ? (
                  <p className="break-all text-xs text-slate-500">
                    Zoug: {send.providerMessageId}
                  </p>
                ) : null}
                {send.failureCode ? (
                  <p className="text-xs text-amber-800">{send.failureCode}</p>
                ) : null}
              </div>
              <div className="flex items-center gap-3">
                <StatusBadge status={send.state} />
                <DateCell value={send.createdAt} />
              </div>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}

function CompanyIntegrationHealth({ publicRef }: { publicRef: string }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['admin', 'company-health', publicRef],
    queryFn: ({ signal }) => getCompanyHealth(publicRef, signal),
    refetchInterval: 60_000,
  })
  const data = query.isError ? undefined : query.data
  const metrics = data
    ? [
        {
          key: 'credentials',
          label: t('companyDetail.activeCredentials'),
          value: data.activeCredentials,
        },
        {
          key: 'dlr',
          label: t('companyDetail.dlrConfigured'),
          value: data.dlrConfigured,
        },
        {
          key: 'mo',
          label: t('companyDetail.moConfigured'),
          value: data.moConfigured,
        },
        {
          key: 'unknown',
          label: t('companyDetail.unknownSends'),
          value: data.unknownSends,
          warning: true,
        },
        {
          key: 'pending',
          label: t('companyDetail.pendingCallbacks'),
          value: data.pendingCallbacks,
        },
        {
          key: 'failed',
          label: t('companyDetail.failedCallbacks'),
          value: data.failedCallbacks,
          warning: true,
        },
      ]
    : []
  return (
    <section
      className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
      aria-label={t('companyDetail.integrationHealth')}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-bold text-slate-900">
            {t('companyDetail.integrationHealth')}
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            {t('companyDetail.integrationHealthHint')}
          </p>
        </div>
        <button
          type="button"
          className="button-secondary"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          {t('companyDetail.refreshHealth')}
        </button>
      </div>
      {query.isPending ? (
        <p className="mt-4 text-sm" role="status">
          {t('common.loading')}
        </p>
      ) : null}
      {query.isError ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {errorMessage(query.error)}
        </p>
      ) : null}
      {data ? (
        <>
          <dl className="mt-5 grid gap-px overflow-hidden rounded-xl border border-slate-200 bg-slate-200 sm:grid-cols-2 xl:grid-cols-3">
            {metrics.map(({ key, label, value, warning }) => (
              <div className="bg-white p-4" key={key}>
                <dt className="text-xs font-medium text-slate-600">{label}</dt>
                <dd
                  className={`mt-1 text-2xl font-bold tabular-nums ${warning && value > 0 ? 'text-amber-800' : 'text-slate-900'}`}
                >
                  {value}
                </dd>
              </div>
            ))}
          </dl>
          {data.expiringSoon > 0 ? (
            <p className="mt-4 text-sm text-amber-800">
              {t('companyDetail.expiringSoon', { count: data.expiringSoon })}
            </p>
          ) : null}
          <dl className="mt-5 grid gap-3 text-sm sm:grid-cols-2">
            {(
              [
                [t('companyDetail.lastSend'), data.lastSendAt],
                [t('companyDetail.lastProvider'), data.lastProviderAt],
                [t('companyDetail.lastClientDlr'), data.lastClientDlrAt],
                [t('companyDetail.lastClientMo'), data.lastClientMoAt],
              ] as const
            ).map(([label, value]) => (
              <div
                className="flex flex-wrap justify-between gap-2 border-b border-slate-100 pb-2"
                key={label}
              >
                <dt className="text-slate-600">{label}</dt>
                <dd className="font-medium text-slate-900">
                  {value ? (
                    <DateCell value={value} />
                  ) : (
                    t('companyDetail.noActivity')
                  )}
                </dd>
              </div>
            ))}
          </dl>
        </>
      ) : null}
    </section>
  )
}

export function CompanyDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams()
  const navigate = useNavigate()
  const legacyId = /^\d+$/.test(id ?? '') ? Number(id) : null
  const validCompanyRef =
    legacyId !== null
      ? Number.isSafeInteger(legacyId) && legacyId > 0
      : isPublicRef(id)
  const [editing, setEditing] = useState(false)
  const [testingToken, setTestingToken] = useState(false)
  const [diagnosing, setDiagnosing] = useState(false)
  const canEdit = useUser()?.role !== 'viewer'
  const canViewUsers = useUser()?.role === 'admin'
  const query = useQuery({
    queryKey: ['admin', 'company', id],
    queryFn: ({ signal }) =>
      legacyId !== null
        ? getCompany(legacyId, signal)
        : getCompanyByPublicRef(id!, signal),
    enabled: validCompanyRef,
  })
  useEffect(() => {
    if (legacyId !== null && query.data)
      navigate(`/empresas/${query.data.publicRef}`, { replace: true })
  }, [legacyId, navigate, query.data])
  if (!validCompanyRef) return <p role="alert">{t('app.notFoundTitle')}</p>
  return (
    <div className="space-y-8">
      <Link
        to="/empresas"
        className="button-ghost inline-flex items-center gap-2"
      >
        <ArrowLeft size={17} aria-hidden="true" /> {t('companyDetail.back')}
      </Link>
      {query.isPending ? <p role="status">{t('common.loading')}</p> : null}
      {query.isError ? (
        <p role="alert" className="rounded-xl bg-rose-50 p-4 text-rose-800">
          {errorMessage(query.error)}
        </p>
      ) : null}
      {query.data ? (
        <>
          <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
            <div className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-100 p-6">
              <div className="flex items-start gap-4">
                <span className="flex size-12 items-center justify-center rounded-xl bg-purple-50 text-brand-purple">
                  <Building2 size={25} aria-hidden="true" />
                </span>
                <div>
                  <p className="text-xs font-bold uppercase tracking-[0.16em] text-brand-purple">
                    {t('companyDetail.eyebrow')}
                  </p>
                  <h1 className="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">
                    {query.data.name}
                  </h1>
                </div>
              </div>
              {canEdit ? (
                <button
                  className="button-secondary"
                  onClick={() => setEditing(true)}
                >
                  <Pencil size={16} aria-hidden="true" />{' '}
                  {t('companies.editTitle')}
                </button>
              ) : null}
            </div>
            <dl className="grid gap-5 p-6 text-sm sm:grid-cols-3">
              <div>
                <dt className="text-slate-500">{t('common.reference')}</dt>
                <dd className="mt-1 font-semibold">
                  {query.data.reference || t('companyDetail.noReference')}
                </dd>
              </div>
              <div>
                <dt className="text-slate-500">{t('common.status')}</dt>
                <dd className="mt-1">
                  <StatusBadge
                    status={query.data.active ? 'active' : 'blocked'}
                  />
                </dd>
              </div>
              <div>
                <dt className="text-slate-500">{t('common.createdAt')}</dt>
                <dd className="mt-1 font-semibold">
                  <DateCell value={query.data.createdAt} />
                </dd>
              </div>
            </dl>
          </section>
          <section
            className={`rounded-2xl border p-6 ${query.data.providerTokenConfigured ? 'border-emerald-200 bg-emerald-50/70' : 'border-amber-200 bg-amber-50/70'}`}
          >
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="flex min-w-0 items-start gap-4">
                <span
                  className={`flex size-11 shrink-0 items-center justify-center rounded-xl ${query.data.providerTokenConfigured ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}
                >
                  <KeyRound size={21} aria-hidden="true" />
                </span>
                <div>
                  <h2 className="text-lg font-bold text-slate-900">
                    {t('companies.providerAccess')}
                  </h2>
                  <span
                    className={`mt-2 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold ${query.data.providerTokenConfigured ? 'bg-emerald-100 text-emerald-800' : 'bg-amber-100 text-amber-900'}`}
                  >
                    {query.data.providerTokenConfigured ? (
                      <CheckCircle2 size={14} aria-hidden="true" />
                    ) : (
                      <TriangleAlert size={14} aria-hidden="true" />
                    )}
                    {t(
                      query.data.providerTokenConfigured
                        ? 'companies.providerSaved'
                        : 'companies.providerNotSaved',
                    )}
                  </span>
                  <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-700">
                    {query.data.providerTokenConfigured
                      ? t('companies.providerConfigured')
                      : query.data.legacyProviderTokenFallback
                        ? t('companies.providerLegacy')
                        : t('companies.providerMissing')}
                  </p>
                  {query.data.providerTokenConfigured ? (
                    <p className="mt-2 flex items-center gap-1.5 text-xs text-slate-600">
                      <ShieldCheck size={15} aria-hidden="true" />
                      {t('companies.providerStoredHint')}
                    </p>
                  ) : null}
                </div>
              </div>
              {canEdit ? (
                <div className="flex flex-wrap gap-2">
                  {query.data.providerTokenConfigured && query.data.active ? (
                    <>
                      <button
                        className="button-primary"
                        onClick={() => setTestingToken(true)}
                      >
                        <Send size={16} aria-hidden="true" />{' '}
                        {t('companies.providerTest')}
                      </button>
                      {canViewUsers ? (
                        <button
                          className="button-secondary"
                          onClick={() => setDiagnosing(true)}
                        >
                          {t('companyDetail.diagnosticTitle')}
                        </button>
                      ) : null}
                    </>
                  ) : null}
                  <button
                    className="button-secondary"
                    onClick={() => setEditing(true)}
                  >
                    {t(
                      query.data.providerTokenConfigured
                        ? 'companies.providerReplace'
                        : 'companies.providerManage',
                    )}
                  </button>
                </div>
              ) : null}
            </div>
          </section>
          <CompanyIntegrationHealth publicRef={query.data.publicRef} />
          <CompanyTestHistory publicRef={query.data.publicRef} />
          <section aria-label={t('companyDetail.accessSection')}>
            <div className="mb-3 flex items-center gap-2 text-brand-purple">
              <KeyRound size={19} aria-hidden="true" />
              <span className="text-xs font-bold uppercase tracking-[0.16em]">
                {t('companyDetail.accessSection')}
              </span>
            </div>
            <CredentialsPage
              companyId={query.data.id}
              companyRef={query.data.publicRef}
              companyName={query.data.name}
            />
          </section>
          {canViewUsers ? (
            <section aria-label={t('users.companyUsersTitle')}>
              <UsersPage
                companyId={query.data.id}
                companyRef={query.data.publicRef}
                companyName={query.data.name}
              />
            </section>
          ) : null}
          {editing ? (
            <CompanyForm
              company={query.data}
              onClose={() => setEditing(false)}
            />
          ) : null}
          {testingToken ? (
            <CredentialTestDialog
              company={query.data}
              onClose={() => setTestingToken(false)}
            />
          ) : null}
          {diagnosing ? (
            <ZougDiagnosticDialog
              publicRef={query.data.publicRef}
              onClose={() => setDiagnosing(false)}
            />
          ) : null}
        </>
      ) : null}
    </div>
  )
}
