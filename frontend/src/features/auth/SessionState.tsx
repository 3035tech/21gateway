import { LoaderCircle, ShieldCheck, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
export function SessionState({
  checking,
  onRetry,
}: {
  checking: boolean
  onRetry?: () => void
}) {
  const { t } = useTranslation()
  return (
    <main className="flex min-h-dvh items-center justify-center px-5 py-10">
      <section className="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b-4 border-brand-magenta bg-brand-purple px-8 py-7">
          <img
            src="/brand/21-mobile.png"
            alt="21 Mobile"
            className="h-12 w-auto"
          />
        </div>
        <div className="p-8">
          {checking ? (
            <LoaderCircle
              size={28}
              aria-hidden="true"
              className="mb-5 animate-spin text-brand-purple"
            />
          ) : (
            <ShieldCheck
              size={28}
              aria-hidden="true"
              className="mb-5 text-brand-purple"
            />
          )}
          <h1 className="text-2xl font-bold">
            {checking ? t('session.checkingTitle') : t('session.failedTitle')}
          </h1>
          <p
            role={checking ? 'status' : 'alert'}
            className="mt-3 text-sm leading-6 text-slate-600"
          >
            {checking
              ? t('session.checkingDescription')
              : t('session.failedDescription')}
          </p>
          {onRetry && !checking ? (
            <button className="button-primary mt-6 w-full" onClick={onRetry}>
              <RefreshCw size={16} aria-hidden="true" />
              {t('common.retry')}
            </button>
          ) : null}
        </div>
      </section>
    </main>
  )
}
