import { expect, it } from 'vitest'
import i18n from './i18n'
import ptBR from '../locales/pt-BR.json'
import enUS from '../locales/en-US.json'

function keys(value: unknown, prefix = ''): string[] {
  if (typeof value !== 'object' || value === null) return [prefix]
  return Object.entries(value).flatMap(([key, nested]) =>
    keys(nested, prefix ? `${prefix}.${key}` : key),
  )
}

it('keeps Portuguese and English catalogs aligned', () => {
  expect(keys(enUS).sort()).toEqual(keys(ptBR).sort())
})

it('switches labels and public errors by language', async () => {
  await i18n.changeLanguage('en-US')
  expect(i18n.t('nav.credentials')).toBe('API credentials')
  expect(i18n.t('errors.MFA_INVALID')).toBe(
    'Security code is invalid or has already been used.',
  )
  await i18n.changeLanguage('pt-BR')
  expect(i18n.t('nav.credentials')).toBe('Credenciais da API')
})
