import { describe, expect, it } from 'vitest'
import {
  formatInAppTimeZone,
  fromAppDateTimeInput,
  toAppDateTimeInput,
} from './timezone'

describe('São Paulo time zone', () => {
  it('formats an instant in São Paulo regardless of browser time zone', () => {
    expect(toAppDateTimeInput('2026-11-01T15:30:00Z')).toBe('2026-11-01T12:30')
    expect(formatInAppTimeZone('2026-11-01T15:30:00Z', 'pt-BR')).toContain(
      '12:30',
    )
  })

  it('converts a wall time back to its UTC instant', () => {
    expect(fromAppDateTimeInput('2026-11-01T12:30')?.toISOString()).toBe(
      '2026-11-01T15:30:00.000Z',
    )
  })

  it('rejects invalid and ambiguous historical daylight saving times', () => {
    expect(fromAppDateTimeInput('2026-02-30T12:30')).toBeUndefined()
    expect(fromAppDateTimeInput('2018-11-04T00:30')).toBeUndefined()
    expect(fromAppDateTimeInput('2018-02-17T23:30')).toBeUndefined()
  })
})
