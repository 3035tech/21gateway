import { useTranslation } from 'react-i18next'
const knownStatuses = new Set([
  'active',
  'blocked',
  'expired',
  'pending',
  'queued',
  'claimed',
  'dispatching',
  'done',
  'accepted',
  'rejected',
  'unknown',
  'processing',
  'delivered',
  'failed',
  'SENT',
  'DELIVERED',
  'NOT_DELIVERED',
  'DUPLICATED',
  'INVALID_DESTINATION_ADDRESS',
  'BLACKLIST',
  'MESSAGE_CANCELLED',
  'INVALID_MESSAGE_FORMAT',
  'GENERAL_ERROR',
])
export function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation()
  const color = ['active', 'accepted', 'delivered', 'DELIVERED'].includes(
    status,
  )
    ? 'bg-emerald-50 text-emerald-800'
    : [
          'blocked',
          'expired',
          'rejected',
          'failed',
          'NOT_DELIVERED',
          'GENERAL_ERROR',
          'INVALID_DESTINATION_ADDRESS',
          'INVALID_MESSAGE_FORMAT',
          'BLACKLIST',
        ].includes(status)
      ? 'bg-rose-50 text-rose-800'
      : ['processing', 'claimed', 'dispatching', 'SENT'].includes(status)
        ? 'bg-sky-50 text-sky-800'
        : 'bg-amber-50 text-amber-800'
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold ${color}`}
    >
      <span
        aria-hidden="true"
        className="h-1.5 w-1.5 rounded-full bg-current"
      />
      {knownStatuses.has(status) ? t(`status.${status}`) : status}
    </span>
  )
}
