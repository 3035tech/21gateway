import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { PropsWithChildren } from 'react'
import { CheckCircle2, CircleAlert, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FeedbackContext } from './feedbackContext'
import type { Feedback } from './feedbackContext'

type Notice = { id: number; kind: 'success' | 'error'; message: string }

export function FeedbackProvider({ children }: PropsWithChildren) {
  const { t } = useTranslation()
  const [notice, setNotice] = useState<Notice | null>(null)
  const nextId = useRef(0)
  useEffect(() => {
    if (!notice) return
    const timer = window.setTimeout(() => setNotice(null), 6000)
    return () => window.clearTimeout(timer)
  }, [notice])
  const show = useCallback((kind: Notice['kind'], message: string) => {
    nextId.current += 1
    setNotice({ id: nextId.current, kind, message })
  }, [])
  const feedback = useMemo<Feedback>(
    () => ({
      success: (message) => show('success', message),
      error: (message) => show('error', message),
    }),
    [show],
  )
  return (
    <FeedbackContext.Provider value={feedback}>
      {children}
      {notice ? (
        <div
          key={notice.id}
          role={notice.kind === 'error' ? 'alert' : 'status'}
          className={`fixed bottom-5 right-5 z-[100] flex max-w-sm items-start gap-3 rounded-xl border bg-white p-4 text-sm font-semibold shadow-xl ${notice.kind === 'error' ? 'border-rose-200 text-rose-800' : 'border-emerald-200 text-emerald-900'}`}
        >
          {notice.kind === 'error' ? (
            <CircleAlert size={20} aria-hidden="true" />
          ) : (
            <CheckCircle2 size={20} aria-hidden="true" />
          )}
          <span>{notice.message}</span>
          <button
            type="button"
            onClick={() => setNotice(null)}
            className="ml-auto rounded p-1 hover:bg-slate-100 focus-visible:outline-2"
            aria-label={t('feedback.dismiss')}
          >
            <X size={16} aria-hidden="true" />
          </button>
        </div>
      ) : null}
    </FeedbackContext.Provider>
  )
}
