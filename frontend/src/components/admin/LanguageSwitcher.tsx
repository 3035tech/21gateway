import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Language } from '../../lib/i18n'

export function LanguageSwitcher() {
  const { t, i18n } = useTranslation()
  return (
    <label className="inline-flex min-h-11 items-center gap-2 text-sm font-semibold text-slate-700">
      <Languages size={17} aria-hidden="true" />
      <span className="sr-only">{t('language.label')}</span>
      <select
        className="form-control min-h-9 border-slate-200 bg-white py-1 text-xs font-semibold"
        value={i18n.language}
        aria-label={t('language.label')}
        onChange={(event) =>
          void i18n.changeLanguage(event.target.value as Language)
        }
      >
        <option value="pt-BR">{t('language.portuguese')}</option>
        <option value="en-US">{t('language.english')}</option>
      </select>
    </label>
  )
}
