import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Building2, CheckCircle2, KeyRound, Pencil, ShieldCheck, TriangleAlert } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { DateCell } from '../../components/admin/DateCell'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { errorMessage } from '../../lib/admin'
import { useUser } from '../auth/context'
import { CredentialsPage } from '../credentials/CredentialsPage'
import { CompanyForm } from './CompanyForm'
import { getCompany } from './api'
import { UsersPage } from '../users/UsersPage'

export function CompanyDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams()
  const companyId = Number(id)
  const [editing, setEditing] = useState(false)
  const canEdit = useUser()?.role !== 'viewer'
  const canViewUsers = useUser()?.role === 'admin'
  const query = useQuery({
    queryKey: ['admin', 'company', companyId],
    queryFn: ({ signal }) => getCompany(companyId, signal),
    enabled: Number.isSafeInteger(companyId) && companyId > 0,
  })
  if (!Number.isSafeInteger(companyId) || companyId < 1)
    return <p role="alert">{t('app.notFoundTitle')}</p>
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
          <section className={`rounded-2xl border p-6 ${query.data.providerTokenConfigured ? 'border-emerald-200 bg-emerald-50/70' : 'border-amber-200 bg-amber-50/70'}`}>
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="flex min-w-0 items-start gap-4">
                <span className={`flex size-11 shrink-0 items-center justify-center rounded-xl ${query.data.providerTokenConfigured ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                  <KeyRound size={21} aria-hidden="true" />
                </span>
                <div>
                  <h2 className="text-lg font-bold text-slate-900">
                    {t('companies.providerAccess')}
                  </h2>
                  <span className={`mt-2 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold ${query.data.providerTokenConfigured ? 'bg-emerald-100 text-emerald-800' : 'bg-amber-100 text-amber-900'}`}>
                    {query.data.providerTokenConfigured ? <CheckCircle2 size={14} aria-hidden="true" /> : <TriangleAlert size={14} aria-hidden="true" />}
                    {t(query.data.providerTokenConfigured ? 'companies.providerSaved' : 'companies.providerNotSaved')}
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
                <button
                  className="button-secondary shrink-0"
                  onClick={() => setEditing(true)}
                >
                  {t(query.data.providerTokenConfigured ? 'companies.providerReplace' : 'companies.providerManage')}
                </button>
              ) : null}
            </div>
          </section>
          <section aria-label={t('companyDetail.accessSection')}>
            <div className="mb-3 flex items-center gap-2 text-brand-purple">
              <KeyRound size={19} aria-hidden="true" />
              <span className="text-xs font-bold uppercase tracking-[0.16em]">
                {t('companyDetail.accessSection')}
              </span>
            </div>
            <CredentialsPage
              companyId={companyId}
              companyName={query.data.name}
            />
          </section>
          {canViewUsers ? (
            <section aria-label={t('users.companyUsersTitle')}>
              <UsersPage companyId={companyId} companyName={query.data.name} />
            </section>
          ) : null}
          {editing ? (
            <CompanyForm
              company={query.data}
              onClose={() => setEditing(false)}
            />
          ) : null}
        </>
      ) : null}
    </div>
  )
}
