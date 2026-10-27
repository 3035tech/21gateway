import { DateCell } from '../../components/admin/DateCell'
import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useFeedback } from '../../components/admin/feedbackContext'
import { useUser } from '../auth/context'
import { errorMessage, roleLabels } from '../../lib/admin'
import {
  listUsers,
  getUser,
  getUserByPublicRef,
  archiveUser,
  resendInvitation,
} from './api'
import type { SystemUser } from './api'
import { UserForm } from './UserForm'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { useCompanyReturnRef } from '../companies/useCompanyReturnRef'
import { isPublicRef } from '../../lib/publicRef'
export function UsersPage({
  companyId,
  companyRef,
  companyName,
}: { companyId?: number; companyRef?: string; companyName?: string } = {}) {
  const { t } = useTranslation()
  const feedback = useFeedback()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const fromCompany = searchParams.get('fromCompany')
  const { publicRef: returnCompanyRef, ready: companyRefReady } =
    useCompanyReturnRef(fromCompany)
  const { recordId } = useParams()
  const legacyId = /^\d+$/.test(recordId ?? '') ? Number(recordId) : null
  const validRecordId =
    legacyId === null
      ? isPublicRef(recordId)
      : Number.isSafeInteger(legacyId) && legacyId > 0
  const linkedRecord = useQuery({
    queryKey: ['admin', 'user', recordId],
    queryFn: ({ signal }) =>
      legacyId !== null
        ? getUser(legacyId, signal)
        : getUserByPublicRef(recordId!, signal),
    enabled: validRecordId,
  })
  useEffect(() => {
    const publicRef =
      legacyId !== null ? linkedRecord.data?.publicRef : recordId
    if (
      publicRef &&
      companyRefReady &&
      (legacyId !== null || fromCompany !== returnCompanyRef)
    )
      navigate(
        `/usuarios/${publicRef}${returnCompanyRef ? `?fromCompany=${returnCompanyRef}` : ''}`,
        { replace: true },
      )
  }, [
    legacyId,
    linkedRecord.data,
    navigate,
    recordId,
    fromCompany,
    returnCompanyRef,
    companyRefReady,
  ])
  const user = useUser()
  const writable = user?.role === 'admin'
  const [editor, setEditor] = useState<SystemUser | null | undefined>()
  const [viewing, setViewing] = useState<SystemUser | null>(null)
  const [archive, setArchive] = useState<SystemUser | null>(null)
  const mutation = useAdminMutation(
    archiveUser,
    () => setArchive(null),
    'feedback.archived',
  )
  const [inviteSent, setInviteSent] = useState(false)
  const resend = useMutation({
    mutationFn: resendInvitation,
    onSuccess: () => {
      setInviteSent(true)
      feedback.success(t('feedback.inviteSent'))
    },
    onError: (error) => feedback.error(errorMessage(error)),
  })
  const columns: ColumnDef<SystemUser>[] = [
    {
      accessorKey: 'name',
      header: t('common.name'),
      cell: ({ row }) =>
        companyId ? (
          <Link
            className="font-semibold text-brand-purple underline-offset-2 hover:underline"
            to={`/usuarios/${row.original.publicRef}${companyRef ? `?fromCompany=${companyRef}` : ''}`}
          >
            {row.original.name}
          </Link>
        ) : (
          row.original.name
        ),
    },
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
      <div className="flex w-max items-center gap-1">
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
        title={companyId ? t('users.companyUsersTitle') : t('nav.users')}
        description={
          companyId
            ? t('users.companyUsersDescription', { name: companyName })
            : t('users.description')
        }
        searchPlaceholder={t('users.searchPlaceholder')}
        queryKey="users"
        fixedCompanyId={companyId}
        headingLevel={companyId ? 2 : 1}
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
          initialCompanyId={companyId}
          onClose={() => setEditor(undefined)}
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
      {(viewing ?? linkedRecord.data) ? (
        <RecordDetailsDialog
          title={(viewing ?? linkedRecord.data)!.name}
          onClose={() => {
            if (viewing) setViewing(null)
            else
              navigate(
                returnCompanyRef
                  ? `/empresas/${returnCompanyRef}`
                  : '/usuarios',
              )
          }}
          fields={[
            {
              label: t('common.name'),
              value: (viewing ?? linkedRecord.data)!.name,
            },
            {
              label: t('common.email'),
              value: (viewing ?? linkedRecord.data)!.email,
            },
            {
              label: t('roles.label'),
              value: t(roleLabels[(viewing ?? linkedRecord.data)!.role]),
            },
            {
              label: t('users.companyAccess'),
              value:
                (viewing ?? linkedRecord.data)!.role === 'admin'
                  ? t('roles.globalAccess')
                  : (viewing ?? linkedRecord.data)!.companyIds
                      .map((id) => t('common.companyNumber', { id }))
                      .join(', '),
            },
            {
              label: t('common.status'),
              value: t(
                (viewing ?? linkedRecord.data)!.active
                  ? 'status.active'
                  : 'status.blocked',
              ),
            },
            {
              label: t('common.createdAt'),
              value: (
                <DateCell value={(viewing ?? linkedRecord.data)!.createdAt} />
              ),
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
