export const APP_TIME_ZONE = 'America/Sao_Paulo'
const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000

const appPartsFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: APP_TIME_ZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

const dateTimeParts = (date: Date) => {
  const parts = appPartsFormatter.formatToParts(date)
  return Object.fromEntries(parts.map(({ type, value }) => [type, value]))
}

export function formatInAppTimeZone(
  value: string | number | Date,
  locale: string,
) {
  return new Date(value).toLocaleString(locale, { timeZone: APP_TIME_ZONE })
}

export function toAppDateTimeInput(value?: string | null) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const { year, month, day, hour, minute } = dateTimeParts(date)
  return `${year}-${month}-${day}T${hour}:${minute}`
}

// datetime-local has no offset. Reject nonexistent or ambiguous wall times rather
// than silently choosing an instant during a future or historical DST transition.
export function fromAppDateTimeInput(value: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value)
  if (!match) return undefined
  const [, year, month, day, hour, minute] = match
  const wall = Date.UTC(+year, +month - 1, +day, +hour, +minute)
  const normalized = new Date(wall)
  if (
    normalized.getUTCFullYear() !== +year ||
    normalized.getUTCMonth() + 1 !== +month ||
    normalized.getUTCDate() !== +day ||
    normalized.getUTCHours() !== +hour ||
    normalized.getUTCMinutes() !== +minute
  )
    return undefined

  const candidates = new Set<number>()
  for (const sample of [
    wall - MILLISECONDS_PER_DAY,
    wall,
    wall + MILLISECONDS_PER_DAY,
  ]) {
    const parts = dateTimeParts(new Date(sample))
    const offset =
      Date.UTC(
        +parts.year,
        +parts.month - 1,
        +parts.day,
        +parts.hour,
        +parts.minute,
      ) - sample
    const instant = wall - offset
    if (toAppDateTimeInput(new Date(instant).toISOString()) === value)
      candidates.add(instant)
  }
  return candidates.size === 1 ? new Date([...candidates][0]) : undefined
}
