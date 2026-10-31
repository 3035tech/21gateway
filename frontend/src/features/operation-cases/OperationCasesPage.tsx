import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { ClipboardList, RefreshCw } from 'lucide-react'
import { DataTable } from '../../components/data-table/DataTable'
import { useListState } from '../../components/data-table/useListState'
import { AppSelect } from '../../components/admin/AppSelect'
import { Dialog } from '../../components/admin/Dialog'
import { DateCell } from '../../components/admin/DateCell'
import { errorMessage } from '../../lib/admin'
import { useCompanyView } from '../companies/CompanyViewContext'
import { useUser } from '../auth/context'
import {
  caseKinds,
  caseStatuses,
  listOperationCases,
  operationCaseHistory,
  updateOperationCase,
} from './api'
import type { CaseKind, CaseStatus, OperationCase } from './api'

const workQueueRefreshMs = 60_000
const maxCaseNoteLength = 1000
const sourceRoutes: Record<CaseKind, string> = {
  unknown_send: '/envios',
  overdue_schedule: '/agendamentos',
  dlr_callback: '/callbacks',
  mo_callback: '/envios',
}

function sourcePath(item: OperationCase) {
  return `${sourceRoutes[item.kind]}?search=${item.kind === 'mo_callback' ? item.attemptId : item.sourceId}`
}

function CaseDialog({
  item,
  onClose,
}: {
  item: OperationCase
  onClose: () => void
}) {
  const { t } = useTranslation()
  const user = useUser()
  const client = useQueryClient()
  const [note, setNote] = useState('')
  const [failure, setFailure] = useState('')
  const history = useQuery({
    queryKey: ['admin', 'operation-case-history', item.kind, item.sourceId],
    queryFn: ({ signal }) => operationCaseHistory(item, signal),
    enabled: item.tracked,
  })
  const mutation = useMutation({
    mutationFn: (status: CaseStatus) => updateOperationCase(item, status, note),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['admin', 'operation-cases'] })
      await client.invalidateQueries({
        queryKey: ['admin', 'operation-case-history', item.kind, item.sourceId],
      })
      onClose()
    },
    onError: (error) => setFailure(errorMessage(error)),
  })
  const canWrite = user?.role !== 'viewer'
  const mine = item.ownerId === user?.id
  return (
    <Dialog
      title={t('workQueue.caseTitle', { id: item.sourceId })}
      onClose={onClose}
    >
      <div className="space-y-5 text-sm">
        <p className="text-slate-600">
          {t(`workQueue.kind.${item.kind}`)} · {item.companyName}
        </p>
        <p>
          {t('workQueue.statusLabel')}:{' '}
          <strong>{t(`workQueue.status.${item.status}`)}</strong>
        </p>
        {item.ownerName ? (
          <p>
            {t('workQueue.owner')}: {item.ownerName}
          </p>
        ) : null}
        {item.note ? (
          <p className="break-words text-slate-600">
            {t('workQueue.lastNote')}: {item.note}
          </p>
        ) : null}
        {!item.sourceActive ? (
          <p className="rounded-lg bg-emerald-50 p-3 text-emerald-900">
            {t('workQueue.sourceCleared')}
          </p>
        ) : null}
        <Link
          className="font-semibold text-brand-purple underline"
          to={sourcePath(item)}
          onClick={onClose}
        >
          {t('workQueue.inspectSource')}
        </Link>
        {item.kind === 'unknown_send' ? (
          <p className="rounded-lg bg-amber-50 p-3 text-amber-900">
            {t('workQueue.noResend')}
          </p>
        ) : null}
        {canWrite &&
        item.status !== 'completed' &&
        (item.ownerId === null || mine) ? (
          <label className="block font-semibold">
            {t('workQueue.note')}
            <textarea
              className="form-control mt-2 min-h-24 w-full"
              maxLength={maxCaseNoteLength}
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder={t('workQueue.noteHint')}
            />
          </label>
        ) : null}
        {history.isPending && item.tracked ? (
          <p>{t('common.loading')}</p>
        ) : null}
        {history.isError ? (
          <p role="alert">{errorMessage(history.error)}</p>
        ) : null}
        {history.data?.length ? (
          <section>
            <h3 className="font-bold">{t('workQueue.history')}</h3>
            <ol className="mt-2 space-y-2">
              {history.data.map((event, index) => (
                <li key={index} className="border-l-2 border-purple-200 pl-3">
                  <span>
                    {event.actorName} · {t(`workQueue.status.${event.status}`)}{' '}
                    · <DateCell value={event.createdAt} />
                  </span>
                  {event.note ? (
                    <p className="break-words text-slate-600">{event.note}</p>
                  ) : null}
                </li>
              ))}
            </ol>
          </section>
        ) : null}
        {failure ? (
          <p role="alert" className="text-rose-700">
            {failure}
          </p>
        ) : null}
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="button-secondary" onClick={onClose}>
            {t('common.close')}
          </button>
          {canWrite && item.status === 'open' ? (
            <button
              type="button"
              className="button-primary"
              disabled={mutation.isPending}
              onClick={() => mutation.mutate('in_progress')}
            >
              {t('workQueue.claim')}
            </button>
          ) : null}
          {canWrite && item.status === 'in_progress' && mine ? (
            <button
              type="button"
              className="button-secondary"
              disabled={mutation.isPending || !note.trim()}
              onClick={() => mutation.mutate('in_progress')}
            >
              {t('workQueue.addNote')}
            </button>
          ) : null}
          {canWrite && item.status === 'in_progress' && mine ? (
            <button
              type="button"
              className="button-secondary"
              disabled={mutation.isPending || !note.trim()}
              onClick={() => mutation.mutate('open')}
            >
              {t('workQueue.release')}
            </button>
          ) : null}
          {canWrite && item.status === 'in_progress' && mine ? (
            <button
              type="button"
              className="button-primary"
              disabled={mutation.isPending || !note.trim()}
              onClick={() => mutation.mutate('completed')}
            >
              {t('workQueue.complete')}
            </button>
          ) : null}
        </div>
      </div>
    </Dialog>
  )
}

export function OperationCasesPage() {
  const { t } = useTranslation()
  const user = useUser()
  const selectedCompany = useCompanyView()
  const table = useListState([{ id: 'createdAt', desc: true }])
  const [company, setCompany] = useState('')
  const [kind, setKind] = useState<CaseKind | ''>('')
  const [status, setStatus] = useState<CaseStatus | ''>('')
  const [selected, setSelected] = useState<OperationCase | null>(null)
  const sort = table.sorting[0]
  const params = {
    page: table.pagination.pageIndex + 1,
    pageSize: table.pagination.pageSize,
    search: table.search,
    companyId: selectedCompany ?? (company ? Number(company) : undefined),
    kind: kind || undefined,
    status: status || undefined,
    sort: sort?.id,
    direction: (sort?.desc ? 'desc' : 'asc') as 'asc' | 'desc',
  }
  const query = useQuery({
    queryKey: ['admin', 'operation-cases', params],
    queryFn: ({ signal }) => listOperationCases(params, signal),
    refetchInterval: workQueueRefreshMs,
  })
  const resetPage = () =>
    table.onPaginationChange({ ...table.pagination, pageIndex: 0 })
  const columns: ColumnDef<OperationCase>[] = [
    {
      accessorKey: 'kind',
      header: t('workQueue.type'),
      cell: ({ row }) => t(`workQueue.kind.${row.original.kind}`),
    },
    {
      accessorKey: 'sourceId',
      header: 'ID',
      meta: { compact: true, centered: true },
    },
    { accessorKey: 'companyName', header: t('common.company') },
    {
      accessorKey: 'createdAt',
      header: t('common.date'),
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
    {
      accessorKey: 'status',
      header: t('common.status'),
      cell: ({ row }) => t(`workQueue.status.${row.original.status}`),
    },
    {
      id: 'owner',
      accessorFn: (item) => item.ownerName,
      header: t('workQueue.owner'),
      cell: ({ row }) => row.original.ownerName || '—',
      enableSorting: false,
    },
    {
      id: 'actions',
      header: t('common.actions'),
      enableSorting: false,
      cell: ({ row }) => (
        <button
          type="button"
          className="button-ghost"
          onClick={() => setSelected(row.original)}
        >
          {t('common.view')}
        </button>
      ),
    },
  ]
  return (
    <>
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold">
            <ClipboardList aria-hidden="true" />
            {t('workQueue.title')}
          </h1>
          <p className="mt-2 text-sm text-slate-600">
            {t('workQueue.description')}
          </p>
        </div>
        <button
          className="button-secondary"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw size={16} aria-hidden="true" />
          {t('overview.refresh')}
        </button>
      </header>
      <DataTable
        {...table}
        caption={t('workQueue.title')}
        columns={columns}
        data={query.data?.items ?? []}
        total={query.data?.total ?? 0}
        getRowId={(item) => `${item.kind}-${item.sourceId}`}
        isLoading={query.isPending}
        error={query.isError ? errorMessage(query.error) : undefined}
        hasActiveFilters={!!kind || !!status || !!company}
        onClearFilters={() => {
          setKind('')
          setStatus('')
          setCompany('')
          table.onSearchChange('')
        }}
        filters={
          <>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('workQueue.type')}
              <AppSelect
                value={kind}
                label={t('workQueue.type')}
                onValueChange={(value) => {
                  setKind(value as CaseKind | '')
                  resetPage()
                }}
                options={[
                  { value: '', label: t('common.all') },
                  ...caseKinds.map((value) => ({
                    value,
                    label: t(`workQueue.kind.${value}`),
                  })),
                ]}
              />
            </label>
            <label className="flex flex-col gap-1 text-sm font-semibold">
              {t('common.status')}
              <AppSelect
                value={status}
                label={t('common.status')}
                onValueChange={(value) => {
                  setStatus(value as CaseStatus | '')
                  resetPage()
                }}
                options={[
                  { value: '', label: t('common.all') },
                  ...caseStatuses.map((value) => ({
                    value,
                    label: t(`workQueue.status.${value}`),
                  })),
                ]}
              />
            </label>
            {user?.role === 'admin' && !selectedCompany ? (
              <label className="flex flex-col gap-1 text-sm font-semibold">
                {t('common.companyId')}
                <input
                  className="form-control w-28"
                  type="number"
                  min="1"
                  value={company}
                  onChange={(event) => {
                    setCompany(event.target.value)
                    resetPage()
                  }}
                />
              </label>
            ) : null}
          </>
        }
      />
      {selected ? (
        <CaseDialog item={selected} onClose={() => setSelected(null)} />
      ) : null}
    </>
  )
}
