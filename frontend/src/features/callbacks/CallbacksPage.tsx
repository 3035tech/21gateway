import { DateCell } from '../../components/admin/DateCell'
import { useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { Confirmation } from '../../components/admin/Confirmation'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
import { useUser } from '../auth/context'
import { listCallbacks, listReceivedCallbacks, retryCallback } from './api'
import type { Callback, ReceivedCallback } from './api'
import { useTranslation } from 'react-i18next'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { TableIconButton } from '../../components/admin/TableIconAction'
import { Eye, RotateCcw } from 'lucide-react'
const callbackErrors = new Set([
  'NETWORK_ERROR',
  'HTTP_ERROR',
  'ACCESS_BLOCKED',
])
export function CallbacksPage() {
  const { t } = useTranslation()
  const writable = useUser()?.role !== 'viewer'
  const [retry, setRetry] = useState<Callback | null>(null)
  const [viewing, setViewing] = useState<Callback | null>(null)
  const mutation = useAdminMutation(
    retryCallback,
    () => setRetry(null),
    'feedback.retryScheduled',
  )
  const receivedColumns: ColumnDef<ReceivedCallback>[] = [
    {
      accessorKey: 'attemptId',
      header: t('common.sending'),
      meta: { compact: true, centered: true },
    },
    { accessorKey: 'companyName', header: t('common.client') },
    {
      accessorKey: 'kind',
      header: t('callbacks.eventKind'),
      meta: { compact: true },
      cell: ({ row }) => t(`callbacks.kind.${row.original.kind}`),
    },
    {
      accessorKey: 'status',
      header: t('common.delivery'),
      meta: { compact: true },
      cell: ({ row }) =>
        row.original.kind === 'mo' ? (
          t('callbacks.moReceived')
        ) : (
          <StatusBadge status={row.original.status} />
        ),
    },
    {
      accessorKey: 'providerMessageId',
      header: t('callbacks.tableProviderMessageId'),
    },
    {
      accessorKey: 'occurredAt',
      header: t('callbacks.tableProviderTime'),
      meta: { compact: true },
      cell: ({ row }) => <DateCell value={row.original.occurredAt} />,
    },
    {
      accessorKey: 'receivedAt',
      header: t('callbacks.tableReceivedAt'),
      meta: { compact: true },
      cell: ({ row }) => <DateCell value={row.original.receivedAt} />,
    },
  ]
  const columns: ColumnDef<Callback>[] = [
    { accessorKey: 'id', header: 'ID', meta: { compact: true, centered: true } },
    {
      accessorKey: 'attemptId',
      header: t('common.sending'),
      meta: { compact: true, centered: true },
    },
    {
      accessorKey: 'companyName',
      header: t('common.client'),
      cell: ({ row }) =>
        row.original.companyName ||
        (row.original.companyId
          ? t('common.companyNumber', { id: row.original.companyId })
          : t('common.noCompany')),
    },
    { accessorKey: 'url', header: t('common.target') },
    {
      accessorKey: 'status',
      header: t('common.delivery'),
      meta: { compact: true },
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: 'state',
      header: 'Callback',
      meta: { compact: true },
      cell: ({ row }) => <StatusBadge status={row.original.state} />,
    },
    {
      accessorKey: 'attempts',
      header: t('callbacks.attempts'),
      meta: { compact: true },
    },
    {
      accessorKey: 'lastError',
      header: t('callbacks.lastFailure'),
      cell: ({ row }) =>
        row.original.lastError ? (
          <span title={row.original.lastError}>
            {callbackErrors.has(row.original.lastError)
              ? t(`callbackErrors.${row.original.lastError}`)
              : row.original.lastError}
          </span>
        ) : (
          t('callbacks.noFailures')
        ),
    },
    {
      accessorKey: 'createdAt',
      header: t('common.date'),
      meta: { compact: true },
      cell: ({ row }) => <DateCell value={row.original.createdAt} />,
    },
  ]
  columns.push({
    id: 'actions',
    header: t('common.actions'),
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex w-max items-center gap-1">
        <TableIconButton
          icon={Eye}
          label={t('common.view')}
          onClick={() => setViewing(row.original)}
        />
        {writable && row.original.state === 'failed' ? (
          <TableIconButton
            icon={RotateCcw}
            label={t('common.retry')}
            onClick={() => {
              mutation.reset()
              setRetry(row.original)
            }}
          />
        ) : null}
      </div>
    ),
  })
  return (
    <>
      <Listing
        title={t('callbacks.receivedTitle')}
        description={t('callbacks.receivedDescription')}
        searchPlaceholder={t('callbacks.receivedSearchPlaceholder')}
        queryKey="receivedCallbacks"
        columns={receivedColumns}
        load={listReceivedCallbacks}
      />
      <div className="mt-12" />
      <Listing
        title={t('callbacks.title')}
        description={t('callbacks.description')}
        searchPlaceholder={t('callbacks.searchPlaceholder')}
        queryKey="callbacks"
        columns={columns}
        load={listCallbacks}
        headingLevel={2}
        filterName="callbackState"
        filterOptions={[
          ['pending', t('messages.pending')],
          ['processing', t('status.processing')],
          ['delivered', t('callbacks.delivered')],
          ['failed', t('callbacks.failed')],
        ]}
      />
      {retry ? (
        <Confirmation
          title={t('callbacks.retryTitle')}
          description={t('callbacks.retryDescription', { id: retry.attemptId })}
          pending={mutation.isPending}
          error={mutation.error}
          onConfirm={() => mutation.mutate(retry.id)}
          onClose={() => setRetry(null)}
        />
      ) : null}
      {viewing ? (
        <RecordDetailsDialog
          title={`Callback #${viewing.id}`}
          onClose={() => setViewing(null)}
          fields={[
            { label: 'ID', value: viewing.id },
            { label: t('common.messageId'), value: viewing.attemptId },
            {
              label: t('common.company'),
              value: viewing.companyName || viewing.companyId,
            },
            { label: t('common.target'), value: viewing.url },
            {
              label: t('callbacks.smsDelivery'),
              value: <StatusBadge status={viewing.status} />,
            },
            {
              label: 'Callback',
              value: <StatusBadge status={viewing.state} />,
            },
            { label: t('callbacks.attempts'), value: viewing.attempts },
            {
              label: t('callbacks.lastFailure'),
              value: viewing.lastError || t('callbacks.noFailures'),
            },
            {
              label: t('common.createdAt'),
              value: <DateCell value={viewing.createdAt} />,
            },
          ]}
        />
      ) : null}
    </>
  )
}
