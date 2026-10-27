import type { ReactNode } from 'react'
import { Dialog } from './Dialog'
import { useTranslation } from 'react-i18next'

export function RecordDetailsDialog({
  title,
  fields,
  onClose,
}: {
  title: string
  fields: { label: string; value: ReactNode }[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog title={title} onClose={onClose}>
      <dl className="grid gap-4 sm:grid-cols-2">
        {fields.map(({ label, value }) => (
          <div className="min-w-0 border-b border-slate-100 pb-3" key={label}>
            <dt className="text-xs font-semibold text-slate-500">{label}</dt>
            <dd className="mt-1 break-words text-sm font-medium text-slate-900">
              {value === null || value === undefined || value === ''
                ? '—'
                : value}
            </dd>
          </div>
        ))}
      </dl>
      <div className="mt-6 flex justify-end">
        <button className="button-secondary" onClick={onClose}>
          {t('common.close')}
        </button>
      </div>
    </Dialog>
  )
}
