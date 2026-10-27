import { ArrowLeft } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export function NotFoundPage() {
  const { t } = useTranslation()
  return (
    <section className="mx-auto max-w-xl py-12 text-center">
      <p className="text-6xl font-bold text-brand-purple" aria-hidden="true">
        404
      </p>
      <h1 className="mt-5 text-2xl font-bold">{t('app.notFoundTitle')}</h1>
      <p className="mt-3 leading-7 text-slate-600">
        {t('app.notFoundDescription')}
      </p>
      <Link to="/" className="button-primary mt-7">
        <ArrowLeft size={16} aria-hidden="true" />
        {t('app.backToOverview')}
      </Link>
    </section>
  )
}
