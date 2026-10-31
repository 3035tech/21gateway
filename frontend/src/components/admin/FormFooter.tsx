import { errorMessage } from '../../lib/admin'
import { useTranslation } from 'react-i18next'
import { Save } from 'lucide-react'
export function FormFooter({
  pending,
  error,
  onCancel,
  submitLabel,
  pendingLabel,
  disabled = false,
}: {
  pending: boolean
  disabled?: boolean
  error: unknown
  submitLabel?: string
  pendingLabel?: string
  onCancel: () => void
}) {
  const { t } = useTranslation()
  return (
    <>
      {error ? (
        <p role="alert" className="text-sm text-rose-700">
          {errorMessage(error)}
        </p>
      ) : null}
      <div className="flex justify-end gap-3 border-t border-slate-100 pt-5 max-[360px]:flex-col-reverse">
        <button
          className="button-secondary max-[360px]:w-full"
          type="button"
          disabled={pending}
          onClick={onCancel}
        >
          {t('common.cancel')}
        </button>
        <button className="button-primary max-[360px]:w-full" type="submit" disabled={pending || disabled}>
          <Save size={17} aria-hidden="true" />
          {pending
            ? (pendingLabel ?? t('common.saving'))
            : (submitLabel ?? t('common.save'))}
        </button>
      </div>
    </>
  )
}
export function FieldError({ message }: { message?: string }) {
  const { t } = useTranslation()
  return message ? (
    <span role="alert" className="text-xs text-rose-700">
      {message.startsWith('validation.') ? t(message) : message}
    </span>
  ) : null
}
