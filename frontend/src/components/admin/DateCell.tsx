import { formatDate } from '../../lib/admin'
export function DateCell({ value }: { value: string }) {
  return (
    <time
      dateTime={value}
      className="whitespace-nowrap tabular-nums text-slate-600"
    >
      {formatDate(value)}
    </time>
  )
}
