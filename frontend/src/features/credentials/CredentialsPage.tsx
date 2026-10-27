import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { Dialog } from '../../components/admin/Dialog'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { listCredentials, archiveCredential, rotateCredential } from './api'
import type { Credential } from './api'
import { CredentialForm } from './CredentialForm'
import { CredentialTestDialog } from './CredentialTestDialog'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
export function CredentialsPage({
  companyId,
  companyName,
}: { companyId?: number; companyName?: string } = {}) {
  const { t } = useTranslation()
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
  const [testCredential, setTestCredential] = useState<Credential | null>(null)
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
  )
  const columns: ColumnDef<Credential>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'username', header: t('common.apiUsername') },
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
          initialCompanyId={companyId}
          initialCompanyName={companyName}
          onClose={() => setEditor(undefined)}
          onSecret={(username, password) => setSecret({ username, password })}
        />
      ) : null}
      {viewing ? (
        <RecordDetailsDialog
          title={viewing.credential.username}
          onClose={() => setViewing(null)}
          fields={[
            { label: 'ID', value: viewing.credential.id },
            {
              label: t('common.company'),
              value: viewing.credential.companyName,
            },
            {
              label: t('common.apiUsername'),
              value: viewing.credential.username,
            },
            { label: t('common.status'), value: t(`status.${viewing.status}`) },
            {
              label: t('credentials.expiresAt'),
              value: viewing.credential.expiresAt ? (
                <DateCell value={viewing.credential.expiresAt} />
              ) : (
                t('credentials.noExpiry')
              ),
            },
            {
              label: t('credentials.legacyId'),
              value: viewing.credential.legacyAccountId,
            },
            {
              label: t('credentials.providerToken'),
              value: viewing.credential.providerTokenConfigured
                ? t('credentials.tokenConfigured')
                : t('credentials.noToken'),
            },
            {
              label: t('credentials.ipAccess'),
              value: viewing.credential.whitelistEnabled
                ? viewing.credential.allowedIPs.join(', ')
                : t('credentials.unrestricted'),
            },
            {
              label: 'DLR',
              value: viewing.credential.callbacksEnabled
                ? viewing.credential.callbackUrl
                : t('credentials.disabled'),
            },
            {
              label: 'MO',
              value: viewing.credential.moCallbacksEnabled
                ? viewing.credential.moCallbackUrl
                : t('credentials.disabled'),
            },
            {
              label: t('common.createdAt'),
              value: <DateCell value={viewing.credential.createdAt} />,
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
          onClose={() => setSecret(null)}
        >
          <p className="mb-4 text-sm text-slate-600">
            {secret.password
              ? t('credentials.secretHint')
              : t('credentials.usernameHint')}
          </p>
          <label className="field">
            {t('credentials.username')}
            <input readOnly className="form-control" value={secret.username} />
          </label>
          {secret.password ? (
            <label className="field mt-4">
              {t('auth.password')}
              <input
                readOnly
                autoComplete="off"
                className="form-control font-mono"
                value={secret.password}
              />
            </label>
          ) : null}
          <button
            className="button-primary mt-6 w-full"
            onClick={() => setSecret(null)}
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
