import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { DateCell } from '../../components/admin/DateCell'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { errorMessage } from '../../lib/admin'
import { listAudit } from './api'
import type { Audit, AuditListParams } from './api'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { AppSelect } from '../../components/admin/AppSelect'
import { TableIconButton } from '../../components/admin/TableIconAction'
import { Eye, RotateCcw } from 'lucide-react'

const actions = [
  'created',
  'updated',
  'archived',
  'password_rotated',
  'login',
  'callback_retried',
]
const resources = ['companies', 'users', 'credentials', 'callbacks']
function auditColumns(
  t: (key: string, options?: Record<string, unknown>) => string,
  onView: (row: Audit) => void,
): ColumnDef<Audit>[] {
  return [
    { accessorKey: 'id', header: 'ID' },
    {
      accessorKey: 'actorName',
      header: t('audit.actor'),
      cell: ({ row }) =>
        row.original.actorId
          ? t('audit.actorNumber', {
              name: row.original.actorName || t('audit.user'),
              id: row.original.actorId,
            })
          : t('audit.system'),
    },
    {
      accessorKey: 'action',
      header: t('audit.action'),
      cell: ({ row }) =>
        actions.includes(row.original.action)
          ? t(`audit.actions.${row.original.action}`)
          : row.original.action,
    },
    {
      accessorKey: 'resource',
      header: t('audit.resource'),
      cell: ({ row }) =>
        resources.includes(row.original.resource)
          ? t(`audit.resources.${row.original.resource}`)
          : row.original.resource,
    },
    { accessorKey: 'resourceId', header: t('audit.resourceId') },
    {
      accessorKey: 'createdAt',
      header: t('common.date'),
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
    {
      id: 'actions',
      header: t('common.actions'),
      enableSorting: false,
      cell: ({ row }) => (
        <TableIconButton
          icon={Eye}
          label={t('common.view')}
          onClick={() => onView(row.original)}
        />
      ),
    },
  ]
}

export function AuditPage() {
  const { t } = useTranslation()
  const [viewing, setViewing] = useState<Audit | null>(null)
  const columns = auditColumns(t, setViewing)
  const table = useListState([{ id: 'createdAt', desc: true }])
  const [actorId, setActorId] = useState('')
  const [action, setAction] = useState('')
  const [resource, setResource] = useState('')
  const sort = table.sorting[0]
  const params: AuditListParams = {
    page: table.pagination.pageIndex + 1,
    pageSize: table.pagination.pageSize,
    search: table.search,
    sort: sort?.id,
    direction: sort?.desc ? 'desc' : 'asc',
    actorId: actorId ? Number(actorId) : undefined,
    action: action || undefined,
    resource: resource || undefined,
  }
  const query = useQuery({
    queryKey: ['admin', 'audit', params],
    queryFn: ({ signal }) => listAudit(params, signal),
  })
  const resetPage = () =>
    table.onPaginationChange({ ...table.pagination, pageIndex: 0 })

  return (
    <>
      <header className="mb-7">
        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
          {t('nav.audit')}
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600">
          {t('audit.description')}
        </p>
      </header>
      <DataTable
        {...table}
        caption={t('audit.caption')}
        searchPlaceholder={t('audit.searchPlaceholder')}
        columns={columns}
        data={query.data?.items ?? []}
        total={query.data?.total ?? 0}
        getRowId={(row) => String(row.id)}
        isLoading={query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        filters={
          <>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('audit.actorId')}
              <input
                className="form-control w-36"
                type="number"
                min={1}
                value={actorId}
                onChange={(event) => {
                  setActorId(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('audit.action')}
              <AppSelect
                value={action}
                label={t('audit.action')}
                onValueChange={(value) => {
                  setAction(value)
                  resetPage()
                }}
                options={[
                  { value: '', label: t('audit.allActions') },
                  ...actions.map((value) => ({
                    value,
                    label: t(`audit.actions.${value}`),
                  })),
                ]}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('audit.resource')}
              <AppSelect
                value={resource}
                label={t('audit.resource')}
                onValueChange={(value) => {
                  setResource(value)
                  resetPage()
                }}
                options={[
                  { value: '', label: t('common.all') },
                  ...resources.map((value) => ({
                    value,
                    label: t(`audit.resources.${value}`),
                  })),
                ]}
              />
            </label>
          </>
        }
      />
      {viewing ? (
        <RecordDetailsDialog
          title={t('nav.audit')}
          onClose={() => setViewing(null)}
          fields={[
            { label: 'ID', value: viewing.id },
            {
              label: t('audit.actor'),
              value: viewing.actorId
                ? t('audit.actorNumber', {
                    name: viewing.actorName || t('audit.user'),
                    id: viewing.actorId,
                  })
                : t('audit.system'),
            },
            {
              label: t('audit.action'),
              value: actions.includes(viewing.action)
                ? t(`audit.actions.${viewing.action}`)
                : viewing.action,
            },
            {
              label: t('audit.resource'),
              value: resources.includes(viewing.resource)
                ? t(`audit.resources.${viewing.resource}`)
                : viewing.resource,
            },
            { label: t('audit.resourceId'), value: viewing.resourceId },
            {
              label: t('common.createdAt'),
              value: <DateCell value={viewing.createdAt} />,
            },
          ]}
        />
      ) : null}
      {query.isError ? (
        <button
          className="button-secondary mt-4"
          onClick={() => void query.refetch()}
        >
          <RotateCcw size={17} aria-hidden="true" />
          {t('common.retry')}
        </button>
      ) : null}
    </>
  )
}
