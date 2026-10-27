import { errorMessage } from '../../lib/admin'
export function FormFooter({
  pending,
  error,
  onCancel,
  submitLabel = 'Salvar alterações',
}: {
  pending: boolean
  error: unknown
  submitLabel?: string
  onCancel: () => void
}) {
  return (
    <>
      {error ? (
        <p role="alert" className="text-sm text-rose-700">
          {errorMessage(error)}
        </p>
      ) : null}
      <div className="flex justify-end gap-3 border-t border-slate-100 pt-5">
        <button
          className="button-secondary"
          type="button"
          disabled={pending}
          onClick={onCancel}
        >
          Cancelar
        </button>
        <button className="button-primary" type="submit" disabled={pending}>
          {pending ? 'Salvando…' : submitLabel}
        </button>
      </div>
    </>
  )
}
export function FieldError({ message }: { message?: string }) {
  return message ? (
    <span role="alert" className="text-xs text-rose-700">
      {message}
    </span>
  ) : null
}
