import { useEffect, useRef } from 'react'
import type { PropsWithChildren } from 'react'
import { X } from 'lucide-react'
export function Dialog({
  title,
  onClose,
  children,
  busy = false,
  wide = false,
}: PropsWithChildren<{
  title: string
  onClose: () => void
  busy?: boolean
  wide?: boolean
}>) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current
    dialog?.showModal()
    return () => dialog?.close()
  }, [])
  return (
    <dialog
      ref={ref}
      onCancel={(event) => {
        if (busy) event.preventDefault()
        else onClose()
      }}
      aria-busy={busy}
      aria-label={title}
      className={`m-auto max-h-[90dvh] w-[calc(100%_-_2rem)] ${wide ? 'max-w-3xl' : 'max-w-xl'} overflow-y-auto rounded-2xl bg-white p-5 shadow-xl backdrop:bg-slate-950/40 sm:p-6`}
    >
      <div className="mb-6 flex items-center justify-between gap-4">
        <h2 className="text-xl font-bold">{title}</h2>
        <button
          type="button"
          onClick={onClose}
          disabled={busy}
          className="button-secondary"
          aria-label="Fechar"
        >
          <X size={18} />
        </button>
      </div>
      {children}
    </dialog>
  )
}
