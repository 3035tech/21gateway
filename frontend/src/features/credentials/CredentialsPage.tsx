import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import { useCompanyView } from '../companies/CompanyViewContext'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { Dialog } from '../../components/admin/Dialog'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import {
  listCredentials,
  getCredential,
  archiveCredential,
  rotateCredential,
} from './api'
import type { Credential } from './api'
import { CredentialForm } from './CredentialForm'
import { CredentialTestDialog } from './CredentialTestDialog'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { errorMessage } from '../../lib/admin'
import { Check, Copy } from 'lucide-react'
export function CredentialsPage({
  companyId,
  companyName,
}: { companyId?: number; companyName?: string } = {}) {
  const selectedCompanyId = useCompanyView()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const fromCompany = searchParams.get('fromCompany')
  const { recordId } = useParams()
  const linkedRecordId = Number(recordId)
  const validRecordId =
    !!recordId && Number.isSafeInteger(linkedRecordId) && linkedRecordId > 0
  const linkedRecord = useQuery({
    queryKey: ['admin', 'credential', linkedRecordId],
    queryFn: ({ signal }) => getCredential(linkedRecordId, signal),
    enabled: validRecordId,
  })
  const role = useUser()?.role
  const writable = role !== 'viewer'
  const canTest = role === 'admin'
  const [editor, setEditor] = useState<Credential | null | undefined>()
  const [viewing, setViewing] = useState<{
    credential: Credential
    status: 'active' | 'blocked' | 'expired'
  } | null>(null)
  const [action, setAction] = useState<{
    credential: Credential
    rotate: boolean
  } | null>(null)
  const [secret, setSecret] = useState<{
    username: string
    password: string
  } | null>(null)
  const [copiedSecret, setCopiedSecret] = useState<'username' | 'password' | 'both' | null>(null)
  const [copyFailed, setCopyFailed] = useState(false)
  async function copySecret(value: string, part: 'username' | 'password' | 'both') {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedSecret(part)
      setCopyFailed(false)
    } catch {
      setCopyFailed(true)
    }
  }
  function closeSecret() {
    setSecret(null)
    setCopiedSecret(null)
    setCopyFailed(false)
  }
  const [testCredential, setTestCredential] = useState<Credential | null>(null)
  const [openedAt] = useState(() => Date.now())
  const linkedCredential = linkedRecord.data
  const activeViewing =
    viewing ??
    (linkedCredential
      ? {
          credential: linkedCredential,
          status: !linkedCredential.active
            ? ('blocked' as const)
            : linkedCredential.expiresAt &&
                new Date(linkedCredential.expiresAt).getTime() <= openedAt
              ? ('expired' as const)
              : ('active' as const),
        }
      : null)
  const mutation = useAdminMutation<{
    credential: Credential
    rotate: boolean
  }>(
    async (value) => {
      if (value.rotate) {
        const result = await rotateCredential(value.credential.id)
        setSecret({
          username: result.credential.username,
          password: result.password,
        })
      } else await archiveCredential(value.credential.id)
    },
    () => setAction(null),
    (value) => (value.rotate ? 'feedback.rotated' : 'feedback.archived'),
  )
  const columns: ColumnDef<Credential>[] = [
    { accessorKey: 'id', header: 'ID' },
    {
      accessorKey: 'username',
      header: t('common.apiUsername'),
      cell: ({ row }) =>
        companyId ? (
          <Link
            className="font-semibold text-brand-purple underline-offset-2 hover:underline"
            to={`/credenciais/${row.original.id}?fromCompany=${companyId}`}
          >
            {row.original.username}
          </Link>
        ) : (
          row.original.username
        ),
    },
    { accessorKey: 'companyName', header: t('common.company') },
    {
      accessorKey: 'providerTokenConfigured',
      header: 'Zoug',
      cell: ({ row }) =>
        row.original.providerTokenConfigured
          ? t('credentials.tokenConfigured')
          : t('credentials.noToken'),
    },
    {
      accessorKey: 'whitelistEnabled',
      header: t('credentials.ipAccess'),
      cell: ({ row }) =>
        row.original.whitelistEnabled
          ? t('credentials.ipCount', { count: row.original.allowedIPs.length })
          : t('credentials.unrestricted'),
    },
    {
      accessorKey: 'active',
      header: t('common.status'),
      cell: ({ row }) => (
        <StatusBadge
          status={
            !row.original.active
              ? 'blocked'
              : row.original.expiresAt &&
                  new Date(row.original.expiresAt).getTime() <= Date.now()
                ? 'expired'
                : 'active'
          }
        />
      ),
    },
    {
      accessorKey: 'expiresAt',
      header: t('credentials.expiresAt'),
      cell: ({ row }) =>
        row.original.expiresAt ? (
          <DateCell value={row.original.expiresAt} />
        ) : (
          t('credentials.noExpiry')
        ),
    },
    {
      accessorKey: 'callbacksEnabled',
      id: 'notifications',
      header: t('credentials.notificationsSection'),
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-1.5">
          {[
            ['DLR', row.original.callbacksEnabled],
            ['MO', row.original.moCallbacksEnabled],
          ].map(([label, enabled]) => (
            <span
              key={String(label)}
              className={`inline-flex rounded-md px-2 py-1 text-xs font-semibold ${enabled ? 'bg-purple-50 text-brand-purple' : 'bg-slate-100 text-slate-500'}`}
              title={
                enabled
                  ? t('credentials.notificationEnabled', { label })
                  : t('credentials.notificationDisabled', { label })
              }
            >
              {label}{' '}
              <span className="sr-only">
                {enabled ? t('credentials.enabled') : t('credentials.disabled')}
              </span>
            </span>
          ))}
        </div>
      ),
    },
    {
      accessorKey: 'createdAt',
      header: t('common.createdAt'),
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
  ]
  columns.push({
    id: 'actions',
    header: t('common.actions'),
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex flex-nowrap gap-1">
        <button
          className="table-action"
          onClick={() =>
            setViewing({
              credential: row.original,
              status: !row.original.active
                ? 'blocked'
                : row.original.expiresAt &&
                    new Date(row.original.expiresAt).getTime() <= Date.now()
                  ? 'expired'
                  : 'active',
            })
          }
        >
          {t('common.view')}
        </button>
        {writable ? (
          <>
            {canTest ? (
              <button
                className="table-action"
                onClick={() => setTestCredential(row.original)}
              >
                {t('credentials.test')}
              </button>
            ) : null}
            <button
              className="table-action"
              onClick={() => setEditor(row.original)}
            >
              {t('common.edit')}
            </button>
            <button
              className="table-action"
              onClick={() => {
                mutation.reset()
                setAction({ credential: row.original, rotate: true })
              }}
            >
              {t('credentials.rotatePassword')}
            </button>
            <button
              className="table-action-danger"
              onClick={() => {
                mutation.reset()
                setAction({ credential: row.original, rotate: false })
              }}
            >
              {t('common.archive')}
            </button>
          </>
        ) : null}
      </div>
    ),
  })
  return (
    <>
      <Listing
        title={
          companyId ? t('companyDetail.credentialsTitle') : t('nav.credentials')
        }
        description={
          companyId
            ? t('companyDetail.credentialsHint')
            : t('credentials.listDescription')
        }
        searchPlaceholder={t('common.apiUsername')}
        queryKey="credentials"
        fixedCompanyId={companyId}
        headingLevel={companyId ? 2 : 1}
        columns={columns}
        load={listCredentials}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel={t('credentials.new')}
        filterName="active"
        filterOptions={[
          ['true', t('companies.activePlural')],
          ['false', t('companies.blockedPlural')],
        ]}
      />
      {editor !== undefined ? (
        <CredentialForm
          credential={editor ?? undefined}
          initialCompanyId={companyId ?? selectedCompanyId}
          initialCompanyName={companyName}
          onClose={() => setEditor(undefined)}
          onSecret={(username, password) => setSecret({ username, password })}
        />
      ) : null}
      {recordId && !validRecordId ? (
        <p role="alert" className="mt-4 text-rose-700">
          {t('app.notFoundTitle')}
        </p>
      ) : null}
      {validRecordId && linkedRecord.isPending ? (
        <p role="status" className="mt-4 text-slate-600">
          {t('common.loading')}
        </p>
      ) : null}
      {validRecordId && linkedRecord.isError ? (
        <p role="alert" className="mt-4 text-rose-700">
          {errorMessage(linkedRecord.error)}
        </p>
      ) : null}
      {activeViewing ? (
        <RecordDetailsDialog
          title={activeViewing.credential.username}
          onClose={() => {
            if (viewing) setViewing(null)
            else
              navigate(
                fromCompany ? `/empresas/${fromCompany}` : '/credenciais',
              )
          }}
          fields={[
            { label: 'ID', value: activeViewing.credential.id },
            {
              label: t('common.company'),
              value: activeViewing.credential.companyName,
            },
            {
              label: t('common.apiUsername'),
              value: activeViewing.credential.username,
            },
            {
              label: t('common.status'),
              value: t(`status.${activeViewing.status}`),
            },
            {
              label: t('credentials.expiresAt'),
              value: activeViewing.credential.expiresAt ? (
                <DateCell value={activeViewing.credential.expiresAt} />
              ) : (
                t('credentials.noExpiry')
              ),
            },
            {
              label: t('credentials.legacyId'),
              value: activeViewing.credential.legacyAccountId,
            },
            {
              label: t('credentials.providerToken'),
              value: activeViewing.credential.providerTokenConfigured
                ? t('credentials.tokenConfigured')
                : t('credentials.noToken'),
            },
            {
              label: t('credentials.ipAccess'),
              value: activeViewing.credential.whitelistEnabled
                ? activeViewing.credential.allowedIPs.join(', ')
                : t('credentials.unrestricted'),
            },
            {
              label: 'DLR',
              value: activeViewing.credential.callbacksEnabled
                ? activeViewing.credential.callbackUrl
                : t('credentials.disabled'),
            },
            {
              label: 'MO',
              value: activeViewing.credential.moCallbacksEnabled
                ? activeViewing.credential.moCallbackUrl
                : t('credentials.disabled'),
            },
            {
              label: t('common.createdAt'),
              value: <DateCell value={activeViewing.credential.createdAt} />,
            },
          ]}
        />
      ) : null}
      {action ? (
        <Confirmation
          title={
            action.rotate
              ? t('credentials.rotateTitle')
              : t('credentials.archiveTitle')
          }
          description={
            action.rotate
              ? t('credentials.rotateDescription', {
                  username: action.credential.username,
                })
              : t('credentials.archiveDescription', {
                  username: action.credential.username,
                })
          }
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(action)}
          onClose={() => setAction(null)}
        />
      ) : null}
      {secret && editor === undefined && action === null ? (
        <Dialog
          title={
            secret.password
              ? t('credentials.secretTitle')
              : t('credentials.usernameTitle')
          }
          onClose={closeSecret}
        >
          <p className="mb-4 text-sm text-slate-600">
            {secret.password
              ? t('credentials.secretHint')
              : t('credentials.usernameHint')}
          </p>
          <div className="field">
            <label htmlFor="credential-secret-username">{t('credentials.username')}</label>
            <div className="flex gap-2">
              <input id="credential-secret-username" readOnly className="form-control min-w-0 flex-1" value={secret.username} />
              <button type="button" className="button-secondary shrink-0" onClick={() => copySecret(secret.username, 'username')} aria-label={t('credentials.copyUsername')}>
                {copiedSecret === 'username' ? <Check size={17} aria-hidden="true" /> : <Copy size={17} aria-hidden="true" />}
              </button>
            </div>
          </div>
          {secret.password ? (
            <div className="field mt-4">
              <label htmlFor="credential-secret-password">{t('auth.password')}</label>
              <div className="flex gap-2">
                <input id="credential-secret-password" readOnly autoComplete="off" className="form-control min-w-0 flex-1 font-mono" value={secret.password} />
                <button type="button" className="button-secondary shrink-0" onClick={() => copySecret(secret.password, 'password')} aria-label={t('credentials.copyPassword')}>
                  {copiedSecret === 'password' ? <Check size={17} aria-hidden="true" /> : <Copy size={17} aria-hidden="true" />}
                </button>
              </div>
            </div>
          ) : null}
          {secret.password ? (
            <button type="button" className="button-secondary mt-4 w-full" onClick={() => copySecret(`${t('credentials.username')}: ${secret.username}\n${t('auth.password')}: ${secret.password}`, 'both')}>
              {copiedSecret === 'both' ? <Check size={17} aria-hidden="true" /> : <Copy size={17} aria-hidden="true" />}
              {t(copiedSecret === 'both' ? 'credentials.copiedAccess' : 'credentials.copyAccess')}
            </button>
          ) : null}
          {copyFailed ? <p role="alert" className="mt-3 text-sm text-rose-700">{t('credentials.copyFailed')}</p> : null}
          <button
            className="button-primary mt-6 w-full"
            onClick={closeSecret}
          >
            {t('credentials.savedCredential')}
          </button>
        </Dialog>
      ) : null}
      {testCredential ? (
        <CredentialTestDialog
          credential={testCredential}
          onClose={() => setTestCredential(null)}
        />
      ) : null}
    </>
  )
}
