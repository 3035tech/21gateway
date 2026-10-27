import { DateCell } from '../../components/admin/DateCell'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { listMessages } from './api'
import type { Message } from './api'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
export function MessagesPage() {
  const { t } = useTranslation()
  const [viewing, setViewing] = useState<Message | null>(null)
  const columns: ColumnDef<Message>[] = [
    { accessorKey: 'id', header: t('common.messageId') },
    {
      accessorKey: 'companyName',
      header: t('common.client'),
      cell: ({ row }) =>
        row.original.companyName ||
        (row.original.companyId
          ? t('common.companyNumber', { id: row.original.companyId })
          : t('common.noCompany')),
    },
    { accessorKey: 'destination', header: t('common.destination') },
    { accessorKey: 'correlationId', header: t('common.reference') },
    {
      accessorKey: 'state',
      header: t('common.sending'),
      cell: ({ row }) => <StatusBadge status={row.original.state} />,
    },
    {
      accessorKey: 'deliveryStatus',
      header: t('common.delivery'),
      cell: ({ row }) =>
        row.original.deliveryStatus ? (
          <StatusBadge status={row.original.deliveryStatus} />
        ) : (
          t('common.unconfirmed')
        ),
    },
    {
      accessorKey: 'failureCode',
      header: t('common.failureCode'),
    },
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
        <button
          className="table-action"
          onClick={() => setViewing(row.original)}
        >
          {t('common.view')}
        </button>
      ),
    },
  ]
  return (
    <>
      <Listing
        title={t('nav.messages')}
        description={t('messages.description')}
        searchPlaceholder={t('common.destinationOrReference')}
        queryKey="messages"
        columns={columns}
        load={listMessages}
        filterName="state"
        filterOptions={[
          ['pending', t('messages.pending')],
          ['accepted', t('messages.accepted')],
          ['rejected', t('messages.rejected')],
          ['unknown', t('messages.unknown')],
        ]}
      />
      {viewing ? (
        <RecordDetailsDialog
          title={t('common.messageId')}
          onClose={() => setViewing(null)}
          fields={[
            { label: t('common.messageId'), value: viewing.id },
            {
              label: t('common.company'),
              value: viewing.companyName || viewing.companyId,
            },
            { label: t('common.destination'), value: viewing.destination },
            { label: t('common.reference'), value: viewing.correlationId },
            {
              label: t('common.sending'),
              value: <StatusBadge status={viewing.state} />,
            },
            {
              label: t('common.delivery'),
              value: viewing.deliveryStatus ? (
                <StatusBadge status={viewing.deliveryStatus} />
              ) : (
                t('common.unconfirmed')
              ),
            },
            { label: t('common.failureCode'), value: viewing.failureCode },
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
