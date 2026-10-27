import { Dialog } from './Dialog'
import { errorMessage } from '../../lib/admin'
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
          Cancelar
        </button>
        <button
          disabled={pending}
          className="button-primary"
          onClick={onConfirm}
        >
          {pending ? 'Processando…' : 'Confirmar'}
        </button>
      </div>
    </Dialog>
  )
}
