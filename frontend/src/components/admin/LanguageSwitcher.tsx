import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Language } from '../../lib/i18n'
import { AppSelect } from './AppSelect'

export function LanguageSwitcher() {
  const { t, i18n } = useTranslation()
  return (
    <label className="inline-flex min-h-11 items-center gap-2 text-sm font-semibold text-slate-700">
      <Languages size={17} aria-hidden="true" />
      <span className="sr-only">{t('language.label')}</span>
      <AppSelect
        className="min-h-9 whitespace-nowrap border-slate-200 py-1 text-xs"
        nowrapOptions
        value={i18n.language}
        label={t('language.label')}
        onValueChange={(value) => void i18n.changeLanguage(value as Language)}
        options={[
          { value: 'pt-BR', label: `🇧🇷 ${t('language.portuguese')}` },
          { value: 'en-US', label: `🇺🇸 ${t('language.english')}` },
        ]}
      />
    </label>
  )
}
