import { Dialog } from './Dialog'
import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
export function Confirmation({
  title,
  description,
  pending,
  error,
  onConfirm,
  onClose,
}: {
  title: string
  description: string
  pending: boolean
  error: unknown
  onConfirm: () => void
  onClose: () => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog title={title} onClose={onClose} busy={pending}>
      <p className="text-sm leading-6 text-slate-600">{description}</p>
      {error ? (
        <p role="alert" className="mt-4 text-sm text-rose-700">
          {errorMessage(error)}
        </p>
      ) : null}
      <div className="mt-6 flex justify-end gap-3">
        <button
          disabled={pending}
          className="button-secondary"
          onClick={onClose}
        >
          {t('common.cancel')}
        </button>
        <button
          disabled={pending}
          className="button-danger"
          onClick={onConfirm}
        >
          {pending ? t('common.processing') : t('common.confirm')}
        </button>
      </div>
    </Dialog>
  )
}
