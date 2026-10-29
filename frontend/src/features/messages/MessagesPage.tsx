import { DateCell } from '../../components/admin/DateCell'
import type { ColumnDef } from '@tanstack/react-table'
import { Listing } from '../../components/admin/Listing'
import { StatusBadge } from '../../components/admin/StatusBadge'
import { listMessages, getMessageContent, getMessageTrace } from './api'
import type { Message } from './api'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { RecordDetailsDialog } from '../../components/admin/RecordDetailsDialog'
import { useQuery } from '@tanstack/react-query'
import { errorMessage } from '../../lib/admin'

function MessageDetails({
  message,
  onClose,
}: {
  message: Message
  onClose: () => void
}) {
  const { t } = useTranslation()
  const content = useQuery({
    queryKey: ['admin', 'message-content', message.id],
    queryFn: ({ signal }) => getMessageContent(message.id, signal),
    gcTime: 0,
    staleTime: 0,
  })
  const trace = useQuery({
    queryKey: ['admin', 'message-trace', message.id],
    queryFn: ({ signal }) => getMessageTrace(message.id, signal),
  })
  const timeline = trace.isPending ? t('common.loading') : trace.isError ? (
    errorMessage(trace.error)
  ) : (
    <div>
      <p className="mb-3 text-xs font-normal text-slate-500">{t('messages.timelineHint')}</p>
      {trace.data.providerMessageId ? <p className="mb-4 break-all text-xs font-normal text-slate-600">{t('messages.providerId')}: {trace.data.providerMessageId}</p> : null}
      <ol className="border-l-2 border-purple-200 pl-4">
        <li className="relative pb-4 before:absolute before:-left-[21px] before:top-1 before:size-2.5 before:rounded-full before:bg-brand-purple">
          <span className="font-semibold">{t('messages.acceptedByGateway')}</span>
          <span className="ml-2 font-normal text-slate-600"><DateCell value={trace.data.createdAt} /></span>
        </li>
        {trace.data.dispatchedAt ? <li className="relative pb-4 before:absolute before:-left-[21px] before:top-1 before:size-2.5 before:rounded-full before:bg-brand-purple">
          <span className="font-semibold">{t('messages.dispatched')}</span>
          <span className="ml-2 font-normal text-slate-600"><DateCell value={trace.data.dispatchedAt} /></span>
        </li> : null}
        {trace.data.submissionState !== 'pending' ? <li className="relative pb-4 before:absolute before:-left-[21px] before:top-1 before:size-2.5 before:rounded-full before:bg-brand-purple">
          <span className="font-semibold">{t('messages.submissionResult')}</span>
          <span className="ml-2"><StatusBadge status={trace.data.submissionState} /></span>
          <p className="mt-1 text-xs font-normal text-slate-500">{t('messages.submissionTimeUnavailable')}</p>
        </li> : null}
        {trace.data.events.map((event) => <li key={`${event.kind}-${event.id}`} className="relative pb-4 before:absolute before:-left-[21px] before:top-1 before:size-2.5 before:rounded-full before:bg-brand-purple">
          <span className="font-semibold">{t(`messages.traceKind.${event.kind}`, { defaultValue: event.kind })}</span>
          <span className="ml-2 font-normal text-slate-600"><DateCell value={event.occurredAt} /></span>
          <p className="mt-1 flex flex-wrap items-center gap-2 font-normal text-slate-700">
            {event.kind === 'mo' ? t('callbacks.moReceived') : <StatusBadge status={event.status} />}
            {event.attempts ? <span>{t('callbacks.attempts')}: {event.attempts}</span> : null}
            {event.failureCode ? <span>{event.failureCode}</span> : null}
          </p>
          {event.kind === 'zoug_delivery' || event.kind === 'mo' ? <p className="mt-1 text-xs font-normal text-slate-500">{t('messages.recordedAt')}: <DateCell value={event.recordedAt} /></p> : null}
        </li>)}
      </ol>
      {trace.data.truncated ? <p className="text-xs font-normal text-amber-800">{t('messages.timelineTruncated')}</p> : null}
    </div>
  )
  const contentValue = content.isPending ? (
    t('common.loading')
  ) : content.isError ? (
    errorMessage(content.error)
  ) : content.data.state === 'available' ? (
    <span className="whitespace-pre-wrap">{content.data.messageText}</span>
  ) : (
    t(`messages.content.${content.data.state}`)
  )
  return (
    <RecordDetailsDialog
      title={t('common.messageId')}
      onClose={onClose}
      fields={[
        { label: t('common.messageId'), value: message.id },
        {
          label: t('common.company'),
          value: message.companyName || message.companyId,
        },
        { label: t('common.destination'), value: message.destination },
        { label: t('common.reference'), value: message.correlationId },
        {
          label: t('common.sending'),
          value: <StatusBadge status={message.state} />,
        },
        {
          label: t('common.delivery'),
          value: message.deliveryStatus ? (
            <StatusBadge status={message.deliveryStatus} />
          ) : (
            t('common.unconfirmed')
          ),
        },
        { label: t('common.failureCode'), value: message.failureCode },
        {
          label: t('common.createdAt'),
          value: <DateCell value={message.createdAt} />,
        },
        { label: t('messages.contentLabel'), value: contentValue, wide: true },
        { label: t('messages.timeline'), value: timeline, wide: true },
      ]}
    />
  )
}
export function MessagesPage() {
  const { t } = useTranslation()
  const [viewing, setViewing] = useState<Message | null>(null)
  const columns: ColumnDef<Message>[] = [
    { accessorKey: 'id', header: t('common.id') },
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
        stickyActions
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
        <MessageDetails message={viewing} onClose={() => setViewing(null)} />
      ) : null}
    </>
  )
}
