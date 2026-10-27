import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { errorMessage, roleLabels } from '../../lib/admin'
import { listUsers, archiveUser, resendInvitation } from './api'
import type { SystemUser } from './api'
import { UserForm } from './UserForm'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
export function UsersPage() {
  const { t } = useTranslation()
  const user = useUser()
  const writable = user?.role === 'admin'
  const [editor, setEditor] = useState<SystemUser | null | undefined>()
  const [viewing, setViewing] = useState<SystemUser | null>(null)
  const [archive, setArchive] = useState<SystemUser | null>(null)
  const mutation = useAdminMutation(archiveUser, () => setArchive(null))
  const [inviteSent, setInviteSent] = useState(false)
  const resend = useMutation({
    mutationFn: resendInvitation,
    onSuccess: () => setInviteSent(true),
  })
  const columns: ColumnDef<SystemUser>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'name', header: t('common.name') },
    { accessorKey: 'email', header: t('common.email') },
    {
      accessorKey: 'role',
      header: t('roles.label'),
      cell: ({ row }) => t(roleLabels[row.original.role]),
    },
    {
      accessorKey: 'active',
      header: t('common.status'),
      cell: ({ row }) => (
        <StatusBadge status={row.original.active ? 'active' : 'blocked'} />
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
      <div className="flex gap-1">
        <button
          className="table-action"
          onClick={() => setViewing(row.original)}
        >
          {t('common.view')}
        </button>
        {writable ? (
          <>
            <button
              className="table-action"
              onClick={() => setEditor(row.original)}
            >
              {t('common.edit')}
            </button>
            <button
              className="table-action"
              disabled={resend.isPending}
              onClick={() => {
                setInviteSent(false)
                resend.mutate(row.original.id)
              }}
            >
              {t('users.resendInvitation')}
            </button>
            <button
              className="table-action-danger"
              onClick={() => setArchive(row.original)}
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
      {inviteSent ? (
        <p
          role="status"
          className="mb-4 rounded-lg bg-emerald-50 p-3 text-sm text-emerald-800"
        >
          {t('users.invitationSent')}
        </p>
      ) : null}
      {resend.isError ? (
        <p
          role="alert"
          className="mb-4 rounded-lg bg-rose-50 p-3 text-sm text-rose-700"
        >
          {errorMessage(resend.error)}
        </p>
      ) : null}
      <Listing
        title={t('nav.users')}
        description={t('users.description')}
        searchPlaceholder={t('users.searchPlaceholder')}
        queryKey="users"
        columns={columns}
        load={listUsers}
        onCreate={writable ? () => setEditor(null) : undefined}
        createLabel={t('users.new')}
        filterName="active"
        filterOptions={[
          ['true', t('users.activePlural')],
          ['false', t('users.blockedPlural')],
        ]}
      />
      {editor !== undefined ? (
        <UserForm
          user={editor ?? undefined}
          onClose={() => setEditor(undefined)}
        />
      ) : null}
      {viewing ? (
        <RecordDetailsDialog
          title={viewing.name}
          onClose={() => setViewing(null)}
          fields={[
            { label: 'ID', value: viewing.id },
            { label: t('common.name'), value: viewing.name },
            { label: t('common.email'), value: viewing.email },
            { label: t('roles.label'), value: t(roleLabels[viewing.role]) },
            {
              label: t('common.status'),
              value: t(viewing.active ? 'status.active' : 'status.blocked'),
            },
            {
              label: t('common.createdAt'),
              value: <DateCell value={viewing.createdAt} />,
            },
          ]}
        />
      ) : null}
      {archive ? (
        <Confirmation
          title={t('users.archiveTitle')}
          description={t('users.archiveDescription', { name: archive.name })}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(archive.id)}
          onClose={() => setArchive(null)}
        />
      ) : null}
    </>
  )
}
