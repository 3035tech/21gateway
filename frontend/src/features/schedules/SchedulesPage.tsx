import { useState } from 'react'
import { useCompanyView } from '../companies/CompanyViewContext'
import { useQuery } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { DateCell } from '../../components/admin/DateCell'
import { TableIconButton } from '../../components/admin/TableIconAction'
import { Eye, RotateCcw } from 'lucide-react'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { errorMessage } from '../../lib/admin'
import { listSchedules, scheduleStates } from './api'
import type { Schedule, ScheduleListParams } from './api'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { AppSelect } from '../../components/admin/AppSelect'

function scheduleColumns(
  t: (key: string) => string,
  onView: (row: Schedule) => void,
): ColumnDef<Schedule>[] {
  return [
    { accessorKey: 'attemptId', header: t('common.messageId') },
    {
      accessorKey: 'companyName',
      header: t('common.client'),
      cell: ({ row }) => row.original.companyName || t('common.noCompany'),
    },
    { accessorKey: 'username', header: t('common.apiUsername') },
    {
      accessorKey: 'createdAt',
      header: t('schedules.requestedAt'),
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
    {
      accessorKey: 'scheduledAt',
      header: t('schedules.scheduledFor'),
      cell: ({ row }) => <DateCell value={row.original.scheduledAt} />,
    },
    {
      accessorKey: 'state',
      header: t('schedules.queue'),
      cell: ({ row }) => <StatusBadge status={row.original.state} />,
    },
    {
      accessorKey: 'submissionState',
      header: t('common.sending'),
      cell: ({ row }) => <StatusBadge status={row.original.submissionState} />,
    },
    { accessorKey: 'destination', header: t('common.destination') },
    { accessorKey: 'correlationId', header: t('common.reference') },
    {
      accessorKey: 'dispatchedAt',
      header: t('schedules.dispatchedAt'),
      cell: ({ row }) =>
        row.original.dispatchedAt ? (
          <DateCell value={row.original.dispatchedAt} />
        ) : (
          t('schedules.notDispatched')
        ),
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

function utc(value: string) {
  if (!value) return undefined
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString()
}

export function SchedulesPage() {
  const { t } = useTranslation()
  const selectedCompanyId = useCompanyView()
  const [viewing, setViewing] = useState<Schedule | null>(null)
  const columns = scheduleColumns(t, setViewing)
  const table = useListState([{ id: 'scheduledAt', desc: true }])
  const [companyId, setCompanyId] = useState('')
  const [credentialId, setCredentialId] = useState('')
  const [state, setState] = useState('')
  const [dueFrom, setDueFrom] = useState('')
  const [dueTo, setDueTo] = useState('')
  const invalidRange = !!dueFrom && !!dueTo && dueFrom > dueTo
  const sort = table.sorting[0]
  const params: ScheduleListParams = {
    page: table.pagination.pageIndex + 1,
    pageSize: table.pagination.pageSize,
    search: table.search,
    sort: sort?.id,
    direction: sort?.desc ? 'desc' : 'asc',
    companyId: selectedCompanyId ?? (companyId ? Number(companyId) : undefined),
    credentialId: credentialId ? Number(credentialId) : undefined,
    state: state || undefined,
    dueFrom: utc(dueFrom),
    dueTo: utc(dueTo),
  }
  const query = useQuery({
    queryKey: ['admin', 'schedules', params],
    queryFn: ({ signal }) => listSchedules(params, signal),
    enabled: !invalidRange,
  })
  const resetPage = () =>
    table.onPaginationChange({ ...table.pagination, pageIndex: 0 })
  return (
    <>
      <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
        {t('nav.schedules')}
      </h1>
      <p className="mb-2 mt-2 max-w-3xl text-sm leading-6 text-slate-600">
        {t('schedules.description')}
      </p>
      <p className="mb-6 text-xs text-slate-500">
        {t('schedules.timezoneHint')}
      </p>
      {invalidRange ? (
        <p role="alert" className="mb-4 text-sm text-rose-700">
          {t('schedules.invalidRange')}
        </p>
      ) : null}
      <DataTable
        {...table}
        stickyActions
        caption={t('schedules.caption')}
        wide
        searchPlaceholder={t('common.destinationOrReference')}
        columns={columns}
        data={invalidRange ? [] : (query.data?.items ?? [])}
        total={invalidRange ? 0 : (query.data?.total ?? 0)}
        getRowId={(row) => String(row.id)}
        isLoading={!invalidRange && query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        filters={
          <>
            {!selectedCompanyId ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                {t('common.companyId')}
                <input
                  className="form-control w-28"
                  type="number"
                  min={1}
                  value={companyId}
                  onChange={(event) => {
                    setCompanyId(event.target.value)
                    resetPage()
                  }}
                />
              </label>
            ) : null}
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('schedules.credentialId')}
              <input
                className="form-control w-28"
                type="number"
                min={1}
                value={credentialId}
                onChange={(event) => {
                  setCredentialId(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('schedules.queueState')}
              <AppSelect
                value={state}
                label={t('schedules.queueState')}
                onValueChange={(value) => {
                  setState(value)
                  resetPage()
                }}
                options={[
                  { value: '', label: t('common.all') },
                  ...scheduleStates.map((value) => ({
                    value,
                    label: t(`status.${value}`),
                  })),
                ]}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('schedules.from')}
              <input
                className="form-control"
                type="datetime-local"
                value={dueFrom}
                onChange={(event) => {
                  setDueFrom(event.target.value)
                  resetPage()
                }}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('schedules.to')}
              <input
                className="form-control"
                type="datetime-local"
                value={dueTo}
                onChange={(event) => {
                  setDueTo(event.target.value)
                  resetPage()
                }}
              />
            </label>
          </>
        }
      />
      {viewing ? (
        <RecordDetailsDialog
          title={t('nav.schedules')}
          onClose={() => setViewing(null)}
          fields={[
            { label: t('common.messageId'), value: viewing.attemptId },
            { label: t('common.company'), value: viewing.companyName },
            { label: t('common.apiUsername'), value: viewing.username },
            { label: t('common.destination'), value: viewing.destination },
            { label: t('common.reference'), value: viewing.correlationId },
            {
              label: t('schedules.requestedAt'),
              value: <DateCell value={viewing.createdAt} />,
            },
            {
              label: t('schedules.scheduledFor'),
              value: <DateCell value={viewing.scheduledAt} />,
            },
            {
              label: t('schedules.queue'),
              value: <StatusBadge status={viewing.state} />,
            },
            {
              label: t('common.sending'),
              value: <StatusBadge status={viewing.submissionState} />,
            },
            {
              label: t('schedules.dispatchedAt'),
              value: viewing.dispatchedAt ? (
                <DateCell value={viewing.dispatchedAt} />
              ) : (
                t('schedules.notDispatched')
              ),
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
