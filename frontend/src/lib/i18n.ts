import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import ptBR from '../locales/pt-BR.json'
import enUS from '../locales/en-US.json'

export const languages = ['pt-BR', 'en-US'] as const
export type Language = (typeof languages)[number]
const preferenceKey = '21gateway-language'

function initialLanguage(): Language {
  try {
    const stored = window.localStorage.getItem(preferenceKey)
    if (stored === 'pt-BR' || stored === 'en-US') return stored
  } catch {
    // Storage may be unavailable in a restricted browser context.
  }
  return window.navigator.language.toLowerCase().startsWith('en')
    ? 'en-US'
    : 'pt-BR'
}

void i18n.use(initReactI18next).init({
  resources: {
    'pt-BR': { translation: ptBR },
    'en-US': { translation: enUS },
  },
  lng: initialLanguage(),
  fallbackLng: 'pt-BR',
  supportedLngs: [...languages],
  interpolation: { escapeValue: false },
  initAsync: false,
})

document.documentElement.lang = i18n.language
i18n.on('languageChanged', (language) => {
  document.documentElement.lang = language
  try {
    window.localStorage.setItem(preferenceKey, language)
  } catch {
    // The selection still works for the current tab.
  }
})

export default i18n
