import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
import i18n from '../lib/i18n'
void i18n.changeLanguage('pt-BR')
afterEach(cleanup)
